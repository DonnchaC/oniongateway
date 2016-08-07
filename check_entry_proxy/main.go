package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strings"
)

type Rule struct {
	Url          string
	ExpectedText string
}

type Checker struct {
	Rules []Rule
	Dial  func(network, addr string) (net.Conn, error)
}

func (c *Checker) CheckEntryProxy(address string) error {
	// make HTTP client
	transport := &http.Transport{
		Dial: func(network, _ string) (net.Conn, error) {
			if c.Dial != nil {
				return c.Dial(network, address)
			} else {
				return net.Dial(network, address)
			}
		},
	}
	client := &http.Client{Transport: transport}
	// choose URL
	if len(c.Rules) == 0 {
		return errors.New("Set of rules to check is empty")
	}
	ruleIndex := rand.Intn(len(c.Rules))
	rule := c.Rules[ruleIndex]
	// request and check response body
	response, err := client.Get(rule.Url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	if !strings.Contains(string(body), rule.ExpectedText) {
		return errors.New(
			fmt.Sprintf(
				"Responce body %q of URL %s proxied through %s "+
					"does not contain expected text %q",
				response,
				rule.Url,
				address,
				rule.ExpectedText,
			),
		)
	}
	return nil
}

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
