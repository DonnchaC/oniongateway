package main

import (
	"sort"
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
		AnswerCount: 1,
	}
	// IPv4
	ips, err := resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv4", "example.com.")
	}
	if ips[0] != "127.0.0.1" {
		t.Fatalf("Wrong IPv4 address was returned: %s", ips)
	}
	// IPv6
	ips, err = resolver.Resolve("example.com.", dns.TypeAAAA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv6", "example.com.")
	}
	if ips[0] != "::1" {
		t.Fatalf("Wrong IPv6 address was returned: %s", ips)
	}
	// TXT
	txts, err := resolver.Resolve("pasta.cf.", dns.TypeTXT, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to get TXT record for %q", "pasta.cf.")
	}
	if txts[0] != "onion=pastagdsp33j7aoq.onion" {
		t.Fatalf("Wrong TXT response was returned: %q", txts)
	}
}

func TestFixedResolverAbsent(t *testing.T) {
	resolver := &FixedResolver{}
	// IPv4
	ips, err := resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err == nil {
		t.Fatalf("IPv4 request expected to fail returned %s", ips)
	}
	// IPv6
	ips, err = resolver.Resolve("example.com.", dns.TypeAAAA, dns.ClassINET)
	if err == nil {
		t.Fatalf("IPv6 request expected to fail returned %s", ips)
	}
	// TXT
	txts, err := resolver.Resolve("pasta.cf.", dns.TypeTXT, dns.ClassINET)
	if err == nil {
		t.Fatalf("TXT request expected to fail returned %s", txts)
	}
}

func TestFixedResolverMulti(t *testing.T) {
	resolver := &FixedResolver{
		IPv4Proxies: []string{
			"127.0.0.1",
			"127.0.0.2",
		},
		AnswerCount: 2,
	}
	// IPv4
	ips, err := resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv4", "example.com.")
	}
	sort.Strings(ips)
	if ips[0] != "127.0.0.1" || ips[1] != "127.0.0.2" {
		t.Fatalf("Wrong IPv4 address was returned: %s", ips)
	}
}
