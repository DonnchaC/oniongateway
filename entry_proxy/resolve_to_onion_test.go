package main

import (
	"errors"
	"testing"
)

type EmptyMockTxtResolver struct{}

func (o EmptyMockTxtResolver) LookupTXT(hostname string) ([]string, error) {
	return []string{}, nil
}

func TestEmptyMockTxtResolver(t *testing.T) {
	resolver := NewDnsHostToOnionResolver()
	resolver.txtResolver = EmptyMockTxtResolver{}
	_, err := resolver.ResolveToOnion("example.com")
	if err == nil {
		t.Fatal("Empty TXT resolver works, but it must not")
	}
}

type NoOnionsMockTxtResolver struct{}

func (o NoOnionsMockTxtResolver) LookupTXT(hostname string) ([]string, error) {
	return []string{"foo=bar", "bar=foo"}, nil
}

func TestNoOnionsMockTxtResolver(t *testing.T) {
	resolver := NewDnsHostToOnionResolver()
	resolver.txtResolver = NoOnionsMockTxtResolver{}
	_, err := resolver.ResolveToOnion("example.com")
	if err == nil {
		t.Fatal("No-onions TXT resolver works, but it must not")
	}
}

type ThrowingMockTxtResolver struct{}

func (o ThrowingMockTxtResolver) LookupTXT(hostname string) ([]string, error) {
	return []string{}, errors.New("I always throw")
}

func TestThrowingMockTxtResolver(t *testing.T) {
	resolver := NewDnsHostToOnionResolver()
	resolver.txtResolver = ThrowingMockTxtResolver{}
	_, err := resolver.ResolveToOnion("example.com")
	if err == nil {
		t.Fatal("Throwing TXT resolver works, but it must not")
	}
}
