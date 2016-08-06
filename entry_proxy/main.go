package main

/*  Entry Proxy for Onion Gateway

See also:

  * https://github.com/DonnchaC/oniongateway/blob/master/docs/design.rst#32-entry-proxy
  * https://gist.github.com/Yawning/bac58e08a05fc378a8cc (SOCKS5 client, Tor)
  * https://habrahabr.ru/post/142527/ (TCP proxy)
*/

import (
    "flag"
    "log"
    "net"
    "net/url"
    "strconv"

    "github.com/polvi/sni"
    "golang.org/x/net/proxy"
)

var (
    proxyUrl = flag.String("proxy", "socks5://127.0.0.1:9050", "Proxy URL")
    listenOn = flag.String("listen-on", "0.0.0.0:443", "Where to listen")
    onionPort = flag.Int("onion-port", 443, "Port on onion site to use")
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

func netCopy(from, to net.Conn, finished chan<- struct{}) {
    defer func() {
        finished<-struct{}{}
    }()
    buffer := make([]byte, *bufferSize)
    for {
        bytesRead, err := from.Read(buffer)
        if err != nil {
            log.Printf("Finished reading: %s", err)
            break
        }
        _, err = to.Write(buffer[:bytesRead])
        if err != nil {
            log.Printf("Finished writting: %s", err)
            break
        }
    }
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
    defer serverConn.Close()
    finished := make(chan struct{})
    go netCopy(clientConn, serverConn, finished)
    go netCopy(serverConn, clientConn, finished)
    <-finished
    <-finished
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
