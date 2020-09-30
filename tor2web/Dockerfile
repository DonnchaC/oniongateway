FROM alpine:latest

RUN apk add --no-cache tor

EXPOSE 9050

ADD torrc /etc/tor/torrc

ENTRYPOINT ["/usr/bin/tor"]


