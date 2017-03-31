package main

/*  Entry Proxy for Onion Gateway

See also:

  * https://github.com/DonnchaC/oniongateway/blob/master/docs/design.rst#32-entry-proxy
  * https://gist.github.com/Yawning/bac58e08a05fc378a8cc (SOCKS5 client, Tor)
  * https://habrahabr.ru/post/142527/ (TCP proxy)
*/

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

func main() {
	var (
		proxyNet = flag.String(
			"proxyNet",
			"tcp",
			"Proxy network type",
		)
		proxyAddr = flag.String(
			"proxyAddr",
			"127.0.0.1:9050",
			"Proxy address",
		)
		entryProxy = flag.String(
			"entry-proxy",
			":443",
			"host:port of entry proxy",
		)
		httpRedirect = flag.String(
			"http-redirect",
			":80",
			"host:port of redirecting HTTP server ('' to disable)",
		)
		onionPort = flag.Int(
			"onion-port",
			443,
			"Port on onion site to use",
		)
		hostToOnionTable = flag.String(
			"host-to-onion",
			"",
			"Yaml file with host->onion map, disables DNS based resolver",
		)
		parentHost = flag.String(
			"parent-host",
			"",
			"Read onion address in subdomain of specified domain, disables DNS based resolver",
		)
	)

	flag.Parse()

	// Check if Tor2Web mode is enabled.
	// Tor does not provide access to clearnet sites in Tor2Web mode.
	dialer := NewSocksDialer(*proxyNet, *proxyAddr)
	site4test := "check.torproject.org:443"
	if _, err := dialer.Dial(site4test); err == nil {
		log.Printf(
			"Warning: Tor can access %s, probably Tor2Web mode is off\n",
			site4test,
		)
	}

	if *httpRedirect != "" {
		redirectingServer, err := NewRedirect(*httpRedirect, *entryProxy)
		if err != nil {
			fmt.Printf("Unable to create redirecting HTTP server: %s\n", err)
			os.Exit(1)
		}
		go redirectingServer.ListenAndServe()
	}

	var resolver HostToOnionResolver
	if *hostToOnionTable != "" {
		log.Printf("Using host2onion map from file %s", *hostToOnionTable)
		configData, err := ioutil.ReadFile(*hostToOnionTable)
		if err != nil {
			log.Fatalf("Error reading %s: %s", *hostToOnionTable, err)
		}
		var staticResolver StaticResolver
		err = yaml.Unmarshal(configData, &staticResolver)
		if err != nil {
			log.Fatalf("Error parsing %s: %s", *hostToOnionTable, err)
		}
		resolver = &staticResolver
	} else if *parentHost != "" {
		log.Printf("Using domain %s as parent host", *parentHost)
		resolver = NewSubdomainResolver(*parentHost)
	} else {
		resolver = NewDnsHostToOnionResolver()
	}

	proxy := NewTLSProxy(*onionPort, *proxyNet, *proxyAddr, resolver)
	proxy.Listen("tcp", *entryProxy)

	log.Printf("starting entry proxy")
	proxy.Start()
}
