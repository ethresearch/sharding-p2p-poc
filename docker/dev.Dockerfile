FROM golang:1.11-alpine

RUN apk add git
RUN go get -v github.com/whyrusleeping/gx

WORKDIR /bin

COPY go.mod go.sum package.json /bin/

RUN go mod download
RUN gx install

COPY . /bin/

RUN CGO_ENABLED=0 go build

CMD ["/bin/sharding-p2p-poc"]
