package main

import (
	"fmt"
	"net"
	"regexp"
)

type TxtResolver interface {
	LookupTXT(string) ([]string, error)
}

type RealTxtResolver struct{}

func (r RealTxtResolver) LookupTXT(hostname string) ([]string, error) {
	txts, err := net.LookupTXT(hostname)
	return txts, err
}

type HostToOnionResolver struct {
	regex       *regexp.Regexp
	txtResolver TxtResolver
}

func NewHostToOnionResolver() HostToOnionResolver {
	var err error
	o := HostToOnionResolver{
		txtResolver: RealTxtResolver{},
	}
	o.regex, err = regexp.Compile("(^| )onion=([a-z0-9]{16}.onion)( |$)")
	if err != nil {
		panic("wtf: failed to compile regex")
	}
	return o
}

func (o *HostToOnionResolver) ResolveToOnion(hostname string) (onion string, err error) {
	txts, err := o.txtResolver.LookupTXT(hostname)
	if err != nil {
		return
	}
	if len(txts) == 0 {
		err = fmt.Errorf("No TXT records for %s", hostname)
		return
	}
	for _, txt := range txts {
		match := o.regex.FindStringSubmatch(txt)
		if match != nil {
			return match[2], nil // the submatch we are interested in
		}
	}
	return "", fmt.Errorf("No suitable TXT records for %s", hostname)
}
