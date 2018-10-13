FROM golang:1.10.3-alpine as base

RUN apk add git python3 make && \
    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go get -v github.com/ethresearch/sharding-p2p-poc

FROM iron/go:latest
COPY --from=base /go/bin/sharding-p2p-poc /go/bin/

ENV PATH=$PATH:/go/bin

EXPOSE 8369 8370 13000 6831 6831/udp

CMD ["sharding-p2p-poc"]

