package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

const anyProxy = "1.2.3.4:5678"

func makeHttpServer(observedText string) *httptest.Server {
	// TODO NewTLSServer
	return httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, observedText)
			},
		),
	)
}

func makeMockChecker(expectedText, observedText string) *Checker {
	ts := makeHttpServer(observedText)
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

func makeRealChecker(
	expectedText, observedText string,
) (
	checker *Checker,
	proxy string,
) {
	ts := makeHttpServer(observedText)
	checker = &Checker{
		Rules: []Rule{
			{"http://example.com/", expectedText},
		},
	}
	proxy = ts.Listener.Addr().String()
	return
}

func TestCheckEntryProxyReal(t *testing.T) {
	checker, proxy := makeRealChecker("test passed", "test passed")
	err := checker.CheckEntryProxy(proxy)
	if err != nil {
		t.Fatalf("Always passing test failed: %s", err)
	}
}

func TestCheckEntryProxyRealFail(t *testing.T) {
	checker, proxy := makeRealChecker("expected", "observed")
	err := checker.CheckEntryProxy(proxy)
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
