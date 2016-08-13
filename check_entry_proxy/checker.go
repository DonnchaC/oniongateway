package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strings"

	"github.com/DonnchaC/oniongateway/util"
)

// Rule for checker: URL and text expected to be found in response
type Rule struct {
	URL          string
	ExpectedText string
}

// Checker for entry_proxy
type Checker struct {
	Rules []Rule
	Dial  func(network, addr string) (net.Conn, error)
}

func (c *Checker) makeHTTPClient(address string) (http.Client, error) {
	transport := &http.Transport{
		Dial: func(network, _ string) (net.Conn, error) {
			if c.Dial != nil {
				return c.Dial(network, address)
			}
			return net.Dial(network, address)
		},
	}
	client := http.Client{
		Transport: transport,
		CheckRedirect: util.IgnoreRedirect,
	}
	return client, nil
}

func (c *Checker) chooseRule() (Rule, error) {
	if len(c.Rules) == 0 {
		return Rule{}, fmt.Errorf("Set of rules to check is empty")
	}
	ruleIndex := rand.Intn(len(c.Rules))
	return c.Rules[ruleIndex], nil
}

func getResponse(theURL string, client http.Client) (string, *http.Response, error) {
	response, err := client.Get(theURL)
	if err != nil {
		if util.IsRedirectError(err) {
			return "", response, nil
		}
		return "", response, err
	}
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", response, err
	}
	return string(body), response, nil
}

func checkResponse(rule Rule, body string) error {
	if !strings.Contains(body, rule.ExpectedText) {
		return fmt.Errorf(
			"Responce body %q of URL %s does not contain expected text %q",
			body,
			rule.URL,
			rule.ExpectedText,
		)
	}
	return nil
}

// CheckEntryProxy checks entry_proxy instance
func (c *Checker) CheckEntryProxy(address string) error {
	client, err := c.makeHTTPClient(address)
	if err != nil {
		return fmt.Errorf("Unable to create HTTP client: %s", err)
	}
	rule, err := c.chooseRule()
	if err != nil {
		return fmt.Errorf("Unable to choose rule: %s", err)
	}
	body, _, err := getResponse(rule.URL, client)
	if err != nil {
		return fmt.Errorf("Unable to download: %s", err)
	}
	if err := checkResponse(rule, body); err != nil {
		return fmt.Errorf("Check failed: %s", err)
	}
	return nil
}
