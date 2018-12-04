FROM golang:1.11-alpine

RUN apk add git

WORKDIR /bin

COPY go.mod go.sum /bin/

RUN go mod download

COPY . /bin/

RUN CGO_ENABLED=0 go build

EXPOSE 8369 8370 13000 6831 6831/udp

CMD ["sharding-p2p-poc"]
