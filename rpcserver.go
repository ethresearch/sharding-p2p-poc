package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pbrpc "github.com/ethresearch/sharding-p2p-poc/pb/rpc"
	"google.golang.org/grpc"

	opentracing "github.com/opentracing/opentracing-go"
)

type server struct {
	pbrpc.PocServer
	node       *Node
	parentSpan opentracing.Span
	rpcServer  *grpc.Server
}

func makeResponse(success bool, message string) *pbrpc.Response {
	var status pbrpc.Response_Status
	if success {
		status = pbrpc.Response_SUCCESS
	} else {
		status = pbrpc.Response_FAILURE
	}
	return &pbrpc.Response{Status: status, Message: message}
}

func makePlainResponse(success bool, message string) *pbrpc.RPCPlainResponse {
	return &pbrpc.RPCPlainResponse{
		Response: makeResponse(success, message),
	}
}

func (s *server) AddPeer(ctx context.Context, req *pbrpc.RPCAddPeerRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for AddPeer
	span := opentracing.StartSpan("AddPeer", opentracing.ChildOf(s.parentSpan.Context()))
	defer span.Finish()

	log.Printf("rpcserver:AddPeer: receive=%v", req)
	_, targetPID, err := makeKey(int(req.Seed))
	mAddr := fmt.Sprintf(
		"/ip4/%s/tcp/%d/ipfs/%s",
		req.Ip,
		req.Port,
		targetPID.Pretty(),
	)
	if err != nil {
		log.Fatal(err)
	}

	var replyMsg string
	success := s.node.AddPeer(mAddr)
	if success {
		replyMsg = fmt.Sprintf("Added Peer %v:%v, pid=%v!", req.Ip, req.Port, targetPID)
		// Tag the span with peer info
		span.SetTag("Peer info", fmt.Sprintf("%v:%v", req.Ip, req.Port))
	} else {
		replyMsg = fmt.Sprintf("Failed to add Peer %v:%v, pid=%v!", req.Ip, req.Port, targetPID)
	}
	span.SetTag("Status", success)
	return makePlainResponse(success, replyMsg), nil
}

func (s *server) SubscribeShard(
	ctx context.Context,
	req *pbrpc.RPCSubscribeShardRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for SubscribeShard
	span := opentracing.StartSpan("SubscribeShard", opentracing.ChildOf(s.parentSpan.Context()))
	defer span.Finish()

	log.Printf("rpcserver:SubscribeShardRequest: receive=%v", req)
	for _, shardID := range req.ShardIDs {
		s.node.ListenShard(shardID)
		time.Sleep(time.Millisecond * 30)
	}
	s.node.PublishListeningShards()
	replyMsg := fmt.Sprintf(
		"Subscribed shard %v",
		req.ShardIDs,
	)
	// Tag the span with shard info
	span.SetTag("Shard info", fmt.Sprintf("shard %v", req.ShardIDs))
	return makePlainResponse(true, replyMsg), nil
}

func (s *server) UnsubscribeShard(
	ctx context.Context,
	req *pbrpc.RPCUnsubscribeShardRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for UnsubscribeShard
	span := opentracing.StartSpan("UnsubscribeShard", opentracing.ChildOf(s.parentSpan.Context()))
	defer span.Finish()

	log.Printf("rpcserver:UnsubscribeShardRequest: receive=%v", req)
	for _, shardID := range req.ShardIDs {
		s.node.UnlistenShard(shardID)
		time.Sleep(time.Millisecond * 30)
	}
	s.node.PublishListeningShards()
	replyMsg := fmt.Sprintf(
		"Unsubscribed shard %v",
		req.ShardIDs,
	)
	// Tag the span with shard info
	span.SetTag("Shard info", fmt.Sprintf("shard %v", req.ShardIDs))
	return makePlainResponse(true, replyMsg), nil
}

func (s *server) GetSubscribedShard(
	ctx context.Context,
	req *pbrpc.RPCGetSubscribedShardRequest) (*pbrpc.RPCGetSubscribedShardResponse, error) {
	// Add span for GetSubscribedShard
	span := opentracing.StartSpan("GetSubscribedShard", opentracing.ChildOf(s.parentSpan.Context()))
	defer span.Finish()

	log.Printf("rpcserver:GetSubscribedShard: receive=%v", req)
	shardIDs := s.node.GetListeningShards()
	res := &pbrpc.RPCGetSubscribedShardResponse{
		Response: makeResponse(true, ""),
		ShardIDs: shardIDs,
	}
	// Tag the span with shards info
	span.SetTag("Shards subscribed to", shardIDs)
	return res, nil
}

func (s *server) BroadcastCollation(
	ctx context.Context,
	req *pbrpc.RPCBroadcastCollationRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for BroadcastCollation
	span := opentracing.StartSpan("BroadcastCollation", opentracing.ChildOf(s.parentSpan.Context()))
	defer span.Finish()

	log.Printf("rpcserver:BroadcastCollationRequest: receive=%v", req)
	shardID := req.ShardID
	numCollations := int(req.Number)
	timeInMs := req.Period
	sizeInBytes := req.Size
	if sizeInBytes > 100 {
		sizeInBytes -= 100
	}
	for i := 0; i < numCollations; i++ {
		// control the speed of sending collations
		time.Sleep(time.Millisecond * time.Duration(timeInMs))
		randBytes := make([]byte, sizeInBytes)
		rand.Read(randBytes)
		err := s.node.broadcastCollation(
			ShardIDType(shardID),
			i,
			randBytes,
		)
		if err != nil {
			failureMsg := fmt.Sprintf("broadcastcollation fails: %v", err)
			log.Println(failureMsg)
			return makePlainResponse(false, failureMsg), err
		}
	}
	replyMsg := fmt.Sprintf(
		"Finished sending %v size=%v collations in shard %v",
		numCollations,
		sizeInBytes,
		shardID,
	)
	// Tag the span with collations info
	span.SetTag("Number of collations", numCollations)
	span.SetTag("Size of collation", sizeInBytes)
	span.SetTag("Shard", shardID)
	return makePlainResponse(true, replyMsg), nil
}

func (s *server) SendCollation(
	ctx context.Context,
	req *pbrpc.RPCSendCollationRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for BroadcastCollation
	span := opentracing.StartSpan("SendCollation", opentracing.ChildOf(s.parentSpan.Context()))
	defer span.Finish()

	log.Printf("rpcserver:SendCollationRequest: receive=%v", req)
	collation := req.Collation
	err := s.node.broadcastCollationMessage(collation)
	if err != nil {
		failureMsg := fmt.Sprintf("broadcastcollation fails: %v", err)
		log.Println(failureMsg)
		return makePlainResponse(false, failureMsg), err
	}
	replyMsg := fmt.Sprintf(
		"Finished sending collation shardID=%v, period=%v, len(blobs)=%v",
		collation.ShardID,
		collation.Period,
		len(collation.Blobs),
	)
	// Tag the span with collations info
	span.SetTag("Shard", collation.ShardID)
	span.SetTag("Period of collation", collation.Period)
	span.SetTag("Blobs of collation", collation.Blobs)
	return makePlainResponse(true, replyMsg), nil
}

func (s *server) StopServer(
	ctx context.Context,
	req *pbrpc.RPCStopServerRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for StopServer
	span := opentracing.StartSpan("StopServer", opentracing.ChildOf(s.parentSpan.Context()))

	log.Printf("rpcserver:StopServer: receive=%v", req)
	time.Sleep(time.Millisecond * 500)
	span.Finish()
	log.Printf("Closing RPC server by rpc call...")
	s.rpcServer.Stop()

	replyMsg := fmt.Sprintf("Closed RPC server")
	return makePlainResponse(true, replyMsg), nil
}

func runRPCServer(n *Node, addr string) {
	// Start a new trace
	span := opentracing.StartSpan("RPC Server")
	span.SetTag("Seed Number", n.number)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	pbrpc.RegisterPocServer(s, &server{node: n, parentSpan: span, rpcServer: s})

	// Catch interupt signal
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Printf("Closing RPC server by Interrupt signal...")
		s.Stop()
	}()

	log.Printf("rpcserver: listening to %v", addr)
	s.Serve(lis)
	span.Finish()
}
