package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

const anyProxy = "1.2.3.4:5678"

func makeMockChecker(expectedText, observedText string) *Checker {
	// TODO NewTLSServer
	ts := httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, observedText)
			},
		),
	)
	return &Checker{
		Rules: []Rule{
			{"http://example.com/", expectedText},
		},
		Dial: func(network, addr string) (net.Conn, error) {
			return net.Dial(network, ts.Listener.Addr().String())
		},
	}
}

func TestCheckEntryProxy(t *testing.T) {
	checker := makeMockChecker("test passed", "test passed")
	err := checker.CheckEntryProxy(anyProxy)
	if err != nil {
		t.Fatalf("Always passing test failed: %s", err)
	}
}

func TestCheckEntryProxyFail(t *testing.T) {
	checker := makeMockChecker("expected", "observed")
	err := checker.CheckEntryProxy(anyProxy)
	if err == nil {
		t.Fatalf("checker did not fail with bad entry_proxy: %s", err)
	}
}

func TestCheckEntryProxyEmptyRules(t *testing.T) {
	checker := &Checker{}
	err := checker.CheckEntryProxy(anyProxy)
	if err == nil {
		t.Fatal("checker did not fail with empty list of rules")
	}
}
