gx:
	go get -u github.com/whyrusleeping/gx
	go get -u github.com/whyrusleeping/gx-go

deps: gx
	gx --verbose install --global
	gx-go rewrite
build:
	docker build -t ethereum/sharding-p2p:latest .
run:
	docker run ethereum/sharding-p2p:latest
run-many:
	docker-compose up --scale node=5