build:
	GIT_COMMIT=$(git rev-list -1 HEAD) && go build -ldflags "-X main.GitCommit=${GIT_COMMIT}"

update-go-mod:
	export GO111MODULE=on && go mod download && go mod tidy

docker-build:
	docker build -t ethresearch/sharding-p2p:dev .
