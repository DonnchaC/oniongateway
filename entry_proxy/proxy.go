package main

import (
	"io"
	"log"
	"net"
	"strconv"
	"sync"

	"github.com/polvi/sni"
	"golang.org/x/net/proxy"
)

type SNIParser interface {
	ServerNameFromConn(c net.Conn) (string, net.Conn, error)
}

type RealSNIParser struct{}

func (t RealSNIParser) ServerNameFromConn(clientConn net.Conn) (string, net.Conn, error) {
	hostname, clientConn, err := sni.ServerNameFromConn(clientConn)
	return hostname, clientConn, err
}

type ProxyDialer interface {
	Dial(string) (net.Conn, error)
}

type SocksDialer struct {
	proxyNet  string
	proxyAddr string
	auth      proxy.Auth
}

func NewSocksDialer(proxyNet, proxyAddr string) *SocksDialer {
	s := SocksDialer{
		proxyNet:  proxyNet,
		proxyAddr: proxyAddr,
		auth: proxy.Auth{
			User:     "",
			Password: "",
		},
	}
	return &s
}

func (t *SocksDialer) Dial(targetServer string) (net.Conn, error) {
	dialer, err := proxy.SOCKS5(t.proxyNet, t.proxyAddr, &t.auth, proxy.Direct)
	if err != nil {
		return nil, err
	}
	connection, err := dialer.Dial("tcp", targetServer)
	return connection, err
}

type TLSProxy struct {
	conn      net.Conn
	onionPort int

	proxyNet  string
	proxyAddr string

	sniParser SNIParser
	resolver  HostToOnionResolver
	dialer    ProxyDialer
	listener  net.Listener
}

func NewTLSProxy(onionPort int, proxyNet, proxyAddr string) *TLSProxy {
	t := TLSProxy{
		onionPort: onionPort,
		proxyNet:  proxyNet,
		proxyAddr: proxyAddr,
		sniParser: RealSNIParser{},
		resolver:  NewHostToOnionResolver(),
		dialer:    NewSocksDialer(proxyNet, proxyAddr),
	}
	return &t
}

func (t *TLSProxy) Listen(listenNet, listenAddr string) {
	listener, err := net.Listen(listenNet, listenAddr)
	t.listener = listener
	if err != nil {
		log.Fatalf("Unable to listen on %s %s: %s", listenNet, listenAddr, err)
	}
}

func (t *TLSProxy) Start() {
	for {
		conn, err := t.listener.Accept()
		if err == nil {
			go t.ProcessRequest(conn)
		} else {
			log.Printf("Unable to accept request: %s", err)
		}
	}
}

func (t *TLSProxy) Addr() net.Addr {
	return t.listener.Addr()
}

func (t *TLSProxy) ProcessRequest(clientConn net.Conn) {
	defer clientConn.Close()
	hostname, clientConn, err := t.sniParser.ServerNameFromConn(clientConn)
	if err != nil {
		log.Printf("Unable to get target server name from SNI: %s", err)
		return
	}
	onion, err := t.resolver.ResolveToOnion(hostname)
	if err != nil {
		log.Printf("Unable to resolve %s using DNS TXT: %s", hostname, err)
		return
	}
	log.Printf("%s was resolved to %s", hostname, onion)
	targetServer := net.JoinHostPort(onion, strconv.Itoa(t.onionPort))
	serverConn, err := t.dialer.Dial(targetServer)

	if err != nil {
		log.Printf("Unable to connect to %s through %s %s: %s\n", targetServer, t.proxyNet, t.proxyAddr, err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	copyLoop := func(dst, src net.Conn) {
		defer wg.Done()
		defer dst.Close()
		io.Copy(dst, src)
	}
	go copyLoop(clientConn, serverConn)
	go copyLoop(serverConn, clientConn)
	wg.Wait()
}
