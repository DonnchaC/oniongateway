FROM golang:alpine

WORKDIR /go/src/entry_proxy

COPY entry_proxy .
RUN ls

RUN apk add --no-cache git \
    && go get -d . \
    && apk del git
RUN go build .

EXPOSE 80
EXPOSE 443
CMD ["entry_proxy"]