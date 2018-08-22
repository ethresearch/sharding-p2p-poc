FROM golang:1.10.3-alpine
WORKDIR /go/src/github.com/ethresearch/sharding-p2p-poc
COPY . /go/src/github.com/ethresearch/sharding-p2p-poc
RUN apk add git python3 make
RUN go get -d -v .
RUN make deps

CMD ["sh"]
