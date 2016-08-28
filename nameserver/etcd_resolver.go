package main

import (
	"fmt"
	"log"
	"path"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/mvcc/mvccpb"
	"github.com/miekg/dns"
	"golang.org/x/net/context"
)

// EtcdResolver resolves DNS requests from etcd
type EtcdResolver struct {
	Client      *clientv3.Client
	Timeout     time.Duration
	AnswerCount int

	ipResolver      StaticResolver
	ipResolverMutex sync.RWMutex
}

// Start gets current lists of addresses and starts watching for updates
func (r *EtcdResolver) Start() {
	if r.AnswerCount == 0 {
		panic("EtcdResolver: set AnswerCount")
	}
	r.ipResolver.AnswerCount = r.AnswerCount
	r.ipResolver.Start()
	go r.watch()
}

func (r *EtcdResolver) watch() {
	log.Printf("Running etcd watcher of nameserver...")
	go r.watchFor(&r.ipResolver.IPv4Proxies, "/ipv4/")
	go r.watchFor(&r.ipResolver.IPv6Proxies, "/ipv6/")
}

func (r *EtcdResolver) watchFor(
	addresses *[]string,
	prefix string,
) {
	address2index := make(map[string]int, len(*addresses))
	for index, address := range *addresses {
		address2index[address] = index
	}
	watcher := clientv3.NewWatcher(r.Client)
	ctx := context.Background()
	const historyStart = 1
	ipv4Chan := watcher.Watch(
		ctx,
		prefix,
		clientv3.WithPrefix(),
		clientv3.WithRev(historyStart),
	)
	for {
		resp := <-ipv4Chan
		if resp.Canceled {
			break
		}
		for _, event := range resp.Events {
			r.processEvent(addresses, address2index, event)
		}
	}
}

func (r *EtcdResolver) processEvent(
	addresses *[]string,
	address2index map[string]int,
	event *clientv3.Event,
) {
	r.ipResolverMutex.Lock()
	defer r.ipResolverMutex.Unlock()
	key := string(event.Kv.Key)
	address := path.Base(key)
	if event.Type == mvccpb.PUT {
		if event.IsCreate() {
			if _, has := address2index[address]; has {
				panic(fmt.Sprintf("Existing address %s added", address))
			}
			address2index[address] = len(*addresses)
			*addresses = append(*addresses, address)
			log.Printf("Address %s was added", key)
		}
	} else if event.Type == mvccpb.DELETE {
		index, has := address2index[address]
		if !has {
			panic(fmt.Sprintf("Absent address %s removed", address))
		}
		// move last element to `index`
		lastIndex := len(*addresses) - 1
		lastAddress := (*addresses)[lastIndex]
		address2index[lastAddress] = index
		(*addresses)[index] = lastAddress
		*addresses = (*addresses)[:lastIndex]
		delete(address2index, address)
		log.Printf("Address %s was removed", key)
	} else {
		panic(fmt.Sprintf("Unknown event type %d", event.Type))
	}
}

// Resolve fetches result value for DNS request from etcd
func (r *EtcdResolver) Resolve(
	domain string,
	qtype, qclass uint16,
) (
	[]string,
	error,
) {
	kv := clientv3.NewKV(r.Client)
	ctx, cancel := context.WithTimeout(context.Background(), r.Timeout)
	defer cancel()
	if qtype == dns.TypeA || qtype == dns.TypeAAAA {
		r.ipResolverMutex.RLock()
		defer r.ipResolverMutex.RUnlock()
		return r.ipResolver.Resolve(domain, qtype, qclass)
	} else if qtype == dns.TypeTXT {
		key := fmt.Sprintf("/domain2onion/%s", domain)
		resp, err := kv.Get(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("etcd GET error qtype=%d: %s", qtype, err)
		}
		if len(resp.Kvs) == 0 {
			return nil, fmt.Errorf("TXT request of unknown domain: %q", domain)
		}
		onion := string(resp.Kvs[0].Value)
		txt := fmt.Sprintf("onion=%s", onion)
		return []string{txt}, nil
	} else {
		return nil, fmt.Errorf("Unknown question type: %d", qtype)
	}
}
