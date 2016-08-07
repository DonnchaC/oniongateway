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

func errorf(message string, args ...interface{}) error {
	return errors.New(fmt.Sprintf(message, args...))
}

type Rule struct {
	Url          string
	ExpectedText string
}

type Checker struct {
	Rules []Rule
	Dial  func(network, addr string) (net.Conn, error)
}

func (c *Checker) makeHttpClient(address string) (http.Client, error) {
	transport := &http.Transport{
		Dial: func(network, _ string) (net.Conn, error) {
			if c.Dial != nil {
				return c.Dial(network, address)
			} else {
				return net.Dial(network, address)
			}
		},
	}
	return http.Client{Transport: transport}, nil
}

func (c *Checker) chooseRule() (Rule, error) {
	if len(c.Rules) == 0 {
		return Rule{}, errors.New("Set of rules to check is empty")
	}
	ruleIndex := rand.Intn(len(c.Rules))
	return c.Rules[ruleIndex], nil
}

func getResponse(rule Rule, client http.Client) (string, error) {
	response, err := client.Get(rule.Url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func checkResponse(rule Rule, body string) error {
	if !strings.Contains(body, rule.ExpectedText) {
		return errorf(
			"Responce body %q of URL %s does not contain expected text %q",
			body,
			rule.Url,
			rule.ExpectedText,
		)
	}
	return nil
}

func (c *Checker) CheckEntryProxy(address string) error {
	client, err := c.makeHttpClient(address)
	if err != nil {
		return errorf("Unable to create HTTP client: %s", err)
	}
	rule, err := c.chooseRule()
	if err != nil {
		return errorf("Unable to choose rule: %s", err)
	}
	body, err := getResponse(rule, client)
	if err != nil {
		return errorf("Unable to get download: %s", err)
	}
	if err := checkResponse(rule, body); err != nil {
		return errorf("Check failed: %s", err)
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
