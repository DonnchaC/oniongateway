package main

import (
	"errors"
	"fmt"
	"net"
	"regexp"
)

var onionTxtRe *regexp.Regexp

const SUBMATCH_OF_INTEREST = 2 // see onionTxtRe

func resolveToOnion(hostname string) (onion string, err error) {
	if onionTxtRe == nil {
		// FIXME: races?
		onionTxtRe, err = regexp.Compile("(^| )onion=([a-z0-9]{16}.onion)( |$)")
		if err != nil {
			return
		}
	}
	txts, err := net.LookupTXT(hostname)
	if err != nil {
		return
	}
	if len(txts) == 0 {
		err = errors.New(fmt.Sprintf("No TXT records for %s", hostname))
		return
	}
	for _, txt := range txts {
		match := onionTxtRe.FindStringSubmatch(txt)
		if match != nil {
			return match[SUBMATCH_OF_INTEREST], nil
		}
	}
	return "", errors.New(fmt.Sprintf("No suitable TXT records for %s", hostname))
}
