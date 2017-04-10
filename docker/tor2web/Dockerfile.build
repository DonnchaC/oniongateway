FROM ubuntu:latest

MAINTAINER "Donncha O'Cearbhaill <donncha@donncha.is>"

RUN apt-get -y update && apt-get install -y build-essential automake libssl-dev libevent-dev git

RUN git clone https://git.torproject.org/tor.git -b release-0.2.9
WORKDIR tor

RUN ./autogen.sh
RUN ./configure --disable-asciidoc --enable-tor2web-mode --enable-static-libevent --enable-static-zlib --with-libevent-dir=/usr/local/ --with-zlib-dir=/usr/local/
RUN make
COPY src/or/tor /build

#ENTRYPOINT ["/bin/sh"]
