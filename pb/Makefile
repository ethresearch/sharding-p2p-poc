RPC_PATH_PATTERN := */rpc/*
EVENT_PATH_PATTERN := */event/*
RPC_PROTOS := $(shell find . -path "$(RPC_PATH_PATTERN)" -name "*.proto")
EVENT_PROTOS := $(shell find . -path "$(EVENT_PATH_PATTERN)" -name "*.proto")
MSG_PROTOS := $(shell find . -not -path "$(RPC_PATH_PATTERN)" -not -path "$(EVENT_PATH_PATTERN)" -name "*.proto")
RPC_PBGOS := $(patsubst %.proto,%.pb.go, $(RPC_PROTOS))
EVENT_PBGOS := $(patsubst %.proto,%.pb.go, $(EVENT_PROTOS))
MSG_PBGOS := $(patsubst %.proto,%.pb.go, $(MSG_PROTOS))

all: $(RPC_PBGOS) $(EVENT_PBGOS) $(MSG_PBGOS)

# Note: ; and \ are necessary, and don't mix up shell scripts with
#	the `make` directives
# Explanation: The following if/else add "plugins=grpc" to `--go_out`
# 	when the file is under the directory rpc/
# TODO: seems better to have one makefile per subdirectory, and one makefile
#	under the pb root directory to trigger all compiling jobs
%.pb.go: %.proto
	@if [ "$(findstring $^,$(RPC_PROTOS))" = "$^" ]; then\
		protoc -I$(GOPATH)/src -I$(dir $^) $^ --go_out=plugins=grpc:$(dir $^);\
	elif [ "$(findstring $^,$(EVENT_PROTOS))" = "$^" ]; then\
		protoc -I$(GOPATH)/src -I$(dir $^) $^ --go_out=plugins=grpc:$(dir $^);\
	else\
		protoc -I$(GOPATH)/src -I$(dir $^) $^ --go_out=$(dir $^);\
	fi

clean:
	rm -f $(RPC_PBGOS) $(EVENT_PBGOS) $(MSG_PBGOS)
