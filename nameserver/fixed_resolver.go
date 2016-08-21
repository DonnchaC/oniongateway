package main

import (
	"fmt"
	"math/rand"

	"github.com/miekg/dns"
)

// FixedResolver resolves DNS requests from static in-memory config
type FixedResolver struct {
	IPv4Proxies  []string
	IPv6Proxies  []string
	Domain2Onion map[string]string
}

// Resolve fetches result value for DNS request from memory
func (r *FixedResolver) Resolve(
	domain string,
	qtype, qclass uint16,
) (
	string,
	error,
) {
	var proxies []string
	if qtype == dns.TypeA {
		proxies = r.IPv4Proxies
	} else if qtype == dns.TypeAAAA {
		proxies = r.IPv6Proxies
	} else if qtype == dns.TypeTXT {
		onion, ok := r.Domain2Onion[domain]
		if !ok {
			return "", fmt.Errorf("TXT request of unknown domain: %q", domain)
		}
		txt := fmt.Sprintf("onion=%s", onion)
		return txt, nil
	} else {
		return "", fmt.Errorf("Unknown question type: %d", qtype)
	}
	if len(proxies) == 0 {
		return "", fmt.Errorf("No proxies for question of type %d", qtype)
	}
	i := rand.Intn(len(proxies))
	return proxies[i], nil
}
