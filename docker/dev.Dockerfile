FROM golang:1.10.3-alpine
WORKDIR /go/src/github.com/ethresearch/sharding-p2p-poc
RUN apk add git python3 make
RUN go get -u -v github.com/whyrusleeping/gx &&\
    go get -u -v github.com/whyrusleeping/gx-go

COPY *.go ./
COPY pb ./pb

RUN go get -d -v .

COPY package.json .
RUN gx install
RUN go build

CMD ["./sharding-p2p-poc"]
