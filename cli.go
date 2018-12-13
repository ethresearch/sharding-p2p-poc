package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	peer "github.com/libp2p/go-libp2p-peer"

	pbrpc "github.com/ethresearch/sharding-p2p-poc/pb/rpc"
	"google.golang.org/grpc"
)

func doIdentify(rpcAddr string) {
	callIdentify(rpcAddr)
}

func doAddPeer(rpcArgs []string, rpcAddr string) {
	if len(rpcArgs) != 3 {
		logger.Fatal("Client: usage: addpeer ip port seed")
	}
	targetIP := rpcArgs[0]
	targetPort, err := strconv.Atoi(rpcArgs[1])
	if err != nil {
		logger.Fatalf("Failed to convert string '%v' to integer, err: %v", rpcArgs[1], err)
	}
	targetSeed, err := strconv.Atoi(rpcArgs[2])
	if err != nil {
		logger.Fatalf("Failed to convert string '%v' to integer, err: %v", rpcArgs[2], err)
	}
	callRPCAddPeer(rpcAddr, targetIP, targetPort, targetSeed)
}

func doSubShard(rpcArgs []string, rpcAddr string) {
	if len(rpcArgs) == 0 {
		logger.Fatalf("Client: usage: subshard shard0 shard1 ...")
	}
	shardIDs := []ShardIDType{}
	for _, shardIDString := range rpcArgs {
		shardID, err := strconv.ParseInt(shardIDString, 10, 64)
		if err != nil {
			logger.Fatalf(
				"Failed to convert string '%v' to integer, err: %v",
				shardIDString,
				err,
			)
		}
		shardIDs = append(shardIDs, shardID)
	}
	callRPCSubscribeShard(rpcAddr, shardIDs)
}

func doUnsubShard(rpcArgs []string, rpcAddr string) {
	if len(rpcArgs) == 0 {
		logger.Fatalf("Client: usage: unsubshard shard0 shard1 ...")
	}
	shardIDs := []ShardIDType{}
	for _, shardIDString := range rpcArgs {
		shardID, err := strconv.ParseInt(shardIDString, 10, 64)
		if err != nil {
			logger.Fatalf(
				"Failed to convert string '%v' to integer, err: %v",
				shardIDString,
				err,
			)
		}
		shardIDs = append(shardIDs, shardID)
	}
	callRPCUnsubscribeShard(rpcAddr, shardIDs)
}

func doBroadcastCollation(rpcArgs []string, rpcAddr string) {
	if len(rpcArgs) != 4 {
		logger.Fatal(
			"Client: usage: broadcastcollation shardID numCollations collationSize timeInMs",
		)
	}
	shardID, err := strconv.ParseInt(rpcArgs[0], 10, 64)
	if err != nil {
		logger.Fatalf("Invalid shard: %v", rpcArgs[0])
	}
	numCollations, err := strconv.Atoi(rpcArgs[1])
	if err != nil {
		logger.Fatalf("Invalid numCollations: %v", rpcArgs[1])
	}
	collationSize, err := strconv.Atoi(rpcArgs[2])
	if err != nil {
		logger.Fatalf("Invalid collationSize: %v", rpcArgs[2])
	}
	timeInMs, err := strconv.Atoi(rpcArgs[3])
	if err != nil {
		logger.Fatalf("Invalid timeInMs: %v", rpcArgs[3])
	}
	callRPCBroadcastCollation(
		rpcAddr,
		shardID,
		numCollations,
		collationSize,
		timeInMs,
	)
}

func doListTopicPeer(rpcArgs []string, rpcAddr string) {
	callRPCListTopicPeer(rpcAddr, rpcArgs)
}

func doListShardPeer(rpcArgs []string, rpcAddr string) {
	callRPCListShardPeer(rpcAddr, rpcArgs)
}

func doRemovePeer(rpcArgs []string, rpcAddr string) {
	if len(rpcArgs) != 1 {
		logger.Fatal("Client: usage: removepeer shardID")
	}
	peerIDString := rpcArgs[0]
	peerID, err := stringToPeerID(peerIDString)
	if err != nil {
		logger.Fatalf("Invalid peerID=%v, err: %v", peerIDString, err)
	}
	callRPCRemovePeer(rpcAddr, peerID)
}

func doBootstrap(rpcArgs []string, rpcAddr string) {
	if len(rpcArgs) == 0 || len(rpcArgs) > 2 {
		logger.Fatal("Client: usage: bootstrap start bootnodesStr")
	}
	flagStr := rpcArgs[0]
	var flag bool
	var bootnodesStr string
	if flagStr == "start" {
		flag = true
		if len(rpcArgs) == 2 {
			bootnodesStr = rpcArgs[1]
		}
	} else if flagStr == "stop" {
		flag = false
	} else {
		logger.Fatalf("Invalid flag: %v", flagStr)
	}
	callRPCBootstrap(rpcAddr, flag, bootnodesStr)
}

func callIdentify(rpcAddr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	identifyReq := &pbrpc.RPCIdentifyRequest{}
	logger.Debugf("rpcclient:Identify: sending=%v", identifyReq)
	res, err := client.Identify(context.Background(), identifyReq)
	if err != nil {
		logger.Fatalf("Failed to request identification from RPC server at %v, err: %v", rpcAddr, err)
	}
	fmt.Println(marshalToJSONString(res))
}

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
	res, err := client.AddPeer(context.Background(), addPeerReq)
	if err != nil {
		logger.Fatalf("Failed to add peer at %v:%v, err: %v", ipAddr, port, err)
	}
	logger.Debugf("rpcclient:AddPeer: result=%v", res)
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
	res, err := client.SubscribeShard(context.Background(), subscribeShardReq)
	if err != nil {
		logger.Fatalf("Failed to subscribe to shards %v, err: %v", shardIDs, err)
	}
	logger.Debugf("rpcclient:SubscribeShard: result=%v", res)
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
	res, err := client.UnsubscribeShard(context.Background(), unsubscribeShardReq)
	if err != nil {
		logger.Fatalf("Failed to unsubscribe shards %v, err: %v", shardIDs, err)
	}
	logger.Debugf("rpcclient:UnsubscribeShard: result=%v", res)
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
	res, err := client.GetSubscribedShard(context.Background(), getSubscribedShardReq)
	if err != nil {
		logger.Fatalf("Failed to get subscribed shards, err: %v", err)
	}
	logger.Debugf("rpcclient:GetSubscribedShard: result=%v", res.ShardIDs)
	shardIDs := make([]ShardIDType, 0)
	if res.ShardIDs != nil {
		shardIDs = res.ShardIDs
	}
	shardIDString := marshalToJSONString(shardIDs)
	fmt.Println(shardIDString)
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
	res, err := client.BroadcastCollation(context.Background(), broadcastCollationReq)
	if err != nil {
		logger.Fatalf(
			"Failed to broadcast %v collations of size %v in period %v in shard %v, err: %v",
			numCollations,
			collationSize,
			period,
			shardID,
			err,
		)
	}
	logger.Debugf("rpcclient:BroadcastCollation: result=%v", res)
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
	res, err := client.StopServer(context.Background(), stopServerReq)
	if err != nil {
		logger.Fatalf("Failed to stop RPC server at %v, err: %v", rpcAddr, err)
	}
	logger.Debugf("rpcclient:StopServer: result=%v", res)
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
	logger.Debugf("rpcclient:ListPeer: result=%v", res)
	peers := make([]string, 0)
	if res.Peers != nil {
		peers = res.Peers
	}
	peersString := marshalToJSONString(peers)
	fmt.Println(peersString)
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
	logger.Debugf("rpcclient:ListTopicPeer: result=%v", res)
	topicPeers := make(map[string][]string)
	for topic, peers := range res.TopicPeers {
		peerSlice := make([]string, 0)
		if peers.Peers != nil {
			peerSlice = peers.Peers
		}
		topicPeers[topic] = peerSlice
	}
	topicPeersString := marshalToJSONString(topicPeers)
	fmt.Println(topicPeersString)
}

func callRPCListShardPeer(rpcAddr string, shards []string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)

	topics := make([]string, len(shards))
	for i := 0; i < len(shards); i++ {
		shardID, err := strconv.ParseInt(shards[i], 10, 64)
		if err != nil {
			logger.Fatalf("Failed to convert shardID %v to int64, err: %v", shards[i], err)
		}
		topics[i] = getCollationsTopic(shardID)
	}

	listTopicPeerReq := &pbrpc.RPCListTopicPeerRequest{
		Topics: topics,
	}
	logger.Debugf("rpcclient:ListTopicPeer: sending=%v", listTopicPeerReq)
	res, err := client.ListTopicPeer(context.Background(), listTopicPeerReq)
	if err != nil {
		logger.Fatalf("Failed to call RPC ListTopicPeer at %v, err: %v", rpcAddr, err)
	}
	logger.Debugf("rpcclient:ListTopicPeer: result=%v", res)
	shardPeers := make(map[string][]string)
	for topic, peers := range res.TopicPeers {
		shard := shardTopicToShardID(topic)
		peerSlice := make([]string, 0)
		if peers.Peers != nil {
			peerSlice = peers.Peers
		}
		shardPeers[shard] = peerSlice
	}
	shardPeersString := marshalToJSONString(shardPeers)
	fmt.Println(shardPeersString)
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
	logger.Debugf("rpcclient:RemovePeer: sending=%v", removePeerReq)
	_, err = client.RemovePeer(context.Background(), removePeerReq)
	if err != nil {
		logger.Fatalf("Failed to call RPC removepeer at %v, err: %v", rpcAddr, err)
	}
}

func callRPCBootstrap(rpcAddr string, flag bool, bootnodesStr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		logger.Fatalf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	req := &pbrpc.RPCBootstrapRequest{
		Flag:         flag,
		BootnodesStr: bootnodesStr,
	}
	logger.Debugf("rpcclient:Bootstrap: sending=%v", req)
	_, err = client.Bootstrap(context.Background(), req)
	if err != nil {
		logger.Fatalf("Failed to call RPC bootstrap at %v, err: %v", rpcAddr, err)
	}
}

func marshalToJSONString(data interface{}) string {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		logger.Fatalf("failed to marshal object: %v, err: %v", data, err)
	}
	dataStr := string(dataBytes)
	return dataStr
}
