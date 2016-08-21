package main

import (
	"flag"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/miekg/dns"
	"gopkg.in/yaml.v2"
)

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
