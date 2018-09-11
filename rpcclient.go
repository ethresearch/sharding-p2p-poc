package main

import (
	"context"
	"fmt"
	"log"

	pbrpc "github.com/ethresearch/sharding-p2p-poc/pb/rpc"
	peer "github.com/libp2p/go-libp2p-peer"
	"google.golang.org/grpc"
)

func exitIfNotSucceed(res *pbrpc.Response) {
	if res == nil || res.Status != pbrpc.Response_SUCCESS {
		log.Fatal()
	}
}

func callRPCAddPeer(rpcAddr string, ipAddr string, port int, seed int) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	addPeerReq := &pbrpc.RPCAddPeerRequest{
		Ip:   ipAddr,
		Port: PBInt(port),
		Seed: PBInt(seed),
	}
	log.Printf("rpcclient:AddPeer: sending=%v", addPeerReq)
	res, err := client.AddPeer(context.Background(), addPeerReq)
	if err != nil {
		log.Fatal(err)
	}
	exitIfNotSucceed(res.Response)
	log.Printf("rpcclient:AddPeer: result=%v", res)
}

func callRPCSubscribeShard(rpcAddr string, shardIDs []ShardIDType) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	subscribeShardReq := &pbrpc.RPCSubscribeShardRequest{
		ShardIDs: shardIDs,
	}
	log.Printf("rpcclient:ShardReq: sending=%v", subscribeShardReq)
	res, err := client.SubscribeShard(context.Background(), subscribeShardReq)
	if err != nil {
		log.Fatal(err)
	}
	exitIfNotSucceed(res.Response)
	log.Printf("rpcclient:ShardReq: result=%v", res)
}

func callRPCUnsubscribeShard(rpcAddr string, shardIDs []ShardIDType) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	unsubscribeShardReq := &pbrpc.RPCUnsubscribeShardRequest{
		ShardIDs: shardIDs,
	}
	log.Printf("rpcclient:UnsubscribeShardReq: sending=%v", unsubscribeShardReq)
	res, err := client.UnsubscribeShard(context.Background(), unsubscribeShardReq)
	if err != nil {
		log.Fatal(err)
	}
	exitIfNotSucceed(res.Response)
	log.Printf("rpcclient:UnsubscribeShardReq: result=%v", res)
}

func callRPCGetSubscribedShard(rpcAddr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	getSubscribedShardReq := &pbrpc.RPCGetSubscribedShardRequest{}
	log.Printf("rpcclient:GetSubscribedShard: sending=%v", getSubscribedShardReq)
	res, err := client.GetSubscribedShard(context.Background(), getSubscribedShardReq)
	if err != nil {
		log.Fatal(err)
	}
	exitIfNotSucceed(res.Response)
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
	client := pbrpc.NewPocClient(conn)
	broadcastCollationReq := &pbrpc.RPCBroadcastCollationRequest{
		ShardID: shardID,
		Number:  PBInt(numCollations),
		Size:    PBInt(collationSize),
		Period:  PBInt(period),
	}
	log.Printf("rpcclient:BroadcastCollation: sending=%v", broadcastCollationReq)
	res, err := client.BroadcastCollation(context.Background(), broadcastCollationReq)
	if err != nil {
		log.Fatal(err)
	}
	exitIfNotSucceed(res.Response)
	log.Printf("rpcclient:BroadcastCollation: result=%v", res)
}

func callRPCStopServer(rpcAddr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	stopServerReq := &pbrpc.RPCStopServerRequest{}
	log.Printf("rpcclient:StopServerReq: sending=%v", stopServerReq)
	res, err := client.StopServer(context.Background(), stopServerReq)
	if err != nil {
		log.Fatal(err)
	}
	exitIfNotSucceed(res.Response)
	fmt.Print(res)
}

// rpc GetConnection(RPCGetConnectionRequest) returns (RPCGetConnectionResponse) {}
// rpc GetPeer(RPCGetPeerRequest) returns (RPCGetPeerResponse) {}
// rpc SyncShardPeer(RPCSyncShardPeerRequest) returns (RPCSyncShardPeerResponse) {}
// rpc SyncCollation(RPCSyncCollationRequest) returns (RPCSyncCollationResponse) {}

func callRPCGetConnection(rpcAddr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	req := &pbrpc.RPCGetConnectionRequest{}
	res, err := client.GetConnection(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}
	exitIfNotSucceed(res.Response)
	fmt.Println(res)
}

func callRPCGetPeer(rpcAddr string, topic string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	req := &pbrpc.RPCGetPeerRequest{
		Topic: topic,
	}
	res, err := client.GetPeer(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}
	exitIfNotSucceed(res.Response)
	fmt.Println(res)
}

func callRPCSyncShardPeer(rpcAddr string, peerID peer.ID, shardIDs []ShardIDType) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	req := &pbrpc.RPCSyncShardPeerRequest{
		PeerID:   peerIDToString(peerID),
		ShardIDs: shardIDs,
	}
	res, err := client.SyncShardPeer(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}
	exitIfNotSucceed(res.Response)
	fmt.Println(res)
}

func callRPCSyncCollation(
	rpcAddr string,
	peerID peer.ID,
	shardID ShardIDType,
	period int,
	collationHash string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	req := &pbrpc.RPCSyncCollationRequest{
		PeerID:        peerIDToString(peerID),
		ShardID:       shardID,
		Period:        PBInt(period),
		CollationHash: collationHash,
	}
	res, err := client.SyncCollation(context.Background(), req)
	if err != nil {
		log.Fatal(err)
	}
	exitIfNotSucceed(res.Response)
	fmt.Println(res)
}
