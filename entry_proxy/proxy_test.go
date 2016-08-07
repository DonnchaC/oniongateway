package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"sync"
	"testing"
)

// MortalService can be killed at any time.
type MortalService struct {
	network            string
	address            string
	connectionCallback func(net.Conn) error

	conns     []net.Conn
	stopping  bool
	listener  net.Listener
	waitGroup *sync.WaitGroup
}

// NewMortalService creates a new MortalService
func NewMortalService(network, address string, connectionCallback func(net.Conn) error) *MortalService {
	l := MortalService{
		network:            network,
		address:            address,
		connectionCallback: connectionCallback,

		conns:     make([]net.Conn, 0, 10),
		stopping:  false,
		waitGroup: &sync.WaitGroup{},
	}
	return &l
}

// Start the MortalService
func (l *MortalService) Start() error {
	var err error
	log.Printf("starting listener service %s:%s", l.network, l.address)
	l.listener, err = net.Listen(l.network, l.address)
	if err != nil {
		return err
	}
	l.waitGroup.Add(1)
	go l.acceptLoop()
	return nil
}

// Stop will kill our listener and all it's connections
func (l *MortalService) Stop() {
	log.Printf("stopping listener service %s:%s", l.network, l.address)
	l.stopping = true
	if l.listener != nil {
		l.listener.Close()
	}
	l.waitGroup.Wait()
}

func (l *MortalService) acceptLoop() {
	defer l.waitGroup.Done()
	defer func() {
		log.Printf("acceptLoop stopping for listener service %s:%s", l.network, l.address)
		for i, conn := range l.conns {
			if conn != nil {
				log.Printf("Closing connection #%d", i)
				conn.Close()
			}
		}
	}()
	defer l.listener.Close()

	for {
		conn, err := l.listener.Accept()
		if err != nil {
			log.Printf("MortalService connection accept failure: %s\n", err)
			if l.stopping {
				return
			}
			continue
		}

		l.conns = append(l.conns, conn)
		go l.handleConnection(conn, len(l.conns)-1)
	}
}

func (l *MortalService) handleConnection(conn net.Conn, id int) error {
	defer func() {
		log.Printf("Closing connection #%d", id)
		conn.Close()
		l.conns[id] = nil
	}()

	log.Printf("Starting connection #%d", id)
	if err := l.connectionCallback(conn); err != nil {
		return err
	}
	return nil
}

type AccumulatingListener struct {
	net, address    string
	buffer          bytes.Buffer
	mortalService   *MortalService
	hasProtocolInfo bool
	hasAuthenticate bool
	Received        chan bool
}

func NewAccumulatingListener(net, address string) *AccumulatingListener {
	l := AccumulatingListener{
		net:             net,
		address:         address,
		hasProtocolInfo: true,
		hasAuthenticate: true,
		Received:        make(chan bool, 0),
	}
	return &l
}

func (a *AccumulatingListener) Start() {
	a.mortalService = NewMortalService(a.net, a.address, a.SessionWorker)
	err := a.mortalService.Start()
	if err != nil {
		panic(err)
	}
}

func (a *AccumulatingListener) Stop() {
	fmt.Println("AccumulatingListener STOP")
	a.mortalService.Stop()
}

func (a *AccumulatingListener) SessionWorker(conn net.Conn) error {
	connReader := bufio.NewReader(conn)
	for {
		line, err := connReader.ReadBytes('\n')
		if err != nil {
			fmt.Println("AccumulatingListener read error:", err)
		}
		fmt.Printf("receive line: %x\n", line)
		a.buffer.WriteString(string(line))
		a.Received <- true
	}
	return nil
}

type MockSNIParser struct{}

func (m MockSNIParser) ServerNameFromConn(clientConn net.Conn) (string, net.Conn, error) {
	return "Horse25519", clientConn, nil
}

type MockTxtResolver struct{}

func (o MockTxtResolver) LookupTXT(hostname string) ([]string, error) {
	return []string{"onion=abcdef1234567654.onion"}, nil
}

type MockProxyDialer struct {
	proxyNet  string
	proxyAddr string
}

func NewMockProxyDialer(proxyNet, proxyAddr string) *MockProxyDialer {
	d := MockProxyDialer{
		proxyNet:  proxyNet,
		proxyAddr: proxyAddr,
	}
	return &d
}

func (d *MockProxyDialer) Dial(targetServer string) (net.Conn, error) {
	conn, err := net.Dial(d.proxyNet, d.proxyAddr)
	return conn, err
}

func TestTLSProxy(t *testing.T) {
	fakeTorNet := "tcp"
	fakeTorListener := NewAccumulatingListener(fakeTorNet, "127.0.0.1:0")
	fakeTorListener.Start()
	fakeTorAddr := fakeTorListener.mortalService.listener.Addr().String()

	proxyNet := "tcp"
	proxy := NewTLSProxy(443, fakeTorNet, fakeTorAddr)
	proxy.sniParser = MockSNIParser{}
	proxy.resolver.txtResolver = MockTxtResolver{}
	proxy.dialer = NewMockProxyDialer(fakeTorNet, fakeTorAddr)
	proxy.Listen(proxyNet, "127.0.0.1:0")
	go proxy.Start()
	proxyAddr := proxy.Addr().String()

	conn, err := net.Dial(proxyNet, proxyAddr)
	if err != nil {
		t.Errorf("failed to connect to proxy: %s", err)
		t.Fail()
	}

	want := "meow\n"
	fmt.Printf("sending string: %x\n", want)
	n, err := conn.Write([]byte(want))
	if err != nil {
		t.Errorf("failed to write: %s", err)
		t.Fail()
	} else {
		if n != len(want) {
			t.Error("failed to write full length of buffer")
			t.Fail()
		}
	}

	<-fakeTorListener.Received
	if fakeTorListener.buffer.String() != want {
		t.Errorf("got:%x but expected:%x", fakeTorListener.buffer.String(), want)
		t.Fail()
	}
	fakeTorListener.Stop()
}
