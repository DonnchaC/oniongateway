FROM alpine:latest

RUN apk add --update sudo
RUN adduser -G abuild -g "Alpine Package Builder" -s /bin/sh -D builder \
  && echo "builder ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

USER builder

WORKDIR /home/builder
RUN sudo apk add --update alpine-sdk \
  && abuild-keygen -i -a \
  && git clone --depth 1 git://dev.alpinelinux.org/aports \
  && cd ~/aports/community/tor \
  && sed -i 's/--enable-transparent/--enable-transparent --enable-tor2web-mode/g' APKBUILD \
  # Disable apk tests as they will fail (https://bugs.torproject.org/20061)
  && sed -i 's/make test//g' APKBUILD \
  && abuild verify && abuild -r \
  && cd ~/packages/community/x86_64 \
  && sudo apk add --allow-untrusted tor-0.*.apk \
  && rm -Rf ~/aports \
  && sudo apk del alpine-sdk

USER root

EXPOSE 9050

ADD tor2web-torrc /etc/tor/torrc

ENTRYPOINT ["/usr/bin/tor"]


