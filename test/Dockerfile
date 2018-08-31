FROM golang:1.10.3-alpine 

WORKDIR /go/src/github.com/ethresearch/sharding-p2p-poc

COPY . /go/src/github.com/ethresearch/sharding-p2p-poc

RUN apk add git python3 make && \
    go get -d -v . && \
    make deps && \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -o main .

EXPOSE 13000 10000

CMD ["/main"]
