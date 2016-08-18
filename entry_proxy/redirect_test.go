package main

import (
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/DonnchaC/oniongateway/util"
)

func TestNewRedirect(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()
	if err != nil {
		t.Fatalf("Failed to create a listener: %s", err)
	}
	server, err := NewRedirect(listener.Addr().String(), ":443")
	if err != nil {
		t.Fatalf("Failed to create redirecting HTTP server: %s", err)
	}
	go server.Serve(listener)
	// check what the server returns
	hostAndPort := listener.Addr().String()
	host, _, err := net.SplitHostPort(hostAndPort)
	if err != nil {
		t.Fatalf("Failed to extract host from %s: %s", hostAndPort, err)
	}
	client := http.Client{CheckRedirect: util.IgnoreRedirect}
	theURL := &url.URL{
		Scheme: "http",
		Host:   hostAndPort,
	}
	response, err := client.Get(theURL.String())
	if !util.IsRedirectError(err) {
		t.Fatalf("Expected to get redirect error, but got %s", err)
	}
	if response.StatusCode != http.StatusMovedPermanently {
		t.Fatalf(
			"Wrong HTTP status returned: %d instead of %d",
			response.StatusCode,
			http.StatusMovedPermanently,
		)
	}
	nextURL, err := response.Location()
	if err != nil {
		t.Fatalf("Failed to get location of redirect: %s", err)
	}
	if nextURL.Scheme != "https" {
		t.Fatalf("Expected scheme https, but got %s", nextURL.Scheme)
	}
	if nextURL.Host != host {
		t.Fatalf("Expected host %q, but got %q", host, nextURL.Host)
	}
}
