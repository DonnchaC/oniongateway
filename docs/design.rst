End-to-end encrypted onion services for non-Tor clients
=======================================================

:Author: Donncha Ó Cearbhaill
:Created: 22-Aug-2015
:Status: Draft

1. Overview:
------------

This proposal describes a system to allow non-Tor users to securely access
location-hidden services via a domain name chosen by the hidden service
operator.

2. Motivation:
--------------

Tor hidden services have typically been used to provide anonymity to both
services and clients. There are however use cases where a publisher or
content provider requires anonymity but their users do not. The publishers
typically would like their content to be available securely and to the
widest possible audience.

Tor2Web currently allows non-Tor users to access hidden services but it has
a number of limitations. Tor2Web deployments are limited due to requirement
to entrust Tor2Web service providers with users location information and the
content of their requests.

The onion subdomain based addressing in Tor2Web poses a usability issue and
can be unwieldy for users without providing any self-authenticating
properties. Tor2Web can be configured with a custom domain for an individual
hidden service but this requires manual configuration and additional
infrastructure.

Ideally non-Tor users should be able to access location-hidden services
under a standard domain name with end-to-end encryption from the client's
browser to the hidden service web server. The system should be decentralized
and it should not require the hidden service operator to run any clearnet
systems.

3. Proposal:
------------

At a high level this proposal specifies a system of TCP entry proxies which
transparently proxy the TLS connections of non-Tor clients to a hidden
service. A client who requests a domain name such as https://myblog.com
should get connected to a hidden service endpoint which has a valid SSL
certificate for myblog.com.

3.1. Hidden Service configuration
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

A hidden service operator obtains a public domain name and a corresponding
CA signed SSL certificate. The operator configures the hidden service web
server with the SSL certificate for the external domain.

The domain name is pointed at an authoritative nameserver provider. The
operator should also set a TXT DNS record containing the onion address of
their hidden service.

3.2. Entry Proxy
~~~~~~~~~~~~~~~~

Modern web browsers send a Server Name Indication (SNI) field in the initial
ClientHello of the TLS handshake. The entry proxy reads this field to
determine the clients destination domain. The proxy performs a DNS lookup at
that domain for a TXT record which specifies the corresponding onion address
for the provided domain.

The TLS entry proxy connects to the onion service via a Tor2Web-type one-hop
circuit and then begins transparently proxying TLS traffic between the
clients browser and the destination onion service.

3.3. Resolving Nameserver
~~~~~~~~~~~~~~~~~~~~~~~~~

The key component of this system is a DNS name server which can direct
clients to an online entry proxy via a round-robin type system. In a basic
solution the onion service operator could manually specify one or more entry
proxies in the A records for their domain on their existing authoritative
DNS provider.

In practice entry proxy churn would result in changing set of entry proxies.
The nameserver will need regularly update its set of online entry proxies
and remove proxies which are malfunctioning, malicious or otherwise unusable.

The resolver may run a scanner to check its known proxies or load a list
from an external service. Additionally the resolver may detect if an
entry proxy blacklists a domain for which it is responsible and avoid
routing clients to that entry proxy.

It is expected that independent service providers will run their own
domain->onion resolving nameserver in diverse jurisdictions as free or paid
services.

4. Implementation:
------------------

The entry proxy component can be implemented with changes to the current Tor
code base. Integration directly within Tor would allow use of the existing
network consensus and bandwidth measurement systems to be used to discover
available entry proxies. It would also allow for malicious entry proxies to
be blacklisted.

Alternatively the entry proxy could be implemented in the existing Tor2Web
software or as a standalone software package. Implementing outside of Tor
would be faster and it would avoid the risk of losing Tor relay capacity as
a result of legal threats to the entry proxies.

The resolving nameserver is the most complicated component of this system.
The component will eventually require a DNS server, a management interface,
and a set of network monitoring tools.

5. Security and resiliency implications:
----------------------------------------

5.1. Availability Attacks:
~~~~~~~~~~~~~~~~~~~~~~~~~~

Adversaries can attack the availability of a publicly-proxied hidden
service at a number of levels:

* Censorship or shutdown of entry proxy:

  Attacks on individual entry proxies are mitigated by performing DNS
  based round-robin between many online entry proxies. The Resolver system
  should be able to quickly remove entry proxies which misbehave or which
  go offline.

* Censorship or seizure of the hidden service public domain:

  DNS based blocks are widely deployed for censorship and may be difficult
  to avoid. Domain registrars can also be forced to suspend domain names.
  Service operators should considering running their service under a TLD
  which is less vulnerable to these type of coercive threats.

* Takedown of a nameserver provider:

  Multiple resolving nameservers can be configure for each forwarded
  domain. Using nameservers maintained by different providers can provide
  resilience to attacks against a single nameserver provider.

5.2. Security Attacks:

Entry proxies and exit relays have a similar ability to monitor and
interfere with client traffic. This is greater risk of targeted
interference from entry proxies as they can also determine the client's
network location.

* Man-in-the-middle HTTP connections:

  Entry proxies have the ability to man-in-the-middle HTTP connections.
  Service operators should send HSTS header to force clients to
  automatically use TLS for all future connections.

* TLS man-in-the-middle with CA-signed certificate

  Some commercial CA cert providers allow for domain ownership to be
  validated by providing a file over HTTP at the domain. A malicious entry
  proxy could successfully obtain a CA-signed certificate from one of
  these certificate authorities.

  Service operators can minimize their exposure to this type of attack by
  using HPKP headers to limit the set of valid certificate authorities for
  their domain.

  A resolver could allow services to register subdomains under a domain
  which uses HSTS preloading to pin the root domain to single CA. Pinning
  to a CA which publishes certificate transparency logs would provide
  a good defense against unknown man-in-the-middle attacks.
