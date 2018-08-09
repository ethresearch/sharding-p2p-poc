package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	"google.golang.org/grpc"

	pb "github.com/ethresearch/sharding-p2p-poc/pb"
)

func callRPCAddPeer(rpcAddr string, ipAddr string, port int, seed int64) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewPocClient(conn)
	addPeerReq := &pb.RPCAddPeerReq{
		Ip:   ipAddr,
		Port: int32(port),
		Seed: seed,
	}
	log.Printf("rpcclient:AddPeer: sending=%v", addPeerReq)
	res, err := client.AddPeer(context.Background(), addPeerReq)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("rpcclient:AddPeer: result=%v", res)
}

func callRPCListShardPeer(rpcAddr string, shardID ShardIDType) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewPocClient(conn)
	listShardPeerReq := &pb.RPCListShardPeerReq{
		ShardID: shardID,
	}
	log.Printf("rpcclient:ListShardPeerReq: sending=%v", listShardPeerReq)
	res, err := client.ListShardPeer(context.Background(), listShardPeerReq)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("rpcclient:ListShardPeerReq: result=%v", res)
}

func callRPCSubscribeShard(rpcAddr string, shardIDs []ShardIDType) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewPocClient(conn)
	subscribeShardReq := &pb.RPCSubscribeShardReq{
		ShardIDs: shardIDs,
	}
	log.Printf("rpcclient:ShardReq: sending=%v", subscribeShardReq)
	res, err := client.SubscribeShard(context.Background(), subscribeShardReq)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("rpcclient:ShardReq: result=%v", res)
}

func callRPCUnsubscribeShard(rpcAddr string, shardIDs []ShardIDType) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewPocClient(conn)
	unsubscribeShardReq := &pb.RPCUnsubscribeShardReq{
		ShardIDs: shardIDs,
	}
	log.Printf("rpcclient:UnsubscribeShardReq: sending=%v", unsubscribeShardReq)
	res, err := client.UnsubscribeShard(context.Background(), unsubscribeShardReq)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("rpcclient:UnsubscribeShardReq: result=%v", res)
}

func callRPCGetSubscribedShard(rpcAddr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewPocClient(conn)
	getSubscribedShardReq := &pb.RPCGetSubscribedShardReq{}
	log.Printf("rpcclient:GetSubscribedShard: sending=%v", getSubscribedShardReq)
	res, err := client.GetSubscribedShard(context.Background(), getSubscribedShardReq)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("rpcclient:GetSubscribedShard: result=%v", res.ShardIDs)
}

func callRPCBroadcastCollation(
	rpcAddr string,
	shardID ShardIDType,
	numCollations int,
	collationSize int,
	period int) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewPocClient(conn)
	broadcastCollationReq := &pb.RPCBroadcastCollationReq{
		ShardID: shardID,
		Number:  int32(numCollations),
		Size:    int32(collationSize),
		Period:  int32(period),
	}
	log.Printf("rpcclient:BroadcastCollation: sending=%v", broadcastCollationReq)
	res, err := client.BroadcastCollation(context.Background(), broadcastCollationReq)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("rpcclient:BroadcastCollation: result=%v", res)
}

type server struct {
	pb.PocServer
	node *Node
}

func (s *server) AddPeer(ctx context.Context, req *pb.RPCAddPeerReq) (*pb.RPCReply, error) {
	log.Printf("rpcserver:AddPeer: receive=%v", req)
	_, targetPID, err := makeKey(req.Seed)
	mAddr := fmt.Sprintf(
		"/ip4/%s/tcp/%d/ipfs/%s",
		req.Ip,
		req.Port,
		targetPID.Pretty(),
	)
	if err != nil {
		log.Fatal(err)
	}
	s.node.AddPeer(mAddr)
	time.Sleep(time.Second * 1)
	res := &pb.RPCReply{
		Message: fmt.Sprintf("Added Peer %v:%v, pid=%v!", req.Ip, req.Port, targetPID),
		Status:  true,
	}
	return res, nil
}

func (s *server) ListShardPeer(
	ctx context.Context,
	req *pb.RPCListShardPeerReq) (*pb.RPCReply, error) {
	shardID := req.GetShardID()
	shardCollationTopic := getCollationsTopic(shardID)
	shardPeers := s.node.pubsubService.ListPeers(shardCollationTopic)
	res := &pb.RPCReply{
		Message: fmt.Sprintf("Shard %v peers: %v", shardID, shardPeers),
		Status:  true,
	}
	return res, nil
}

func (s *server) SubscribeShard(
	ctx context.Context,
	req *pb.RPCSubscribeShardReq) (*pb.RPCReply, error) {
	log.Printf("rpcserver:SubscribeShardReq: receive=%v", req)
	for _, shardID := range req.ShardIDs {
		s.node.ListenShard(shardID)
		time.Sleep(time.Millisecond * 30)
	}
	s.node.PublishListeningShards()
	res := &pb.RPCReply{
		Message: fmt.Sprintf(
			"Subscribed shard %v",
			req.ShardIDs,
		),
		Status: true,
	}
	return res, nil
}

func (s *server) UnsubscribeShard(
	ctx context.Context,
	req *pb.RPCUnsubscribeShardReq) (*pb.RPCReply, error) {
	log.Printf("rpcserver:UnsubscribeShardReq: receive=%v", req)
	for _, shardID := range req.ShardIDs {
		s.node.UnlistenShard(shardID)
		time.Sleep(time.Millisecond * 30)
	}
	s.node.PublishListeningShards()
	res := &pb.RPCReply{
		Message: fmt.Sprintf(
			"Unsubscribed shard %v",
			req.ShardIDs,
		),
		Status: true,
	}
	return res, nil
}

func (s *server) GetSubscribedShard(
	ctx context.Context,
	req *pb.RPCGetSubscribedShardReq) (*pb.RPCGetSubscribedShardReply, error) {
	log.Printf("rpcserver:GetSubscribedShard: receive=%v", req)
	shardIDs := s.node.GetListeningShards()
	res := &pb.RPCGetSubscribedShardReply{
		ShardIDs: shardIDs,
		Status:   true,
	}
	return res, nil
}

func (s *server) BroadcastCollation(
	ctx context.Context,
	req *pb.RPCBroadcastCollationReq) (*pb.RPCReply, error) {
	log.Printf("rpcserver:BroadcastCollationReq: receive=%v", req)
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
		s.node.broadcastCollation(
			ShardIDType(shardID),
			int64(i),
			string(randBytes),
		)
	}
	res := &pb.RPCReply{
		Message: fmt.Sprintf(
			"Finished sending %v size=%v collations in shard %v",
			numCollations,
			sizeInBytes,
			shardID,
		),
		Status: true,
	}
	return res, nil
}

func runRPCServer(n *Node, addr string) {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	s := grpc.NewServer()
	pb.RegisterPocServer(s, &server{node: n})
	log.Printf("%v: rpcserver: listening to %v", n.Name(), addr)
	s.Serve(lis)
}
