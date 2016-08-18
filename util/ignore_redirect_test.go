package util

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestIgnoreRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "http://example.com/", http.StatusMovedPermanently)
		},
	))
	defer server.Close()
	client := http.Client{CheckRedirect: IgnoreRedirect}
	theURL := &url.URL{
		Scheme: "http",
		Host:   server.Listener.Addr().String(),
	}
	_, err := client.Get(theURL.String())
	if !IsRedirectError(err) {
		t.Fatalf("Expected to get redirect error, but got %s", err)
	}
}
