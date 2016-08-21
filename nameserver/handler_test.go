package main

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

type mockResolver struct {
}

func (m *mockResolver) Resolve(_ string, _, _ uint16) (string, error) {
	return "1.1.1.1", nil
}

func TestHandler(t *testing.T) {
	dnsListener, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create UDP listener: %s", err)
	}
	dnsAddr := dnsListener.LocalAddr().String()
	server := &dns.Server{
		Net:        "udp",
		PacketConn: dnsListener,
		Handler:    &dnsHandler{resolver: &mockResolver{}},
	}
	go server.ActivateAndServe()
	defer server.Shutdown()
	time.Sleep(time.Second) // race: time to start DNS server
	// DNS client
	dnsMessage := &dns.Msg{}
	dnsMessage.SetQuestion("example.com.", dns.TypeA)
	dnsClient := &dns.Client{}
	in, _, err := dnsClient.Exchange(dnsMessage, dnsAddr)
	if err != nil {
		t.Fatalf("Failed to get DNS response from %s: %s", dnsAddr, err)
	}
	if firstAnswer, ok := in.Answer[0].(*dns.A); ok {
		ipAddressStr := firstAnswer.A.String()
		if ipAddressStr != "1.1.1.1" {
			t.Fatalf("DNS answered %q, exp. %q", ipAddressStr, "1.1.1.1")
		}
	} else {
		t.Fatalf("DNS response can't be converted to type A")
	}
}
