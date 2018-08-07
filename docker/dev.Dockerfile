FROM golang:1.10.3
WORKDIR /go/sharding-p2p
COPY . /go/sharding-p2p
RUN go get -d -v .
RUN make deps

CMD ["sh"]