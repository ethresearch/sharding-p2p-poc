build:
	GIT_COMMIT=$(git rev-list -1 HEAD) && go build -ldflags "-X main.GitCommit=${GIT_COMMIT}"
gx:
	go get -u github.com/whyrusleeping/gx
	go get -u github.com/whyrusleeping/gx-go

deps: gx
	gx --verbose install --global
	gx-go rewrite
	python3 ./script/partial-gx-uw.py .

build-prod:
	docker build -f docker/prod.Dockerfile -t ethresearch/sharding-p2p:latest .

build-dev:
	docker build -f docker/dev.Dockerfile -t ethresearch/sharding-p2p:dev .

run-dev:
	docker run -it --rm -v $(PWD):/go/src/github.com/ethresearch/sharding-p2p-poc ethresearch/sharding-p2p:dev sh -c "go build -v -o main ."

test-dev: partial-gx-rw
	docker run -it --rm -v $(PWD):/go/src/github.com/ethresearch/sharding-p2p-poc ethresearch/sharding-p2p:dev sh -c "go test"
	gx-go uw

run-many-dev:
	docker-compose -f docker/dev.docker-compose.yml up --scale node=5

down-dev:
	docker-compose -f docker/dev.docker-compose.yml down

run-many-prod:
	docker-compose -f docker/prod.docker-compose.yml up --scale node=5

down-prod:
	docker-compose -f docker/prod.docker-compose.yml down

partial-gx-rw:
	gx-go rw
	python3 ./script/partial-gx-uw.py .

gx-rw:
	gx-go rw

gx-uw:
	gx-go uw
