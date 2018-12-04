update-go-mod:
	export GO111MODULE=on && go mod download && go mod tidy

build:
	docker build -t ethresearch/sharding-p2p:dev .
