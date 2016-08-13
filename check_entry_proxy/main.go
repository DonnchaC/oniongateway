package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
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
	config = flag.String(
		"config",
		"",
		"config file with rules in YAML format",
	)
)

func main() {
	flag.Parse()
	var checker Checker
	if *config != "" {
		configData, err := ioutil.ReadFile(*config)
		if err != nil {
			fmt.Printf("Error reading config %s: %s\n", *config, err)
			os.Exit(1)
		}
		err = yaml.Unmarshal(configData, &checker)
		if err != nil {
			fmt.Printf("Error parsing config %s: %s\n", *config, err)
			os.Exit(1)
		}
	} else {
		// default values
		checker = Checker{
			Rules: []Rule{
				{"https://www.pasta.cf/mind-take-boyfriend/raw", "entry_proxy"},
			},
			RedirectRules: []string{
				"http://example.com/foo",
			},
		}
	}
	err := checker.CheckHost(*host, *proxyPort, *redirectPort)
	if err == nil {
		fmt.Printf("OK\n")
	} else {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
}
