package main

/*  Entry Proxy for Onion Gateway

See also:

  * https://github.com/DonnchaC/oniongateway/blob/master/docs/design.rst#32-entry-proxy
  * https://gist.github.com/Yawning/bac58e08a05fc378a8cc (SOCKS5 client, Tor)
  * https://habrahabr.ru/post/142527/ (TCP proxy)
*/

import (
	"flag"
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

type TLSProxy struct {
	conn      net.Conn
	onionPort int

	proxyNet  string
	proxyAddr string

	sniParser SNIParser
	resolver  OnionResolver
}

func NewTLSProxy(onionPort int, proxyNet, proxyAddr string) *TLSProxy {
	t := TLSProxy{
		onionPort: onionPort,
		proxyNet:  proxyNet,
		proxyAddr: proxyAddr,
		sniParser: RealSNIParser{},
		resolver:  NewRealOnionResolver(),
	}
	return &t
}

func (t *TLSProxy) Dial(targetServer string) (net.Conn, error) {
	auth := proxy.Auth{
		User:     "",
		Password: "",
	}
	dialer, err := proxy.SOCKS5(t.proxyNet, t.proxyAddr, &auth, proxy.Direct)
	if err != nil {
		return nil, err
	}
	connection, err := dialer.Dial("tcp", targetServer)
	return connection, err
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
	serverConn, err := t.Dial(targetServer)

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

func main() {

	var (
		proxyNet  = flag.String("proxyNet", "tcp", "Proxy network type")
		proxyAddr = flag.String("proxyAddr", "127.0.0.1:9050", "Proxy address")
		listenOn  = flag.String("listen-on", ":443", "Where to listen")
		onionPort = flag.Int("onion-port", 443, "Port on onion site to use")
	)

	flag.Parse()

	listener, err := net.Listen("tcp", *listenOn)
	if err != nil {
		log.Fatalf("Unable to listen on %s: %s", *listenOn, err)
	}

	proxy := NewTLSProxy(*onionPort, *proxyNet, *proxyAddr)
	for {
		conn, err := listener.Accept()
		if err == nil {
			go proxy.ProcessRequest(conn)
		} else {
			log.Printf("Unable to accept request: %s", err)
		}
	}
}
