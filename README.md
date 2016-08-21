oniongateway
============

[![Travis build][travis-badge]][travis-page]
[![AppVeyor build][appveyor-badge]][appveyor-page]
[![Coverage Status][coveralls-badge]][coveralls-page]

End-to-End encrypted Tor2Web gateway.

This software is under active development and likely contains many bugs.
Please open bugs on Github if you discover any issues with the software
or documentation.

Installation and Usage
----------------------

OnionGateway requires a working Go build environment. Once you have that configured you can build the `entry_proxy` and it's dependencies with the `go get` command. The binary will be built inside your `$GOPATH/bin` directory. You may need to add this directory to your shell `$PATH` environment variable.

```bash
go get github.com/DonnchaC/oniongateway
sudo setcap 'cap_net_bind_service=+ep' $(which entry_proxy)
entry_proxy
```

To improve performance, the server running the `entry_proxy` should have a Tor daemon which is running in `Tor2Web` mode. There are instructions for compiling Tor in this mode at https://github.com/globaleaks/Tor2web/wiki/Installation-Guide#build-tor-with-tor2web-mode-and-some-patches.

`entry_proxy` uses the DNS system to resolve domain names to hidden service addresses. You should install a local caching DNS server to avoid making a DNS query for every client connection.

```bash
apt-get install unbound
vi /etc/resolv.conf # insert top: nameserver 127.0.0.1
```


Using a domain with OnionGateway
--------------------------------

To use a domain with OnionGateway you must configure your hidden service and point your domain at one or more `oniongateway` servers.

Your hidden service should be configured to listen on port 443 with a valid CA-signed certificate for your public domain. For example it could present a valid cert for `myblog.com` from LetsEncrypt. You should configure your hidden service to also serve content directly to hidden service users over HTTP.

Example torrc file:

```
HiddenServiceDir /var/lib/tor/myblog/
HiddenServicePort 80 127.0.0.1:80
HiddenServicePort 443 127.0.0.1:443
```

Example nginx configuration:

```
server {
    listen 127.0.0.1:80;
    server_name .myblogaaaaaaaaaa.onion;

    include sites-available/myblog_com.inc;
}

server {
    listen 127.0.0.1:443;
    server_name .myblog.com;

    ssl on;
    ssl_certificate /path/to/myblog_com.crt;
    ssl_certificate_key /path/to/myblog_com.key;
    add_header Strict-Transport-Security "max-age=31536000; includeSubdomains";
    # TODO: HTTP Public Key Pinning (HPKP)

    include sites-available/myblog_com.inc;
}
```

You can place site specific configuration options in the `myblog_com.inc` to avoid repeating options between host blocks.

You will need to add an A and AAAA records for you domain `myblog.com` which point to one or more online `oniongateway` servers. Finally you need to create a DNS record to indicate your hidden service address to the OnionGateway.

Test your DNS settings with `dig`:

```bash
$ dig pasta.cf TXT

...

;; QUESTION SECTION:
;pasta.cf.                      IN      TXT

;; ANSWER SECTION:
pasta.cf.               21600   IN      TXT     "onion=pastagdsp33j7aoq.onion"
```

Once you have the DNS and hidden service configured you should be able to access your site at `https://myblog.com`.

[travis-page]: https://travis-ci.org/DonnchaC/oniongateway
[travis-badge]: https://travis-ci.org/DonnchaC/oniongateway.png
[appveyor-page]: https://ci.appveyor.com/project/DonnchaC/oniongateway
[appveyor-badge]: https://ci.appveyor.com/api/projects/status/i98wvpauvnrbvemw
[coveralls-page]: https://coveralls.io/github/DonnchaC/oniongateway
[coveralls-badge]: https://coveralls.io/repos/github/DonnchaC/oniongateway/badge.png
