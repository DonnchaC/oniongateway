package main

import (
	"fmt"

	"github.com/miekg/dns"
)

type StaticResolver struct {
	Host2Onion map[string]string
}

func (r *StaticResolver) ResolveToOnion(host string) (string, error) {
	onion, ok := r.Host2Onion[dns.Fqdn(host)]
	if !ok {
		return "", fmt.Errorf("No key %q in host->onion map", host)
	}
	return onion, nil
}
