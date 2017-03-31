package main

import (
    "fmt"
    "strings"
    "regexp"
)

type SubdomainResolver struct {
    regex       *regexp.Regexp
    parentDomain string
}

func NewSubdomainResolver(parent_domain string) *SubdomainResolver {
    return &SubdomainResolver{
        parentDomain: parent_domain,
        regex:        regexp.MustCompile("^([a-z0-9]{16})$"),
    }
}

func (r *SubdomainResolver) ResolveToOnion(host string) (string, error) {

    if !strings.HasSuffix(host, r.parentDomain) {
        return "", fmt.Errorf("Host %q is not a subdomain of %q", host, r.parentDomain)
    }

    subdomains := strings.TrimSuffix(host, "." + r.parentDomain)
    subdomain_parts := strings.Split(subdomains, ".")
    onion := subdomain_parts[len(subdomain_parts)-1]

    match := r.regex.FindStringSubmatch(onion)
    if match != nil {
        return match[1] + ".onion", nil // the match we are interested in
    }
    return "", fmt.Errorf("The hostname %q did not have a valid onion address subdomain", host)
}
