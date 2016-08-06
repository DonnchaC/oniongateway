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
	"net/url"
	"strconv"
	"sync"

	"github.com/polvi/sni"
	"golang.org/x/net/proxy"
)

var (
	proxyUrl   = flag.String("proxy", "socks5://127.0.0.1:9050", "Proxy URL")
	listenOn   = flag.String("listen-on", ":443", "Where to listen")
	onionPort  = flag.Int("onion-port", 443, "Port on onion site to use")
	bufferSize = flag.Int("buffer-size", 1024, "Proxy buffer size, bytes")
)

func connectToProxy(targetServer string) (net.Conn, error) {
	parsedUrl, err := url.Parse(*proxyUrl)
	if err != nil {
		return nil, err
	}
	dialer, err := proxy.FromURL(parsedUrl, proxy.Direct)
	if err != nil {
		return nil, err
	}
	connection, err := dialer.Dial("tcp", targetServer)
	return connection, err
}

func processRequest(clientConn net.Conn) {
	defer clientConn.Close()
	hostname, clientConn, err := sni.ServerNameFromConn(clientConn)
	if err != nil {
		log.Printf("Unable to get target server name from SNI: %s", err)
		return
	}
	onion, err := resolveToOnion(hostname)
	if err != nil {
		log.Printf("Unable to resolve %s using DNS TXT: %s", hostname, err)
		return
	}
	log.Printf("%s was resolved to %s", hostname, onion)
	targetServer := net.JoinHostPort(onion, strconv.Itoa(*onionPort))
	serverConn, err := connectToProxy(targetServer)
	if err != nil {
		log.Printf(
			"Unable to connect to %s through %s: %s\n",
			targetServer,
			*proxyUrl,
			err,
		)
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
	flag.Parse()
	listener, err := net.Listen("tcp", *listenOn)
	if err != nil {
		log.Fatalf("Unable to listen on %s: %s", *listenOn, err)
	}
	for {
		conn, err := listener.Accept()
		if err == nil {
			go processRequest(conn)
		} else {
			log.Printf("Unable to accept request: %s", err)
		}
	}
}
