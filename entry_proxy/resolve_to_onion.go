package main

import (
	"errors"
	"fmt"
	"net"
	"regexp"
)

type OnionResolver interface {
	ResolveToOnion(string) (string, error)
}

type RealOnionResolver struct {
	regex *regexp.Regexp
}

func NewRealOnionResolver() *RealOnionResolver {
	var err error
	o := RealOnionResolver{}
	o.regex, err = regexp.Compile("(^| )onion=([a-z0-9]{16}.onion)( |$)")
	if err != nil {
		panic("wtf: failed to compile regex")
	}
	return &o
}

func (o *RealOnionResolver) ResolveToOnion(hostname string) (onion string, err error) {
	txts, err := net.LookupTXT(hostname)
	if err != nil {
		return
	}
	if len(txts) == 0 {
		err = errors.New(fmt.Sprintf("No TXT records for %s", hostname))
		return
	}
	for _, txt := range txts {
		match := o.regex.FindStringSubmatch(txt)
		if match != nil {
			return match[2], nil // the submatch we are interested in
		}
	}
	return "", errors.New(fmt.Sprintf("No suitable TXT records for %s", hostname))
}
