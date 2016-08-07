package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	entryProxy = flag.String("entry-proxy", ":443", "Entry proxy to test")
)

func main() {
	flag.Parse()
	checker := &Checker{
		// TODO: move to flag
		Rules: []Rule{
			{"https://www.pasta.cf/mind-take-boyfriend/raw", "entry_proxy"},
		},
	}
	err := checker.CheckEntryProxy(*entryProxy)
	if err == nil {
		fmt.Printf("OK\n")
	} else {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}
