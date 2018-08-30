FROM alpine AS base
RUN apk add make gcc musl-dev linux-headers libc6-compat

FROM golang:1.10.3-alpine AS go-builder
WORKDIR /go/src/github.com/ethresearch/sharding-p2p-poc
COPY . /go/src/github.com/ethresearch/sharding-p2p-poc
RUN apk add git python3 make && \
    go get -d -v . && \
    make deps
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -o main 
RUN chmod 777 ./main

FROM scratch 
COPY --from=go-builder /go/src/github.com/ethresearch/sharding-p2p-poc/main /main
CMD ["/main"]
