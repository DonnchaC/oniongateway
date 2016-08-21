package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/embed"
	"github.com/miekg/dns"
	"github.com/phayes/freeport"
	"golang.org/x/net/context"
)

const timeout = 5 * time.Second

func makeEtcdServer() (
	server *embed.Etcd,
	client *clientv3.Client,
	closer func(),
	err error,
) {
	// choose ports and data directory
	peerPort := freeport.GetPort()
	clientPort := freeport.GetPort()
	peerURLStr := fmt.Sprintf("http://127.0.0.1:%d", peerPort)
	peerURL, err := url.Parse(peerURLStr)
	if err != nil {
		err = fmt.Errorf("Failed to parse peer URL %q: %s", peerURLStr, err)
		return
	}
	clientURLStr := fmt.Sprintf("http://127.0.0.1:%d", clientPort)
	clientURL, err := url.Parse(clientURLStr)
	if err != nil {
		err = fmt.Errorf("Failed to parse client URL %q: %s", clientURLStr, err)
		return
	}
	tmpDir, err := ioutil.TempDir("", "oniongateway-nameserver-etcd-test")
	if err != nil {
		err = fmt.Errorf("Failed to create temp dir: %s", err)
		return
	}
	// create server
	cfg := embed.NewConfig()
	cfg.Dir = tmpDir
	cfg.APUrls = []url.URL{*peerURL}
	cfg.LPUrls = []url.URL{*peerURL}
	cfg.ACUrls = []url.URL{*clientURL}
	cfg.LCUrls = []url.URL{*clientURL}
	cfg.InitialCluster = cfg.InitialClusterFromName(cfg.Name)
	server, err = embed.StartEtcd(cfg)
	if err != nil {
		err = fmt.Errorf("Failed to start etcd server: %s", err)
		os.RemoveAll(tmpDir)
		return
	}
	time.Sleep(time.Second) // https://github.com/coreos/etcd/issues/6029#issuecomment-234664336
	// create client
	client, err = clientv3.New(clientv3.Config{
		Endpoints:   []string{fmt.Sprintf("127.0.0.1:%d", clientPort)},
		DialTimeout: timeout,
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		err = fmt.Errorf("Failed to create etcd client: %s", err)
		return
	}
	closer = func() {
		client.Close()
		server.Close()
		os.RemoveAll(tmpDir)
	}
	return
}

func populateDatabase(client *clientv3.Client) error {
	kv := clientv3.NewKV(client)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	_, err := kv.Put(ctx, "/ipv4/127.0.0.1", "not used")
	if err != nil {
		return fmt.Errorf("Failed to put key /ipv4/127.0.0.1: %s", err)
	}
	_, err = kv.Put(ctx, "/ipv6/::1", "not used")
	if err != nil {
		return fmt.Errorf("Failed to put key /ipv6/::1: %s", err)
	}
	_, err = kv.Put(ctx, "/domain2onion/pasta.cf.", "pastagdsp33j7aoq.onion")
	if err != nil {
		return fmt.Errorf("Failed to put key /domain2onion/pasta.cf.: %s", err)
	}
	return nil
}

func TestEtcdResolver(t *testing.T) {
	_, client, closer, err := makeEtcdServer()
	if err != nil {
		t.Fatalf("Failed to create etcd server and client: %s", err)
	}
	defer closer()
	if err = populateDatabase(client); err != nil {
		t.Fatalf("Failed to populate etcd with example data: %s", err)
	}
	resolver := &EtcdResolver{
		Client:  client,
		Timeout: timeout,
	}
	// IPv4
	address, err := resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv4", "example.com.")
	}
	if address != "127.0.0.1" {
		t.Fatalf("Wrong IPv4 address was returned: %q", address)
	}
	// IPv6
	address, err = resolver.Resolve("example.com.", dns.TypeAAAA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv6", "example.com.")
	}
	if address != "::1" {
		t.Fatalf("Wrong IPv6 address was returned: %q", address)
	}
	// TXT
	txt, err := resolver.Resolve("pasta.cf.", dns.TypeTXT, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to get TXT record for %q", "pasta.cf.")
	}
	if txt != "onion=pastagdsp33j7aoq.onion" {
		t.Fatalf("Wrong TXT response was returned: %q", txt)
	}
}

func TestEmptyResolverAbsent(t *testing.T) {
	_, client, closer, err := makeEtcdServer()
	if err != nil {
		t.Fatalf("Failed to create etcd server and client: %s", err)
	}
	defer closer()
	resolver := &EtcdResolver{
		Client:  client,
		Timeout: timeout,
	}
	// IPv4
	address, err := resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err == nil {
		t.Fatalf("IPv4 request expected to fail returned %q", address)
	}
	// IPv6
	address, err = resolver.Resolve("example.com.", dns.TypeAAAA, dns.ClassINET)
	if err == nil {
		t.Fatalf("IPv6 request expected to fail returned %q", address)
	}
	// TXT
	txt, err := resolver.Resolve("pasta.cf.", dns.TypeTXT, dns.ClassINET)
	if err == nil {
		t.Fatalf("TXT request expected to fail returned %q", txt)
	}
}
