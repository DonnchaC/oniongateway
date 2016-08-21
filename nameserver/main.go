package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
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

var (
	fixedConfig = flag.String(
		"fixed-config",
		"",
		"config file with rules in YAML format",
	)
	etcdEndpoints = flag.String(
		"etcd-endpoints",
		"",
		"comma-separated list of etcd endpoints",
	)
	etcdTimeout = flag.Duration(
		"etcd-timeout",
		5*time.Second,
		"Timeout used in etcd client",
	)
	listenAddr = flag.String(
		"listen-addr",
		":53",
		"Address of DNS server to create",
	)
	listenNet = flag.String(
		"listen-net",
		"udp",
		"Network of DNS server to create",
	)
)

func main() {
	flag.Parse()
	var handler dnsHandler
	if *fixedConfig != "" && *etcdEndpoints != "" {
		log.Fatalf("Provide one of -fixed-config and -etcd-endpoints")
	} else if *fixedConfig != "" {
		configData, err := ioutil.ReadFile(*fixedConfig)
		if err != nil {
			log.Fatalf("Error reading config %s: %s\n", *fixedConfig, err)
		}
		var fixedResolver FixedResolver
		err = yaml.Unmarshal(configData, &fixedResolver)
		if err != nil {
			log.Fatalf("Error parsing config %s: %s\n", *fixedConfig, err)
		}
		handler.resolver = &fixedResolver
	} else if *etcdEndpoints != "" {
		endpoints := strings.Split(*etcdEndpoints, ",")
		client, err := clientv3.New(clientv3.Config{
			Endpoints:   endpoints,
			DialTimeout: *etcdTimeout,
		})
		if err != nil {
			log.Fatalf("Error creating etcd client: %s", err)
		}
		handler.resolver = &EtcdResolver{
			Client:  client,
			Timeout: *etcdTimeout,
		}
	} else {
		log.Fatalf("Provide one of -fixed-config and -etcd-endpoints")
	}
	server := &dns.Server{
		Addr:    *listenAddr,
		Net:     *listenNet,
		Handler: &handler,
	}
	err := server.ListenAndServe()
	defer server.Shutdown()
	if err != nil {
		log.Fatalf("Failed to setup the udp server: %s", err)
	}
}
