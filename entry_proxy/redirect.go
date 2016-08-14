package main

import (
	"net"
	"net/http"
)

// NewRedirect returns redirecting HTTP server,
// which listens on `listenOn` and redirects to `redirectTo`.
// Entry proxy is expected to listen on `redirectTo`.
func NewRedirect(listenOn, redirectTo string) (*http.Server, error) {
	_, port, err := net.SplitHostPort(redirectTo)
	if err != nil {
		return nil, err
	}
	server := &http.Server{
		Addr: listenOn,
		Handler: http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				newURL := *r.URL
				newURL.Scheme = "https"
				host, _, err := net.SplitHostPort(r.Host)
				if err != nil {
					host = r.Host
				}
				if port != "443" {
					host = net.JoinHostPort(host, port)
				}
				newURL.Host = host
				http.Redirect(w, r, newURL.String(), http.StatusMovedPermanently)
			},
		),
	}
	return server, nil
}
