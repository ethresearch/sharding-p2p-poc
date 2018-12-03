gx:
	go get -u github.com/whyrusleeping/gx
	go get -u github.com/whyrusleeping/gx-go

deps: gx
	gx --verbose install --global
	gx-go rewrite
	python3 ./script/partial-gx-uw.py .

update-go-mod:
	export GO111MODULE=on && go mod download && go mod tidy

build:
	docker build -t ethresearch/sharding-p2p:dev .

partial-gx-rw:
	gx-go rw
	python3 ./script/partial-gx-uw.py .

gx-rw:
	gx-go rw

gx-uw:
	gx-go uw
