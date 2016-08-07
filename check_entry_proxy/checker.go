package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strings"
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
	return http.Client{Transport: transport}, nil
}

func (c *Checker) chooseRule() (Rule, error) {
	if len(c.Rules) == 0 {
		return Rule{}, fmt.Errorf("Set of rules to check is empty")
	}
	ruleIndex := rand.Intn(len(c.Rules))
	return c.Rules[ruleIndex], nil
}

func getResponse(rule Rule, client http.Client) (string, error) {
	response, err := client.Get(rule.URL)
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
	body, err := getResponse(rule, client)
	if err != nil {
		return fmt.Errorf("Unable to get download: %s", err)
	}
	if err := checkResponse(rule, body); err != nil {
		return fmt.Errorf("Check failed: %s", err)
	}
	return nil
}
