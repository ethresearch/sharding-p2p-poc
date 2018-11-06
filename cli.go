package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	peer "github.com/libp2p/go-libp2p-peer"

	pbrpc "github.com/ethresearch/sharding-p2p-poc/pb/rpc"
	"google.golang.org/grpc"
)

func doAddPeer(rpcArgs []string, rpcAddr string) {
	if len(rpcArgs) != 3 {
		errMsg := fmt.Sprintf("Client: usage: addpeer ip port seed")
		emitCLIFailure(errMsg)
	}
	targetIP := rpcArgs[0]
	targetPort, err := strconv.Atoi(rpcArgs[1])
	if err != nil {
		errMsg := fmt.Sprintf("Failed to convert string '%v' to integer, err: %v", rpcArgs[1], err)
		emitCLIFailure(errMsg)
	}
	targetSeed, err := strconv.Atoi(rpcArgs[2])
	if err != nil {
		errMsg := fmt.Sprintf("Failed to convert string '%v' to integer, err: %v", rpcArgs[2], err)
		emitCLIFailure(errMsg)
	}
	callRPCAddPeer(rpcAddr, targetIP, targetPort, targetSeed)
}

func doSubShard(rpcArgs []string, rpcAddr string) {
	if len(rpcArgs) == 0 {
		errMsg := fmt.Sprintf("Client: usage: subshard shard0 shard1 ...")
		emitCLIFailure(errMsg)
	}
	shardIDs := []ShardIDType{}
	for _, shardIDString := range rpcArgs {
		shardID, err := strconv.ParseInt(shardIDString, 10, 64)
		if err != nil {
			errMsg := fmt.Sprintf(
				"Failed to convert string '%v' to integer, err: %v",
				shardIDString,
				err,
			)
			emitCLIFailure(errMsg)
		}
		shardIDs = append(shardIDs, shardID)
	}
	callRPCSubscribeShard(rpcAddr, shardIDs)
}

func doUnsubShard(rpcArgs []string, rpcAddr string) {
	if len(rpcArgs) == 0 {
		errMsg := fmt.Sprintf("Client: usage: unsubshard shard0 shard1 ...")
		emitCLIFailure(errMsg)
	}
	shardIDs := []ShardIDType{}
	for _, shardIDString := range rpcArgs {
		shardID, err := strconv.ParseInt(shardIDString, 10, 64)
		if err != nil {
			errMsg := fmt.Sprintf(
				"Failed to convert string '%v' to integer, err: %v",
				shardIDString,
				err,
			)
			emitCLIFailure(errMsg)
		}
		shardIDs = append(shardIDs, shardID)
	}
	callRPCUnsubscribeShard(rpcAddr, shardIDs)
}

func doBroadcastCollation(rpcArgs []string, rpcAddr string) {
	if len(rpcArgs) != 4 {
		emitCLIFailure(
			"Client: usage: broadcastcollation shardID numCollations collationSize timeInMs",
		)
	}
	shardID, err := strconv.ParseInt(rpcArgs[0], 10, 64)
	if err != nil {
		errMsg := fmt.Sprintf("Invalid shard: %v", rpcArgs[0])
		emitCLIFailure(errMsg)
	}
	numCollations, err := strconv.Atoi(rpcArgs[1])
	if err != nil {
		errMsg := fmt.Sprintf("Invalid numCollations: %v", rpcArgs[1])
		emitCLIFailure(errMsg)
	}
	collationSize, err := strconv.Atoi(rpcArgs[2])
	if err != nil {
		errMsg := fmt.Sprintf("Invalid collationSize: %v", rpcArgs[2])
		emitCLIFailure(errMsg)
	}
	timeInMs, err := strconv.Atoi(rpcArgs[3])
	if err != nil {
		errMsg := fmt.Sprintf("Invalid timeInMs: %v", rpcArgs[3])
		emitCLIFailure(errMsg)
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

func doRemovePeer(rpcArgs []string, rpcAddr string) {
	if len(rpcArgs) != 1 {
		emitCLIFailure("Client: usage: removepeer shardID")
	}
	peerIDString := rpcArgs[0]
	peerID, err := stringToPeerID(peerIDString)
	if err != nil {
		errMsg := fmt.Sprintf("Invalid peerID=%v, err: %v", peerIDString, err)
		emitCLIFailure(errMsg)
	}
	callRPCRemovePeer(rpcAddr, peerID)
}

func callRPCAddPeer(rpcAddr string, ipAddr string, port int, seed int) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
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
		errMsg := fmt.Sprintf("Failed to add peer at %v:%v, err: %v", ipAddr, port, err)
		emitCLIFailure(errMsg)
	} else {
		logger.Debugf("rpcclient:AddPeer: result=%v", res)
	}
	emitCLIResponse(emptyResponse)
}

func callRPCSubscribeShard(rpcAddr string, shardIDs []ShardIDType) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	subscribeShardReq := &pbrpc.RPCSubscribeShardRequest{
		ShardIDs: shardIDs,
	}
	logger.Debugf("rpcclient:SubscribeShard: sending=%v", subscribeShardReq)
	if res, err := client.SubscribeShard(context.Background(), subscribeShardReq); err != nil {
		errMsg := fmt.Sprintf("Failed to subscribe to shards %v, err: %v", shardIDs, err)
		emitCLIFailure(errMsg)
	} else {
		logger.Debugf("rpcclient:SubscribeShard: result=%v", res)
	}
	emitCLIResponse(emptyResponse)
}

func callRPCUnsubscribeShard(rpcAddr string, shardIDs []ShardIDType) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	unsubscribeShardReq := &pbrpc.RPCUnsubscribeShardRequest{
		ShardIDs: shardIDs,
	}
	logger.Debugf("rpcclient:UnsubscribeShard: sending=%v", unsubscribeShardReq)
	if res, err := client.UnsubscribeShard(context.Background(), unsubscribeShardReq); err != nil {
		errMsg := fmt.Sprintf("Failed to unsubscribe shards %v, err: %v", shardIDs, err)
		emitCLIFailure(errMsg)
	} else {
		logger.Debugf("rpcclient:UnsubscribeShard: result=%v", res)
	}
	emitCLIResponse(emptyResponse)
}

func callRPCGetSubscribedShard(rpcAddr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	getSubscribedShardReq := &pbrpc.RPCGetSubscribedShardRequest{}
	logger.Debugf("rpcclient:GetSubscribedShard: sending=%v", getSubscribedShardReq)
	if res, err := client.GetSubscribedShard(context.Background(), getSubscribedShardReq); err != nil {
		errMsg := fmt.Sprintf("Failed to get subscribed shards, err: %v", err)
		emitCLIFailure(errMsg)
	} else {
		logger.Debugf("rpcclient:GetSubscribedShard: result=%v", res.ShardIDs)
		shardResp := &ShardResponse{
			ShardIDs: res.ShardIDs,
		}
		emitCLIResponse(shardResp)
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
		errMsg := fmt.Sprintf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
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
		errMsg := fmt.Sprintf(
			"Failed to broadcast %v collations of size %v in period %v in shard %v, err: %v",
			numCollations,
			collationSize,
			period,
			shardID,
			err,
		)
		emitCLIFailure(errMsg)
	} else {
		logger.Debugf("rpcclient:BroadcastCollation: result=%v", res)
	}
	emitCLIResponse(emptyResponse)
}

func callRPCStopServer(rpcAddr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	stopServerReq := &pbrpc.RPCStopServerRequest{}
	logger.Debugf("rpcclient:StopServer: sending=%v", stopServerReq)
	if res, err := client.StopServer(context.Background(), stopServerReq); err != nil {
		errMsg := fmt.Sprintf("Failed to stop RPC server at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
	} else {
		logger.Debugf("rpcclient:StopServer: result=%v", res)
	}
	emitCLIResponse(emptyResponse)
}

func callRPCListPeer(rpcAddr string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	listPeerReq := &pbrpc.RPCListPeerRequest{}
	logger.Debugf("rpcclient:ListPeer: sending=%v", listPeerReq)
	res, err := client.ListPeer(context.Background(), listPeerReq)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to call RPC ListPeer at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
	} else {
		logger.Debugf("rpcclient:ListPeer: result=%v", res)
	}
	peers := &PeerResponse{
		Peers: res.Peers.Peers,
	}
	emitCLIResponse(peers)
}

func callRPCListTopicPeer(rpcAddr string, topics []string) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	listTopicPeerReq := &pbrpc.RPCListTopicPeerRequest{
		Topics: topics,
	}
	logger.Debugf("rpcclient:ListTopicPeer: sending=%v", listTopicPeerReq)
	res, err := client.ListTopicPeer(context.Background(), listTopicPeerReq)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to call RPC ListTopicPeer at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
	} else {
		logger.Debugf("rpcclient:ListTopicPeer: result=%v", res)
	}
	topicPeers := make(map[string][]string)
	for topic, peers := range res.TopicPeers {
		topicPeers[topic] = peers.Peers
	}
	emitCLIResponse(topicPeers)
}

func callRPCRemovePeer(rpcAddr string, peerID peer.ID) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect to RPC server at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
	}
	defer conn.Close()
	client := pbrpc.NewPocClient(conn)
	removePeerReq := &pbrpc.RPCRemovePeerRequest{
		PeerID: peerIDToString(peerID),
	}
	logger.Debugf("rpcclient:removePeerReq: sending=%v", removePeerReq)
	_, err = client.RemovePeer(context.Background(), removePeerReq)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to call RPC listpeer at %v, err: %v", rpcAddr, err)
		emitCLIFailure(errMsg)
	}
	emitCLIResponse(emptyResponse)
}

type CLIResponse struct {
	Status bool
	Data   interface{}
}

type CLIFailure struct {
	Status  bool
	Message string
}

type ShardResponse struct {
	ShardIDs []ShardIDType
}

type PeerResponse struct {
	Peers []string
}

type TopicPeerResponse struct {
	TopicPeers map[string]([]string)
}

var emptyResponse interface{}

func marshalToJSONString(data interface{}) (string, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal object: %v, err: %v", data, err)
	}
	dataStr := string(dataBytes)
	return dataStr, nil
}

func emitCLIFailure(msg string) {
	resp := &CLIFailure{
		Status:  false,
		Message: msg,
	}
	respStr, err := marshalToJSONString(resp)
	if err != nil {
		logger.Error(err)
	}
	fmt.Println(respStr)
	os.Exit(1)
}

func emitCLIResponse(data interface{}) {
	resp := &CLIResponse{
		Status: true,
		Data:   data,
	}
	respStr, err := marshalToJSONString(resp)
	if err != nil {
		logger.Error(err)
	}
	fmt.Println(respStr)
}
