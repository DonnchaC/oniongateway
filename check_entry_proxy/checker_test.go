package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

const anyProxy = "1.2.3.4:5678"

func makeHTTPServer(observedText string) *httptest.Server {
	// TODO NewTLSServer
	return httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprintln(w, observedText)
			},
		),
	)
}

func makeMockChecker(
	expectedText, observedText, url string,
) (
	checker *Checker,
	server *httptest.Server,
) {
	server = makeHTTPServer(observedText)
	checker = &Checker{
		Rules: []Rule{
			{url, expectedText},
		},
		Dial: func(network, addr string) (net.Conn, error) {
			return net.Dial(network, server.Listener.Addr().String())
		},
	}
	return
}

func TestCheckEntryProxy(t *testing.T) {
	checker, server := makeMockChecker(
		"test passed",
		"test passed",
		"http://example.com/",
	)
	defer server.Close()
	err := checker.CheckEntryProxy(anyProxy)
	if err != nil {
		t.Fatalf("Always passing test failed: %s", err)
	}
}

func TestCheckEntryProxyFailNotContains(t *testing.T) {
	checker, server := makeMockChecker(
		"expected",
		"observed",
		"http://example.com/",
	)
	defer server.Close()
	err := checker.CheckEntryProxy(anyProxy)
	if err == nil {
		t.Fatalf("checker did not fail with bad entry_proxy: %s", err)
	}
}

func TestCheckEntryProxyFailDownloading(t *testing.T) {
	checker, server := makeMockChecker(
		"expected",
		"observed",
		"https://example.com/",
	)
	defer server.Close()
	err := checker.CheckEntryProxy(anyProxy)
	if err == nil {
		t.Fatalf("checker did not fail with bad entry_proxy: %s", err)
	}
}

func makeRealChecker(
	expectedText, observedText string,
) (
	checker *Checker,
	server *httptest.Server,
	proxy string,
) {
	server = makeHTTPServer(observedText)
	checker = &Checker{
		Rules: []Rule{
			{"http://example.com/", expectedText},
		},
	}
	proxy = server.Listener.Addr().String()
	return
}

func TestCheckEntryProxyReal(t *testing.T) {
	checker, server, proxy := makeRealChecker("test passed", "test passed")
	defer server.Close()
	err := checker.CheckEntryProxy(proxy)
	if err != nil {
		t.Fatalf("Always passing test failed: %s", err)
	}
}

func TestCheckEntryProxyRealFail(t *testing.T) {
	checker, server, proxy := makeRealChecker("expected", "observed")
	defer server.Close()
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

func makeRedirectingHTTPServer(code, port int, overrideHost string) *httptest.Server {
	return httptest.NewServer(
		http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				newURL := *r.URL
				newURL.Scheme = "https"
				host, _, err := net.SplitHostPort(r.Host)
				if err != nil {
					host = r.Host
				}
				if port != 443 {
					portStr := fmt.Sprintf("%d", port)
					host = net.JoinHostPort(host, portStr)
				}
				if overrideHost != "" {
					host = overrideHost
				}
				newURL.Host = host
				http.Redirect(w, r, newURL.String(), code)
			},
		),
	)
}

func getHostPort(location string) (host string, port int) {
	host, portStr, err := net.SplitHostPort(location)
	if err != nil {
		panic(fmt.Sprintf(
			"Failed to parse %s to host:port: %s",
			location,
			err,
		))
	}
	fmt.Sscanf(portStr, "%d", &port)
	return
}

func TestCheckRedirect(t *testing.T) {
	server := makeRedirectingHTTPServer(http.StatusMovedPermanently, 443, "")
	checker := &Checker{
		RedirectRules: []string{
			"http://example.com/foo",
		},
	}
	redirectLocation := server.Listener.Addr().String()
	host, _ := getHostPort(redirectLocation)
	proxyLocation := net.JoinHostPort(host, "443")
	err := checker.CheckRedirect(redirectLocation, proxyLocation)
	if err != nil {
		t.Fatalf("Always passing test failed: %s", err)
	}
}

func TestCheckRedirectNonStandardPort(t *testing.T) {
	server := makeRedirectingHTTPServer(http.StatusMovedPermanently, 1443, "")
	checker := &Checker{
		RedirectRules: []string{
			"http://example.com/foo",
		},
	}
	redirectLocation := server.Listener.Addr().String()
	host, _ := getHostPort(redirectLocation)
	proxyLocation := net.JoinHostPort(host, "1443")
	err := checker.CheckRedirect(redirectLocation, proxyLocation)
	if err != nil {
		t.Fatalf("Always passing test failed: %s", err)
	}
}

func TestCheckRedirectOtherHost(t *testing.T) {
	server := makeRedirectingHTTPServer(http.StatusMovedPermanently, 443, "hacker.host")
	checker := &Checker{
		RedirectRules: []string{
			"http://example.com/foo",
		},
	}
	redirectLocation := server.Listener.Addr().String()
	host, _ := getHostPort(redirectLocation)
	proxyLocation := net.JoinHostPort(host, "443")
	err := checker.CheckRedirect(redirectLocation, proxyLocation)
	if err == nil {
		t.Fatalf("Expected to fail, but not failed")
	}
}

func TestCheckRedirectOtherPort(t *testing.T) {
	server := makeRedirectingHTTPServer(http.StatusMovedPermanently, 443, "")
	checker := &Checker{
		RedirectRules: []string{
			"http://example.com/foo",
		},
	}
	redirectLocation := server.Listener.Addr().String()
	host, _ := getHostPort(redirectLocation)
	proxyLocation := net.JoinHostPort(host, "1443")
	err := checker.CheckRedirect(redirectLocation, proxyLocation)
	if err == nil {
		t.Fatalf("Expected to fail, but not failed")
	}
}

func TestCheckRedirectOtherHTTPStatus(t *testing.T) {
	server := makeRedirectingHTTPServer(http.StatusFound, 443, "")
	checker := &Checker{
		RedirectRules: []string{
			"http://example.com/foo",
		},
	}
	redirectLocation := server.Listener.Addr().String()
	host, _ := getHostPort(redirectLocation)
	proxyLocation := net.JoinHostPort(host, "443")
	err := checker.CheckRedirect(redirectLocation, proxyLocation)
	if err == nil {
		t.Fatalf("Expected to fail, but not failed")
	}
}

func TestCheckRedirectEmptyRules(t *testing.T) {
	server := makeRedirectingHTTPServer(http.StatusMovedPermanently, 443, "")
	checker := &Checker{}
	redirectLocation := server.Listener.Addr().String()
	host, _ := getHostPort(redirectLocation)
	proxyLocation := net.JoinHostPort(host, "443")
	err := checker.CheckRedirect(redirectLocation, proxyLocation)
	if err == nil {
		t.Fatalf("Expected to fail, but not failed")
	}
}

func TestCheckHost(t *testing.T) {
	checker, server, _ := makeRealChecker("test passed", "test passed")
	checker.RedirectRules = []string{
		"http://example.com/foo",
	}
	defer server.Close()
	proxyHost, proxyPort := getHostPort(server.Listener.Addr().String())
	server = makeRedirectingHTTPServer(http.StatusMovedPermanently, proxyPort, "")
	defer server.Close()
	redirectHost, redirectPort := getHostPort(server.Listener.Addr().String())
	if redirectHost != proxyHost {
		t.Fatalf("host(proxy)=%q, host(redirect)=%q", proxyHost, redirectHost)
	}
	err := checker.CheckHost(redirectHost, proxyPort, redirectPort)
	if err != nil {
		t.Fatalf("Always passing test failed: %s", err)
	}
}
