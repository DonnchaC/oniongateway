package main

import (
	"net"
	"testing"

	"github.com/miekg/dns"
)

type mockResolver struct {
}

func (m *mockResolver) Resolve(_ string, _, _ uint16) ([]string, error) {
	return []string{"1.1.1.1", "2.2.2.2"}, nil
}

func (m *mockResolver) Start() {
}

func TestHandler(t *testing.T) {
	dnsListener, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create UDP listener: %s", err)
	}
	dnsAddr := dnsListener.LocalAddr().String()
	started := make(chan bool)
	server := &dns.Server{
		Net:        "udp",
		PacketConn: dnsListener,
		Handler:    &dnsHandler{resolver: &mockResolver{}},
		NotifyStartedFunc: func() {
			started <- true
		},
	}
	go server.ActivateAndServe()
	defer server.Shutdown()
	<-started
	// DNS client
	dnsMessage := &dns.Msg{}
	dnsMessage.SetQuestion("example.com.", dns.TypeA)
	dnsClient := &dns.Client{}
	in, _, err := dnsClient.Exchange(dnsMessage, dnsAddr)
	if err != nil {
		t.Fatalf("Failed to get DNS response from %s: %s", dnsAddr, err)
	}
	checkAnswer := func(i int, expected string) {
		if answer, ok := in.Answer[i].(*dns.A); ok {
			ipAddressStr := answer.A.String()
			if ipAddressStr != expected {
				t.Fatalf("DNS answered %q, exp. %q", ipAddressStr, expected)
			}
		} else {
			t.Fatalf("DNS response can't be converted to type A")
		}
	}
	checkAnswer(0, "1.1.1.1")
	checkAnswer(1, "2.2.2.2")
}
