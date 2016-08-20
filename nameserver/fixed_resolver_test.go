package main

import (
	"testing"

	"github.com/miekg/dns"
)

func TestFixedResolver(t *testing.T) {
	resolver := &FixedResolver{
		IPv4Proxies: []string{
			"127.0.0.1",
		},
		IPv6Proxies: []string{
			"::1",
		},
		Domain2Onion: map[string]string{
			"pasta.cf.": "pastagdsp33j7aoq.onion",
		},
	}
	// IPv4
	address, err := resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv4", "example.com.")
	}
	if address != "127.0.0.1" {
		t.Fatalf("Wrong IPv4 address was returned: %q", address)
	}
	// IPv6
	address, err = resolver.Resolve("example.com.", dns.TypeAAAA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv6", "example.com.")
	}
	if address != "::1" {
		t.Fatalf("Wrong IPv6 address was returned: %q", address)
	}
	// TXT
	txt, err := resolver.Resolve("pasta.cf.", dns.TypeTXT, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to get TXT record for %q", "pasta.cf.")
	}
	if txt != "onion=pastagdsp33j7aoq.onion" {
		t.Fatalf("Wrong TXT response was returned: %q", txt)
	}
}

func TestFixedResolverAbsent(t *testing.T) {
	resolver := &FixedResolver{}
	// IPv4
	address, err := resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err == nil {
		t.Fatalf("IPv4 request expected to fail returned %q", address)
	}
	// IPv6
	address, err = resolver.Resolve("example.com.", dns.TypeAAAA, dns.ClassINET)
	if err == nil {
		t.Fatalf("IPv6 request expected to fail returned %q", address)
	}
	// TXT
	txt, err := resolver.Resolve("pasta.cf.", dns.TypeTXT, dns.ClassINET)
	if err == nil {
		t.Fatalf("TXT request expected to fail returned %q", txt)
	}
}
