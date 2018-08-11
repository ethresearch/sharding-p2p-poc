RPC_PATH_PATTERN := */rpc/*
RPC_PROTOS := $(shell find . -path "$(RPC_PATH_PATTERN)")
MSG_PROTOS := $(shell find . -not -path "$(RPC_PATH_PATTERN)" -name "*.proto")
RPC_PBGOS := $(patsubst %.proto,%.pb.go, $(RPC_PROTOS))
MSG_PBGOS := $(patsubst %.proto,%.pb.go, $(MSG_PROTOS))

all: $(RPC_PBGOS) $(MSG_PBGOS)

# Note: ; and \ are necessary, and don't mix up shell scripts with
#	the `make` directives
# Explanation: The following if/else add "plugins=grpc" to `--go_out`
# 	when the file is under the directory rpc/
# TODO: seems better to have one makefile per subdirectory, and one makefile
%.pb.go: %.proto
	@if [ "$(findstring $^,$(RPC_PROTOS))" = "$^" ]; then\
		protoc -I $(dir $^) $^ --go_out=plugins=grpc:$(dir $^);\
	else\
		protoc -I $(dir $^) $^ --go_out=$(dir $^);\
	fi

clean:
	rm */*.pb.go
