FROM alpine AS base
RUN apk add --no-cache make gcc musl-dev linux-headers libc6-compat

FROM golang:1.10.3-alpine AS go-builder
WORKDIR /go/src/github.com/ethresearch/sharding-p2p-poc
COPY . /go/src/github.com/ethresearch/sharding-p2p-poc
RUN apk add git python3 make
RUN go get -d -v .
RUN make deps
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -v -o main .

FROM base
COPY --from=go-builder /go/src/github.com/ethresearch/sharding-p2p-poc/main /main
CMD ["/main"]
