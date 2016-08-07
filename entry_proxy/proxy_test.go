package main

import (
	"net"
	"testing"
)

type MockSNIParser struct{}

func (m MockSNIParser) ServerNameFromConn(clientConn net.Conn) (string, net.Conn, error) {
	return "Horse25519", clientConn, nil
}

type MockOnionResolver struct{}

func (o *MockOnionResolver) ResolveToOnion(hostname string) (string, error) {
	return "MockOnionHaHa", nil
}

func TestTLSProxy(t *testing.T) {
	proxyNet := "tcp"
	proxyAddr := "127.0.0.1:34627"

	proxy := NewTLSProxy(443, proxyNet, proxyAddr)
	proxy.sniParser = MockSNIParser{}
	proxy.resolver = &MockOnionResolver{}
	//go proxy.Start("tcp", ":44333")

	//t.Error("expected failure")
	//t.Fail()
}
