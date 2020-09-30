FROM alpine:latest

RUN apk add --no-cache wget

EXPOSE 9050

ADD torrc /etc/tor/torrc

ENTRYPOINT ["/usr/bin/tor"]


