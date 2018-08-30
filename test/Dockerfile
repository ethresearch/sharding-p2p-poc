FROM iron/go:dev as build-env

ADD . /src

RUN go get -d github.com/ethresearch/sharding-p2p-poc/...  
    go get ./... &&\
    make deps &&\
    go build 

COPY .sharding-p2p-poc    .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o sharding-p2p-poc .

FROM iron/go:dev

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=0 /go/src/github.com/ethresearch/sharding-p2p-poc/    .

CMD ["./sharding-p2p-poc"]
