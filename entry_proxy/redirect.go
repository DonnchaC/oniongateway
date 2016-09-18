package main

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"strings"
)

// NewRedirect returns redirecting HTTP server,
// which listens on `listenOn` and redirects to `redirectTo`.
// Entry proxy is expected to listen on `redirectTo`.
func NewRedirect(listenOn, redirectTo string) (*http.Server, error) {
	_, port, err := net.SplitHostPort(redirectTo)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
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
				path := newURL.Path
				if strings.HasPrefix(path, "/.well-known/acme-challenge/") {
					// https://github.com/DonnchaC/oniongateway/issues/18
					// proxy /.well-known/acme-challenge/ to HTTPS
					resp, err := client.Get(newURL.String())
					if err != nil {
						w.WriteHeader(http.StatusBadGateway)
						return
					}
					defer resp.Body.Close()
					w.WriteHeader(resp.StatusCode)
					io.Copy(w, resp.Body)
				} else {
					http.Redirect(
						w,
						r,
						newURL.String(),
						http.StatusMovedPermanently,
					)
				}
			},
		),
	}
	return server, nil
}
