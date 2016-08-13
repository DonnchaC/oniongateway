package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	host = flag.String(
		"host",
		"127.0.0.1",
		"Host of entry proxy to test",
	)
	proxyPort = flag.Int(
		"proxy-port",
		443,
		"HTTPS port of entry proxy",
	)
	redirectPort = flag.Int(
		"redirect-port",
		80,
		"HTTP port redirecting to entry proxy",
	)
)

func main() {
	flag.Parse()
	checker := &Checker{
		// TODO: move to flag
		Rules: []Rule{
			{"https://www.pasta.cf/mind-take-boyfriend/raw", "entry_proxy"},
		},
		RedirectRules: []string{
			"http://example.com/foo",
		},
	}
	err := checker.CheckHost(*host, *proxyPort, *redirectPort)
	if err == nil {
		fmt.Printf("OK\n")
	} else {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}
