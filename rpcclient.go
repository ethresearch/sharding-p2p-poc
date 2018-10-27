package main

import (
	"context"
	"fmt"
	peer "github.com/libp2p/go-libp2p-peer"

	pbrpc "github.com/ethresearch/sharding-p2p-poc/pb/rpc"
	"google.golang.org/grpc"
)

func callRPCAddPeer(rpcAddr string, ipAddr string, port int, seed int) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	addPeerReq := &pbrpc.RPCAddPeerRequest{
		Ip:   ipAddr,
		Port: PBInt(port),
		Seed: PBInt(seed),
	}
	logger.Debugf("rpcclient:AddPeer: sending=%v", addPeerReq)
	if res, err := client.AddPeer(context.Background(), addPeerReq); err != nil {
		logger.Fatalf("Failed to add peer at %v:%v, err: %v", ipAddr, port, err)
	} else {
		logger.Debugf("rpcclient:AddPeer: result=%v", res)
	}
}

func callRPCSubscribeShard(rpcAddr string, shardIDs []ShardIDType) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	subscribeShardReq := &pbrpc.RPCSubscribeShardRequest{
		ShardIDs: shardIDs,
	}
	logger.Debugf("rpcclient:SubscribeShard: sending=%v", subscribeShardReq)
	if res, err := client.SubscribeShard(context.Background(), subscribeShardReq); err != nil {
		logger.Fatalf("Failed to subscribe to shards %v, err: %v", shardIDs, err)
	} else {
		logger.Debugf("rpcclient:SubscribeShard: result=%v", res)
	}
}

func callRPCUnsubscribeShard(rpcAddr string, shardIDs []ShardIDType) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	unsubscribeShardReq := &pbrpc.RPCUnsubscribeShardRequest{
		ShardIDs: shardIDs,
	}
	logger.Debugf("rpcclient:UnsubscribeShard: sending=%v", unsubscribeShardReq)
	if res, err := client.UnsubscribeShard(context.Background(), unsubscribeShardReq); err != nil {
		logger.Fatalf("Failed to unsubscribe shards %v, err: %v", shardIDs, err)
	} else {
		logger.Debugf("rpcclient:UnsubscribeShard: result=%v", res)
	}
}

func callRPCGetSubscribedShard(rpcAddr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	getSubscribedShardReq := &pbrpc.RPCGetSubscribedShardRequest{}
	logger.Debugf("rpcclient:GetSubscribedShard: sending=%v", getSubscribedShardReq)
	if res, err := client.GetSubscribedShard(context.Background(), getSubscribedShardReq); err != nil {
		logger.Fatalf("Failed to get subscribed shards, err: %v", err)
	} else {
		logger.Debugf("rpcclient:GetSubscribedShard: result=%v", res.ShardIDs)
	}
}

func callRPCBroadcastCollation(
	rpcAddr string,
	shardID ShardIDType,
	numCollations int,
	collationSize int,
	period int) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	broadcastCollationReq := &pbrpc.RPCBroadcastCollationRequest{
		ShardID: shardID,
		Number:  PBInt(numCollations),
		Size:    PBInt(collationSize),
		Period:  PBInt(period),
	}
	logger.Debugf("rpcclient:BroadcastCollation: sending=%v", broadcastCollationReq)
	if res, err := client.BroadcastCollation(context.Background(), broadcastCollationReq); err != nil {
		logger.Fatalf("Failed to broadcast %v collations of size %v in period %v in shard %v, err: %v", numCollations, collationSize, period, shardID, err)
	} else {
		logger.Debugf("rpcclient:BroadcastCollation: result=%v", res)
	}
}

func callRPCStopServer(rpcAddr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	stopServerReq := &pbrpc.RPCStopServerRequest{}
	logger.Debugf("rpcclient:StopServer: sending=%v", stopServerReq)
	if res, err := client.StopServer(context.Background(), stopServerReq); err != nil {
		logger.Fatalf("Failed to stop RPC server at %v, err: %v", rpcAddr, err)
	} else {
		logger.Debugf("rpcclient:StopServer: result=%v", res)
	}
}

func callRPCListPeer(rpcAddr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	listPeerReq := &pbrpc.RPCListPeerRequest{}
	logger.Debugf("rpcclient:ListPeer: sending=%v", listPeerReq)
	res, err := client.ListPeer(context.Background(), listPeerReq)
	if err != nil {
		logger.Fatalf("Failed to call RPC ListPeer at %v, err: %v", rpcAddr, err)
	}
	fmt.Println(res)
}

func callRPCListTopicPeer(rpcAddr string, topics []string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	listTopicPeerReq := &pbrpc.RPCListTopicPeerRequest{
		Topics: topics,
	}
	logger.Debugf("rpcclient:ListTopicPeer: sending=%v", listTopicPeerReq)
	res, err := client.ListTopicPeer(context.Background(), listTopicPeerReq)
	if err != nil {
		logger.Fatalf("Failed to call RPC ListTopicPeer at %v, err: %v", rpcAddr, err)
	}
	fmt.Println(res)
}

func callRPCRemovePeer(rpcAddr string, peerID peer.ID) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	removePeerReq := &pbrpc.RPCRemovePeerRequest{
		PeerID: peerIDToString(peerID),
	}
	logger.Debugf("rpcclient:removePeerReq: sending=%v", removePeerReq)
	res, err := client.RemovePeer(context.Background(), removePeerReq)
	if err != nil {
		logger.Fatalf("Failed to call RPC listpeer at %v, err: %v", rpcAddr, err)
	}
	fmt.Println(res)
}
