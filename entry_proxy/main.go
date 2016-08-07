package main

/*  Entry Proxy for Onion Gateway

See also:

  * https://github.com/DonnchaC/oniongateway/blob/master/docs/design.rst#32-entry-proxy
  * https://gist.github.com/Yawning/bac58e08a05fc378a8cc (SOCKS5 client, Tor)
  * https://habrahabr.ru/post/142527/ (TCP proxy)
*/

import (
	"flag"
)

func main() {
	var (
		proxyNet  = flag.String("proxyNet", "tcp", "Proxy network type")
		proxyAddr = flag.String("proxyAddr", "127.0.0.1:9050", "Proxy address")
		listenOn  = flag.String("listen-on", ":443", "Where to listen")
		onionPort = flag.Int("onion-port", 443, "Port on onion site to use")
	)

	flag.Parse()

	proxy := NewTLSProxy(*onionPort, *proxyNet, *proxyAddr)
	proxy.Start("tcp", *listenOn)
}
