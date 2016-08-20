package main

import (
	"fmt"
	"log"

	"github.com/miekg/dns"
)

type Resolver interface {
	Resolve(domain string, qtype, qclass uint16) (string, error)
}

type dnsHandler struct {
	resolver Resolver
}

func (h *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	log.Printf("question: %s", r.Question)
	if r.Opcode == dns.OpcodeQuery {
		m := new(dns.Msg)
		m.SetReply(r)
		for _, q := range m.Question {
			proxy, err := h.resolver.Resolve(q.Name, q.Qtype, q.Qclass)
			if err == nil {
				recordString := fmt.Sprintf(
					"%s IN %s %s",
					q.Name,
					dns.TypeToString[q.Qtype],
					proxy,
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
			} else {
				log.Printf("Unable to get proxy: %s", err)
			}
		}
		w.WriteMsg(m)
	} else {
		log.Printf("Opcode %d ignored", r.Opcode)
	}
}

func main() {
	server := &dns.Server{
		Addr: ":4253",
		Net:  "udp",
		Handler: &dnsHandler{
			resolver: &FixedResolver{
				IPv4Proxies: []string{
					"127.0.0.1",
					"127.0.0.2",
				},
				IPv6Proxies: []string{
					"::1",
				},
				Domain2Onion: map[string]string{
					"pasta.cf.": "pastagdsp33j7aoq.onion",
				},
			},
		},
	}
	err := server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		log.Fatalf("Failed to setup the udp server: %s", err)
	}
}
