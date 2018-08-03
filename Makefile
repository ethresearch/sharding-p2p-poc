gx:
	go get -u github.com/whyrusleeping/gx
	go get -u github.com/whyrusleeping/gx-go

deps: gx
	gx --verbose install --global
	gx-go rewrite

build-prod:
	docker build -f docker/prod.Dockerfile -t ethereum/sharding-p2p:latest .

build-dev:
	docker build -f docker/dev.Dockerfile -t ethereum/sharding-p2p:dev .

run-dev:
	docker run -it --rm -v $(PWD):/go/sharding-p2p/ ethereum/sharding-p2p:dev sh -c "go build -v -o main ."

run-many-dev:
	docker-compose -f docker/dev.docker-compose.yml up --scale node=5

down-dev:
	docker-compose -f docker/dev.docker-compose.yml down

run-many-prod:
	docker-compose -f docker/prod.docker-compose.yml up --scale node=5

down-prod:
	docker-compose -f docker/prod.docker-compose.yml down