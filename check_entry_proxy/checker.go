package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
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
	Rules         []Rule
	RedirectRules []string
	Dial          func(network, addr string) (net.Conn, error)
	RandIntn      func(n int) int
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
	ruleIndex := c.RandIntn(len(c.Rules))
	return c.Rules[ruleIndex], nil
}

func (c *Checker) chooseRedirectURL() (*url.URL, error) {
	if len(c.RedirectRules) == 0 {
		return nil, fmt.Errorf("Set of redirect URLs to check is empty")
	}
	ruleIndex := c.RandIntn(len(c.RedirectRules))
	rawurl := c.RedirectRules[ruleIndex]
	return url.Parse(rawurl)
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

func getLocation(url string, client http.Client) (*url.URL, error) {
	_, response, err := getResponse(url, client)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusMovedPermanently {
		return nil, fmt.Errorf(
			"Wrong HTTP status returned: %d instead of %d",
			response.StatusCode,
			http.StatusMovedPermanently,
		)
	}
	return response.Location()
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

// CheckRedirect checks if redirect works
func (c *Checker) CheckRedirect(redirect, proxy string) error {
	client, err := c.makeHTTPClient(redirect)
	if err != nil {
		return fmt.Errorf("Unable to create HTTP client: %s", err)
	}
	testingURL, err := c.chooseRedirectURL()
	if err != nil {
		return err
	}
	expectedHost := testingURL.Host
	_, proxyPort, err := net.SplitHostPort(proxy)
	if err != nil {
		return fmt.Errorf("Failed to get proxy port from %q: %s", proxy, err)
	}
	if proxyPort != "443" {
		expectedHost += ":" + proxyPort
	}
	url, err := getLocation(testingURL.String(), client)
	if err != nil {
		return fmt.Errorf("Unable to download: %s", err)
	}
	if url.Scheme != "https" {
		return fmt.Errorf("redirects to non-https URL %q", url.String())
	}
	if url.Host != expectedHost {
		return fmt.Errorf("redirect host is %q, expected %q", url.Host, expectedHost)
	}
	if url.Path != testingURL.Path {
		return fmt.Errorf("redirect path is %q, expected %q", url.Path, testingURL.Path)
	}
	return nil
}

// CheckHost checks both redirect and proxy
func (c *Checker) CheckHost(host string, proxyPort, redirectPort int) error {
	proxyLocation := net.JoinHostPort(host, fmt.Sprintf("%d", proxyPort))
	redirectLocation := net.JoinHostPort(host, fmt.Sprintf("%d", redirectPort))
	err := c.CheckEntryProxy(proxyLocation)
	if err != nil {
		return fmt.Errorf("Unable to access onion via entry proxy: %s", err)
	}
	err = c.CheckRedirect(redirectLocation, proxyLocation)
	if err != nil {
		return fmt.Errorf(
			"http://%s does not redirect to https://%s: %s\n",
			redirectLocation,
			proxyLocation,
			err,
		)
	}
	return nil
}
