package main

import (
	"fmt"
	"math/rand"
	"path"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

// EtcdResolver resolves DNS requests from etcd
type EtcdResolver struct {
	Client  *clientv3.Client
	Timeout time.Duration
}

// Resolve fetches result value for DNS request from etcd
func (r *EtcdResolver) Resolve(
	domain string,
	qtype, qclass uint16,
) (
	string,
	error,
) {
	kv := clientv3.NewKV(r.Client)
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()
	if qtype == dns.TypeA || qtype == dns.TypeAAAA {
		// TODO: watch on updates of the lists
		var keyPrefix string
		if qtype == dns.TypeA {
			keyPrefix = "/ipv4/"
		} else if qtype == dns.TypeAAAA {
			keyPrefix = "/ipv6/"
		}
		resp, err := kv.Get(ctx, keyPrefix, clientv3.WithPrefix())
		if err != nil {
			return "", fmt.Errorf("etcd GET error qtype=%d: %s", qtype, err)
		}
		n := len(resp.Kvs)
		if n == 0 {
			return "", fmt.Errorf("No proxies for question of type %d", qtype)
		}
		i := rand.Intn(n)
		key := string(resp.Kvs[i].Key)
		address := path.Base(key)
		return address, nil
	} else if qtype == dns.TypeTXT {
		key := fmt.Sprintf("/domain2onion/%s", domain)
		resp, err := kv.Get(ctx, key)
		if err != nil {
			return "", fmt.Errorf("etcd GET error qtype=%d: %s", qtype, err)
		}
		if len(resp.Kvs) == 0 {
			return "", fmt.Errorf("TXT request of unknown domain: %q", domain)
		}
		onion := string(resp.Kvs[0].Value)
		txt := fmt.Sprintf("onion=%s", onion)
		return txt, nil
	} else {
		return "", fmt.Errorf("Unknown question type: %d", qtype)
	}
}
