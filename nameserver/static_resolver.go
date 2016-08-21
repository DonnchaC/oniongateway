package main

import (
	"fmt"

	"github.com/dgryski/go-randsample"
	"github.com/miekg/dns"
)

// StaticResolver resolves DNS requests from static in-memory config
type StaticResolver struct {
	IPv4Proxies  []string
	IPv6Proxies  []string
	Domain2Onion map[string]string
	AnswerCount  int
}

// Resolve fetches result value for DNS request from memory
func (r *StaticResolver) Resolve(
	domain string,
	qtype, qclass uint16,
) (
	[]string,
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
			return nil, fmt.Errorf("TXT request of unknown domain: %q", domain)
		}
		txt := fmt.Sprintf("onion=%s", onion)
		return []string{txt}, nil
	} else {
		return nil, fmt.Errorf("Unknown question type: %d", qtype)
	}
	n := len(proxies)
	if n == 0 {
		return nil, fmt.Errorf("No proxies for question of type %d", qtype)
	}
	k := r.AnswerCount
	if n < k {
		k = n
	}
	var result []string
	for _, i := range randsample.Sample(n, k) {
		address := proxies[i]
		result = append(result, address)
	}
	return result, nil
}
