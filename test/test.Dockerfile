FROM golang:1.10.3-alpine as base

RUN apk add --no-cache make gcc musl-dev linux-headers libc6-compat git python3 make
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go get -v github.com/ethresearch/sharding-p2p-poc

FROM iron/go:latest
COPY --from=base /go/bin/sharding-p2p-poc /go/bin/

ENV PATH=$PATH:/go/bin

EXPOSE 8369 8370

CMD ["sharding-p2p-poc"]

