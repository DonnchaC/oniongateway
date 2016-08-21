package main

import (
	"fmt"
	"log"

	"github.com/miekg/dns"
)

// Resolver fetches result value for DNS request
type Resolver interface {
	Resolve(domain string, qtype, qclass uint16) ([]string, error)
}

type dnsHandler struct {
	resolver Resolver
}

func (h *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	log.Printf("DNS message: %s", r)
	if r.Opcode == dns.OpcodeQuery {
		m := new(dns.Msg)
		m.SetReply(r)
		for _, q := range m.Question {
			answers, err := h.resolver.Resolve(q.Name, q.Qtype, q.Qclass)
			if err == nil && len(answers) >= 1 {
				for _, answer := range answers {
					recordString := fmt.Sprintf(
						"%s IN %s %s",
						q.Name,
						dns.TypeToString[q.Qtype],
						answer,
					)
					record, err := dns.NewRR(recordString)
					if err == nil {
						m.Answer = append(m.Answer, record)
					} else {
						log.Printf(
							"Unable to parse answer record %s: %s",
							recordString,
							err,
						)
					}
				}
			} else {
				log.Printf("Unable to get proxy: %s", err)
			}
		}
		w.WriteMsg(m)
	} else {
		log.Printf("Opcode %d ignored", r.Opcode)
	}
}
