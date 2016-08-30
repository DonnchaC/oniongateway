package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"sort"
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

func changeDatabase(
	action func(ctx context.Context, key string, value string) error,
	kvs map[string]string,
) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	for key, value := range kvs {
		if err := action(ctx, key, value); err != nil {
			return fmt.Errorf("Failed to put (%q, %q): %s", key, value, err)
		}
	}
	return nil
}

func populateDatabase(client *clientv3.Client, kvs map[string]string) error {
	kv := clientv3.NewKV(client)
	action := func(ctx context.Context, key string, value string) error {
		_, err := kv.Put(ctx, key, value)
		return err
	}
	return changeDatabase(action, kvs)
}

func deleteKeys(client *clientv3.Client, kvs map[string]string) error {
	kv := clientv3.NewKV(client)
	action := func(ctx context.Context, key string, _ string) error {
		_, err := kv.Delete(ctx, key)
		return err
	}
	return changeDatabase(action, kvs)
}

func TestEtcdResolver(t *testing.T) {
	_, client, closer, err := makeEtcdServer()
	if err != nil {
		t.Fatalf("Failed to create etcd server and client: %s", err)
	}
	defer closer()
	data := map[string]string{
		"/ipv4/127.0.0.1":         "value is not used",
		"/ipv6/::1":               "value is not used",
		"/domain2onion/pasta.cf.": "pastagdsp33j7aoq.onion",
	}
	if err = populateDatabase(client, data); err != nil {
		t.Fatalf("Failed to populate etcd with example data: %s", err)
	}
	changed := make(chan bool)
	resolver := &EtcdResolver{
		Client:      client,
		Timeout:     timeout,
		AnswerCount: 1,
		NotifyChangedFunc: func(added bool, key string) {
			changed <- true
		},
	}
	resolver.Start()
	<-changed
	<-changed
	// IPv4
	ips, err := resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv4", "example.com.")
	}
	if len(ips) < 1 || ips[0] != "127.0.0.1" {
		t.Fatalf("Wrong IPv4 address was returned: %s", ips)
	}
	// IPv6
	ips, err = resolver.Resolve("example.com.", dns.TypeAAAA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv6", "example.com.")
	}
	if len(ips) < 1 || ips[0] != "::1" {
		t.Fatalf("Wrong IPv6 address was returned: %s", ips)
	}
	// TXT
	txts, err := resolver.Resolve("pasta.cf.", dns.TypeTXT, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to get TXT record for %q", "pasta.cf.")
	}
	if txts[0] != "onion=pastagdsp33j7aoq.onion" {
		t.Fatalf("Wrong TXT response was returned: %s", txts)
	}
}

func TestEmptyResolverAbsent(t *testing.T) {
	_, client, closer, err := makeEtcdServer()
	if err != nil {
		t.Fatalf("Failed to create etcd server and client: %s", err)
	}
	defer closer()
	resolver := &EtcdResolver{
		Client:      client,
		Timeout:     timeout,
		AnswerCount: 1,
	}
	resolver.Start()
	// IPv4
	ips, err := resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err == nil {
		t.Fatalf("IPv4 request expected to fail returned %s", ips)
	}
	// IPv6
	ips, err = resolver.Resolve("example.com.", dns.TypeAAAA, dns.ClassINET)
	if err == nil {
		t.Fatalf("IPv6 request expected to fail returned %s", ips)
	}
	// TXT
	txts, err := resolver.Resolve("pasta.cf.", dns.TypeTXT, dns.ClassINET)
	if err == nil {
		t.Fatalf("TXT request expected to fail returned %s", txts)
	}
}

func TestEtcdResolverMulti(t *testing.T) {
	_, client, closer, err := makeEtcdServer()
	if err != nil {
		t.Fatalf("Failed to create etcd server and client: %s", err)
	}
	defer closer()
	data := map[string]string{
		"/ipv4/127.0.0.1": "value is not used",
		"/ipv4/127.0.0.2": "value is not used",
	}
	if err = populateDatabase(client, data); err != nil {
		t.Fatalf("Failed to populate etcd with example data: %s", err)
	}
	changed := make(chan bool)
	resolver := &EtcdResolver{
		Client:      client,
		Timeout:     timeout,
		AnswerCount: 2,
		NotifyChangedFunc: func(added bool, key string) {
			changed <- true
		},
	}
	resolver.Start()
	<-changed
	<-changed
	// IPv4
	ips, err := resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv4: %s", "example.com.", err)
	}
	sort.Strings(ips)
	if len(ips) < 2 || ips[0] != "127.0.0.1" || ips[1] != "127.0.0.2" {
		t.Fatalf("Wrong IPv4 address was returned: %s", ips)
	}
}

func TestEtcdResolverChange(t *testing.T) {
	_, client, closer, err := makeEtcdServer()
	if err != nil {
		t.Fatalf("Failed to create etcd server and client: %s", err)
	}
	defer closer()
	data1 := map[string]string{
		"/ipv4/127.0.0.1": "value is not used",
	}
	data2 := map[string]string{
		"/ipv4/127.0.0.2": "value is not used",
	}
	if err = populateDatabase(client, data1); err != nil {
		t.Fatalf("Failed to populate etcd with example data1: %s", err)
	}
	changed := make(chan bool)
	resolver := &EtcdResolver{
		Client:      client,
		Timeout:     timeout,
		AnswerCount: 2,
		NotifyChangedFunc: func(added bool, key string) {
			changed <- true
		},
	}
	resolver.Start()
	<-changed
	// Check initial state
	ips, err := resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv4: %s", "example.com.", err)
	}
	if len(ips) < 1 || ips[0] != "127.0.0.1" {
		t.Fatalf("Wrong IPv4 address was returned: %s", ips)
	}
	// Change
	if err = populateDatabase(client, data2); err != nil {
		t.Fatalf("Failed to populate etcd with example data2: %s", err)
	}
	<-changed
	if err = deleteKeys(client, data1); err != nil {
		t.Fatalf("Failed to delete etcd keys of data1: %s", err)
	}
	<-changed
	// Check initial state
	ips, err = resolver.Resolve("example.com.", dns.TypeA, dns.ClassINET)
	if err != nil {
		t.Fatalf("Failed to resolve %q to IPv4", "example.com.")
	}
	if len(ips) < 1 || ips[0] != "127.0.0.2" {
		t.Fatalf("Wrong IPv4 address was returned: %s", ips)
	}
}
