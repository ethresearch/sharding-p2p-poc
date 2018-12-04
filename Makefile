build:
	sh scripts/build.sh
	./sharding-p2p-poc version

update-go-mod:
	export GO111MODULE=on && go mod download && go mod tidy

docker-build:
	sh scripts/docker-build.sh
