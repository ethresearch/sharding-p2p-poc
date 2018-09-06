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

	logging "github.com/ipfs/go-log"
)

var logger = logging.Logger("tracing")

type server struct {
	pbrpc.PocServer
	node      *Node
	ctx       context.Context
	rpcServer *grpc.Server
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

func (s *server) AddPeer(
	ctx context.Context,
	req *pbrpc.RPCAddPeerRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for AddPeer of RPC Server
	spanctx := logger.Start(s.ctx, "RPCServer.AddPeer")
	defer logger.Finish(spanctx)

	log.Printf("rpcserver:AddPeer: receive=%v", req)
	_, targetPID, err := makeKey(int(req.Seed))
	mAddr := fmt.Sprintf(
		"/ip4/%s/tcp/%d/ipfs/%s",
		req.Ip,
		req.Port,
		targetPID.Pretty(),
	)
	if err != nil {
		logger.FinishWithErr(spanctx, fmt.Errorf("Failed to generate peer key/ID with seed: %v, err: %v", req.Seed, err))
		log.Fatalf("Failed to generate peer key/ID with seed: %v, err: %v", req.Seed, err)
	}

	var replyMsg string
	err = s.node.AddPeer(spanctx, mAddr)
	success := (err == nil)
	if success {
		replyMsg = fmt.Sprintf("Added Peer %v:%v, pid=%v!", req.Ip, req.Port, targetPID)
		// Tag the span with peer info
		logger.SetTag(spanctx, "Added peer", fmt.Sprintf("%v:%v", req.Ip, req.Port))
	} else {
		replyMsg = fmt.Sprintf("Failed to add Peer %v:%v, pid=%v!", req.Ip, req.Port, targetPID)
	}
	return makePlainResponse(success, replyMsg), nil
}

func (s *server) SubscribeShard(
	ctx context.Context,
	req *pbrpc.RPCSubscribeShardRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for SubscribeShard
	spanctx := logger.Start(s.ctx, "RPCServer.SubscribeShard")
	defer logger.Finish(spanctx)

	subscribedShardID := make([]int64, 0)
	log.Printf("rpcserver:SubscribeShardRequest: receive=%v", req)
	for _, shardID := range req.ShardIDs {
		if err := s.node.ListenShard(spanctx, shardID); err != nil {
			logger.SetErr(spanctx, fmt.Errorf("Failed to listen to shard %v", shardID))
			log.Printf("Failed to listen to shard %v", shardID)
		} else {
			subscribedShardID = append(subscribedShardID, shardID)
		}
		time.Sleep(time.Millisecond * 30)
	}
	s.node.PublishListeningShards(spanctx)
	replyMsg := fmt.Sprintf(
		"Subscribed shard %v",
		req.ShardIDs,
	)

	// Tag the shardIDs which are successfully subscribed to
	logger.SetTag(spanctx, "ShardIDs", fmt.Sprintf("%v", subscribedShardID))
	return makePlainResponse(true, replyMsg), nil
}

func (s *server) UnsubscribeShard(
	ctx context.Context,
	req *pbrpc.RPCUnsubscribeShardRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for UnsubscribeShard
	spanctx := logger.Start(s.ctx, "RPCServer.UnsubscribeShard")
	defer logger.Finish(spanctx)

	unsubscribedShardID := make([]int64, 0)
	log.Printf("rpcserver:UnsubscribeShardRequest: receive=%v", req)
	for _, shardID := range req.ShardIDs {
		if err := s.node.UnlistenShard(spanctx, shardID); err != nil {
			logger.SetErr(spanctx, fmt.Errorf("Failed to unlisten shard %v", shardID))
			log.Printf("Failed to unlisten shard %v", shardID)
		} else {
			unsubscribedShardID = append(unsubscribedShardID, shardID)
		}
		time.Sleep(time.Millisecond * 30)
	}
	s.node.PublishListeningShards(spanctx)
	replyMsg := fmt.Sprintf(
		"Unsubscribed shard %v",
		req.ShardIDs,
	)
	// Tag the span with shard info
	logger.SetTag(spanctx, "Shard info", fmt.Sprintf("shard %v", unsubscribedShardID))
	return makePlainResponse(true, replyMsg), nil
}

func (s *server) GetSubscribedShard(
	ctx context.Context,
	req *pbrpc.RPCGetSubscribedShardRequest) (*pbrpc.RPCGetSubscribedShardResponse, error) {
	// Add span for GetSubscribedShard
	spanctx := logger.Start(s.ctx, "RPCServer.GetSubscribedShard")
	defer logger.Finish(spanctx)

	log.Printf("rpcserver:GetSubscribedShard: receive=%v", req)
	shardIDs := s.node.GetListeningShards()
	res := &pbrpc.RPCGetSubscribedShardResponse{
		Response: makeResponse(true, ""),
		ShardIDs: shardIDs,
	}
	// Tag the span with shards info
	logger.SetTag(spanctx, "shardIDs", shardIDs)
	return res, nil
}

func (s *server) BroadcastCollation(
	ctx context.Context,
	req *pbrpc.RPCBroadcastCollationRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for BroadcastCollation
	spanctx := logger.Start(s.ctx, "RPCServer.BroadcastCollation")
	defer logger.Finish(spanctx)
	// Set shardID info in Baggage
	logger.SetTag(spanctx, "shardID", fmt.Sprintf("%v", req.ShardID))

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
			spanctx,
			ShardIDType(shardID),
			i,
			randBytes,
		)
		if err != nil {
			failureMsg := fmt.Sprintf("broadcastcollation fails: %v", err)
			log.Println(failureMsg)
			logger.FinishWithErr(spanctx, fmt.Errorf("Failed to broadcast collation"))
			return makePlainResponse(false, failureMsg), err
		}
	}
	replyMsg := fmt.Sprintf(
		"Finished sending %v size=%v collations in shard %v",
		numCollations,
		sizeInBytes,
		shardID,
	)
	// Tag collations info if nothing goes wrong
	logger.SetTag(spanctx, "numCollations", numCollations)
	logger.SetTag(spanctx, "sizeInBytes", sizeInBytes)
	return makePlainResponse(true, replyMsg), nil
}

func (s *server) SendCollation(
	ctx context.Context,
	req *pbrpc.RPCSendCollationRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for SendCollation
	spanctx := logger.Start(s.ctx, "RPCServer.SendCollation")
	defer logger.Finish(spanctx)

	log.Printf("rpcserver:SendCollationRequest: receive=%v", req)
	collation := req.Collation
	err := s.node.broadcastCollationMessage(collation)
	if err != nil {
		failureMsg := fmt.Sprintf("broadcastcollation failed: %v", err)
		log.Println(failureMsg)
		logger.FinishWithErr(spanctx, fmt.Errorf("Failed to broadcast collation message, err: %v", err))
		return makePlainResponse(false, failureMsg), err
	}
	replyMsg := fmt.Sprintf(
		"Finished sending collation shardID=%v, period=%v, len(blobs)=%v",
		collation.ShardID,
		collation.Period,
		len(collation.Blobs),
	)
	// Tag collation info if nothing goes wrong
	logger.SetTag(spanctx, "Shard", collation.ShardID)
	logger.SetTag(spanctx, "Period of collation", collation.Period)
	logger.SetTag(spanctx, "Blobs of collation", collation.Blobs)
	return makePlainResponse(true, replyMsg), nil
}

func (s *server) StopServer(
	ctx context.Context,
	req *pbrpc.RPCStopServerRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for StopServer
	spanctx := logger.Start(s.ctx, "RPCServer.StopServer")
	defer logger.Finish(spanctx)

	log.Printf("rpcserver:StopServer: receive=%v", req)
	time.Sleep(time.Millisecond * 500)
	log.Printf("Closing RPC server by rpc call...")
	go func() {
		time.Sleep(time.Second * 1)
		s.rpcServer.Stop()
	}()

	replyMsg := fmt.Sprintf("Closed RPC server")
	return makePlainResponse(true, replyMsg), nil
}

func runRPCServer(n *Node, addr string) {
	// Start a new trace
	ctx := context.Background()
	ctx = logger.Start(ctx, "RPCServer")
	defer logger.Finish(ctx)
	logger.SetTag(ctx, "Seed Number", n.number)

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.FinishWithErr(ctx, fmt.Errorf("Failed to set up a service listening on %s, err: %v", addr, err))
		log.Fatalf("Failed to set up a service listening on %s, err: %v", addr, err)
	}
	s := grpc.NewServer()
	pbrpc.RegisterPocServer(s, &server{node: n, ctx: ctx, rpcServer: s})

	// Catch interupt signal
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Printf("Closing RPC server by Interrupt signal...")
		s.Stop()
	}()

	log.Printf("rpcserver: listening to %v", addr)
	if err := s.Serve(lis); err != nil {
		logger.FinishWithErr(ctx, fmt.Errorf("Failed to serve the RPC server, err: %v", err))
		log.Fatalf("Failed to serve the RPC server, err: %v", err)
	}
}
