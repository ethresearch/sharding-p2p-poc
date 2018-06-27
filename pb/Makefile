all: minimal.pb.go rpc.pb.go

minimal.pb.go rpc.pb.go: minimal.proto rpc.proto
	protoc -I . rpc.proto minimal.proto --go_out=plugins=grpc:.
