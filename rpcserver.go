package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/golang/protobuf/proto"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"

	pbmsg "github.com/ethresearch/sharding-p2p-poc/pb/message"
	pbrpc "github.com/ethresearch/sharding-p2p-poc/pb/rpc"
	"google.golang.org/grpc"
)

type server struct {
	pbrpc.PocServer
	node              *Node
	serializedSpanCtx []byte
	rpcServer         *grpc.Server
}

func parseAddr(addrString string) (peer.ID, ma.Multiaddr, error) {
	// The following code extracts target's the peer ID from the
	// given multiaddress
	ipfsaddr, err := ma.NewMultiaddr(addrString) // ipfsaddr=/ip4/127.0.0.1/tcp/10000/ipfs/QmVmDaabYcS3pn23KaFjkdw6hkReUUma8sBKqSDHrPYPd2
	if err != nil {
		return "", nil, err
	}

	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS) // pid=QmVmDaabYcS3pn23KaFjkdw6hkReUUma8sBKqSDHrPYPd2
	if err != nil {
		return "", nil, err
	}

	peerid, err := peer.IDB58Decode(pid) // peerid=<peer.ID VmDaab>
	if err != nil {
		return "", nil, err
	}

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)),
	)
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)
	return peerid, targetAddr, nil
}

func (s *server) ShowPID(
	ctx context.Context,
	req *pbrpc.RPCShowPIDRequest) (*pbrpc.RPCShowPIDResponse, error) {
	// Add span for ShowPID
	spanctx, err := logger.StartFromParentState(ctx, "RPCServer.ShowPID", s.serializedSpanCtx)
	if err != nil {
		logger.Debugf("Failed to deserialze the trace context. Tracer won't be able to put rpc call traces together. err: %v", err)
		spanctx = logger.Start(ctx, "RPCServer.ShowPID")
	}
	defer logger.Finish(spanctx)

	logger.Debugf("rpcserver:ShowPIDRequest: receive=%v", req)
	res := &pbrpc.RPCShowPIDResponse{
		PeerID: s.node.ID().Pretty(),
	}
	logger.Debug("rpcserver:ShowPID: finished")
	return res, nil
}

func (s *server) AddPeer(
	ctx context.Context,
	req *pbrpc.RPCAddPeerRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for AddPeer of RPC Server
	spanctx, err := logger.StartFromParentState(ctx, "RPCServer.AddPeer", s.serializedSpanCtx)
	if err != nil {
		logger.Debugf("Failed to deserialze the trace context. Tracer won't be able to put rpc call traces together. err: %v", err)
		spanctx = logger.Start(ctx, "RPCServer.AddPeer")
	}
	defer logger.Finish(spanctx)

	logger.Debugf("rpcserver:AddPeer: ip=%v, port=%v, seed=%v", req.Ip, req.Port, req.Seed)
	_, targetPID, err := makeKey(int(req.Seed))
	mAddr := fmt.Sprintf(
		"/ip4/%s/tcp/%d/ipfs/%s",
		req.Ip,
		req.Port,
		targetPID.Pretty(),
	)
	if err != nil {
		errMsg := fmt.Errorf("Failed to generate peer key/ID with seed: %v, err: %v", req.Seed, err)
		logger.FinishWithErr(spanctx, errMsg)
		logger.Error(errMsg.Error())
		return nil, errMsg
	}

	peerid, targetAddr, err := parseAddr(mAddr)
	if err != nil {
		errMsg := fmt.Errorf("Failed to parse peer address: %s, err: %v", mAddr, err)
		logger.FinishWithErr(spanctx, errMsg)
		logger.Error(errMsg.Error())
		return nil, errMsg
	}
	s.node.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)

	if err := s.node.Connect(ctx, s.node.Peerstore().PeerInfo(peerid)); err != nil {
		errMsg := fmt.Errorf("Failed to connect to peer %v, err: %v", peerid, err)
		logger.FinishWithErr(spanctx, errMsg)
		logger.Error(errMsg)
		return nil, errMsg
	}

	// Tag the span with peer info
	logger.SetTag(spanctx, "Added peer", fmt.Sprintf("%v:%v", req.Ip, req.Port))
	logger.Debug("rpcserver:AddPeer: finished")
	return &pbrpc.RPCPlainResponse{}, nil
}

func (s *server) SubscribeShard(
	ctx context.Context,
	req *pbrpc.RPCSubscribeShardRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for SubscribeShard
	spanctx, err := logger.StartFromParentState(ctx, "RPCServer.SubscribeShard", s.serializedSpanCtx)
	if err != nil {
		logger.Debugf("Failed to deserialze the trace context. Tracer won't be able to put rpc call traces together. err: %v", err)
		spanctx = logger.Start(ctx, "RPCServer.SubscribeShard")
	}
	defer logger.Finish(spanctx)

	subscribedShardID := make([]int64, 0)
	logger.Debugf("rpcserver:SubscribeShard: shardIDs=%v", req.ShardIDs)
	for _, shardID := range req.ShardIDs {
		if err := s.node.ListenShard(spanctx, shardID); err != nil {
			logger.SetErr(spanctx, fmt.Errorf("Failed to listen to shard %v", shardID))
			logger.Errorf("Failed to listen to shard %v", shardID)
		} else {
			subscribedShardID = append(subscribedShardID, shardID)
		}
		time.Sleep(time.Millisecond * 30)
	}
	// Tag the span with shardIDs which are successfully subscribed to
	logger.SetTag(spanctx, "ShardIDs", fmt.Sprintf("%v", subscribedShardID))
	logger.Debug("rpcserver:SubscribeShard: finished")
	return &pbrpc.RPCPlainResponse{}, nil
}

func (s *server) UnsubscribeShard(
	ctx context.Context,
	req *pbrpc.RPCUnsubscribeShardRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for UnsubscribeShard
	spanctx, err := logger.StartFromParentState(ctx, "RPCServer.UnsubscribeShard", s.serializedSpanCtx)
	if err != nil {
		logger.Debugf("Failed to deserialze the trace context. Tracer won't be able to put rpc call traces together. err: %v", err)
		spanctx = logger.Start(ctx, "RPCServer.UnsubscribeShard")
	}
	defer logger.Finish(spanctx)

	unsubscribedShardID := make([]int64, 0)
	logger.Debugf("rpcserver:UnsubscribeShard: shardIDs=%v", req.ShardIDs)
	for _, shardID := range req.ShardIDs {
		if err := s.node.UnlistenShard(spanctx, shardID); err != nil {
			logger.SetErr(spanctx, fmt.Errorf("Failed to unlisten shard %v", shardID))
			logger.Errorf("Failed to unlisten shard %v", shardID)
		} else {
			unsubscribedShardID = append(unsubscribedShardID, shardID)
		}
		time.Sleep(time.Millisecond * 30)
	}
	// Tag the span with shardIDs which are successfully unsubscribed from
	logger.SetTag(spanctx, "ShardIDs", fmt.Sprintf("shard %v", unsubscribedShardID))
	logger.Debug("rpcserver:UnsubscribeShard: finished")
	return &pbrpc.RPCPlainResponse{}, nil
}

func (s *server) GetSubscribedShard(
	ctx context.Context,
	req *pbrpc.RPCGetSubscribedShardRequest) (*pbrpc.RPCGetSubscribedShardResponse, error) {
	// Add span for GetSubscribedShard
	spanctx, err := logger.StartFromParentState(ctx, "RPCServer.GetSubscribedShard", s.serializedSpanCtx)
	if err != nil {
		logger.Debugf("Failed to deserialze the trace context. Tracer won't be able to put rpc call traces together. err: %v", err)
		spanctx = logger.Start(ctx, "RPCServer.GetSubscribedShard")
	}
	defer logger.Finish(spanctx)

	logger.Debug("rpcserver:GetSubscribedShard")
	shardIDs := s.node.GetListeningShards()
	res := &pbrpc.RPCGetSubscribedShardResponse{
		ShardIDs: shardIDs,
	}
	// Tag the span with shardIDs returned
	logger.SetTag(spanctx, "shardIDs", shardIDs)
	return res, nil
}

// This is the BroadcastCollation for testing purpose.
// Given ShardID, Number and Size,
// it broadcasts Number collations of shard ShardID each with size Size bytes.
func (s *server) BroadcastCollation(
	ctx context.Context,
	req *pbrpc.RPCBroadcastCollationRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for BroadcastCollation
	spanctx, err := logger.StartFromParentState(ctx, "RPCServer.BroadcastCollation", s.serializedSpanCtx)
	if err != nil {
		logger.Debugf("Failed to deserialze the trace context. Tracer won't be able to put rpc call traces together. err: %v", err)
		spanctx = logger.Start(ctx, "RPCServer.BroadcastCollation")
	}
	defer logger.Finish(spanctx)

	shardID := req.ShardID
	numCollations := int(req.Number)
	sizeInBytes := req.Size
	if sizeInBytes > 100 {
		sizeInBytes -= 100
	}
	timeInMs := req.Period
	logger.Debugf(
		"rpcserver:BroadcastCollation: broadcasting: shardID=%v, numCollations=%v, sizeInBytes=%v, timeInMs=%v",
		shardID,
		numCollations,
		sizeInBytes,
		timeInMs,
	)
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
			errMsg := fmt.Errorf("Failed to broadcast collation, err: %v", err)
			logger.SetErr(spanctx, errMsg)
			logger.Error(errMsg.Error())
			return nil, errMsg
		}
	}
	// Tag the span with collations info if nothing goes wrong
	logger.SetTag(spanctx, "shardID", req.ShardID)
	logger.SetTag(spanctx, "numCollations", numCollations)
	logger.SetTag(spanctx, "sizeInBytes", sizeInBytes)
	logger.Debug("rpcserver:BroadcastCollation: finished")
	return &pbrpc.RPCPlainResponse{}, nil
}

// This is the real BroadcastCollation.
// TODO: Replace BroadcastCollation with this one.
func (s *server) SendCollation(
	ctx context.Context,
	req *pbrpc.RPCSendCollationRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for SendCollation
	spanctx, err := logger.StartFromParentState(ctx, "RPCServer.SendCollation", s.serializedSpanCtx)
	if err != nil {
		logger.Debugf("Failed to deserialze the trace context. Tracer won't be able to put rpc call traces together. err: %v", err)
		spanctx = logger.Start(ctx, "RPCServer.SendCollation")
	}
	defer logger.Finish(spanctx)

	collation := req.Collation
	logger.Debugf(
		"rpcserver:SendCollationRequest: shardID=%v, period=%v",
		collation.ShardID,
		collation.Period,
	)
	err = s.node.broadcastCollationMessage(collation)
	if err != nil {
		errMsg := fmt.Errorf("Failed to broadcast collation message, err: %v", err)
		logger.FinishWithErr(spanctx, errMsg)
		logger.Error(errMsg.Error())
		return nil, errMsg
	}
	// Tag collation info if nothing goes wrong
	logger.SetTag(spanctx, "Shard", collation.ShardID)
	logger.SetTag(spanctx, "Period of collation", collation.Period)
	logger.SetTag(spanctx, "Blobs of collation", collation.Blobs)
	logger.Debug("rpcserver:SendCollation: finished")
	return &pbrpc.RPCPlainResponse{}, nil
}

func (s *server) StopServer(
	ctx context.Context,
	req *pbrpc.RPCStopServerRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for StopServer
	spanctx, err := logger.StartFromParentState(ctx, "RPCServer.StopServer", s.serializedSpanCtx)
	if err != nil {
		logger.Debugf("Failed to deserialze the trace context. Tracer won't be able to put rpc call traces together. err: %v", err)
		spanctx = logger.Start(ctx, "RPCServer.StopServer")
	}
	defer logger.Finish(spanctx)

	logger.Debug("rpcserver:StopServer")
	go func() {
		time.Sleep(time.Second * 1)
		logger.Info("Closing RPC server by rpc call...")
		s.rpcServer.Stop()
	}()
	logger.Debug("rpcserver:StopServer: finished")
	return &pbrpc.RPCPlainResponse{}, nil
}

func (s *server) Send(ctx context.Context, req *pbrpc.SendRequest) (*pbrpc.SendResponse, error) {
	// Add span for Send
	spanctx, err := logger.StartFromParentState(ctx, "RPCServer.Send", s.serializedSpanCtx)
	if err != nil {
		logger.FinishWithErr(
			spanctx,
			fmt.Errorf("Failed to deserialize parent span context, err: %v", err),
		)
	}
	defer logger.Finish(spanctx)

	logger.Debugf("rpcserver:Send: msgType=%v, data=%v", req.MsgType, req.Data)
	if req.PeerID == "" {
		typedMessage := &pbmsg.MessageWithType{
			MsgType: req.MsgType,
			Data:    req.Data,
		}
		msgBytes, err := proto.Marshal(typedMessage)
		if err != nil {
			errMsg := fmt.Errorf(
				"failed to marshall typedMessage %v. reason: %v",
				typedMessage,
				err,
			)
			logger.FinishWithErr(spanctx, errMsg)
			logger.Error(errMsg.Error())
			return nil, errMsg
		}
		err = s.node.pubsubService.Publish(req.Topic, msgBytes)
		if err != nil {
			errMsg := fmt.Errorf(
				"failed to publish %v bytes in topic %v. reason: %v",
				len(req.Data),
				req.Topic,
				err,
			)
			logger.FinishWithErr(spanctx, errMsg)
			logger.Error(errMsg.Error())
			return nil, errMsg
		}
		return &pbrpc.SendResponse{}, nil
	}
	// direct request
	peerID, err := peer.IDB58Decode(req.PeerID)
	if err != nil {
		return nil, fmt.Errorf("invalid peerID %v", peerID)
	}
	dataBytes, err := s.node.generalRequest(ctx, peerID, int(req.MsgType), req.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to make request to peer %v", peerID)
	}
	logger.Debug("rpcserver:Send: finished")
	return &pbrpc.SendResponse{
		Data: dataBytes,
	}, nil
}

func (s *server) ListPeer(
	ctx context.Context,
	req *pbrpc.RPCListPeerRequest) (*pbrpc.RPCListPeerResponse, error) {
	logger.Debug("rpcserver:ListPeer")
	peerIDs := s.node.Network().Peers()
	return &pbrpc.RPCListPeerResponse{
		Peers: peerIDsToPeersString(peerIDs),
	}, nil
}

func (s *server) ListTopicPeer(
	ctx context.Context,
	req *pbrpc.RPCListTopicPeerRequest) (*pbrpc.RPCListTopicPeerResponse, error) {
	logger.Debugf("rpcserver:ListTopicPeer: topics=%v", req.Topics)
	var topics []string
	if len(req.Topics) == 0 {
		topics = s.node.pubsubService.GetTopics()
	} else {
		topics = req.Topics
	}
	topicPeers := make(map[string]*pbmsg.Peers)
	for _, topic := range topics {
		peerIDs := s.node.pubsubService.ListPeers(topic)
		topicPeers[topic] = peerIDsToPBPeers(peerIDs)
	}
	return &pbrpc.RPCListTopicPeerResponse{
		TopicPeers: topicPeers,
	}, nil
}

func (s *server) RemovePeer(
	ctx context.Context,
	req *pbrpc.RPCRemovePeerRequest) (*pbrpc.RPCPlainResponse, error) {
	// Add span for RemovePeer
	spanctx, err := logger.StartFromParentState(ctx, "RPCServer.RemovePeer", s.serializedSpanCtx)
	if err != nil {
		logger.Debugf("Failed to deserialze the trace context. Tracer won't be able to put rpc call traces together. err: %v", err)
		spanctx = logger.Start(ctx, "RPCServer.RemovePeer")
	}
	defer logger.Finish(spanctx)

	logger.Debugf("rpcserver:RemovePeer: peerID=%v", req.PeerID)

	peerID, err := stringToPeerID(req.PeerID)
	if err != nil {
		errMsg := fmt.Errorf("failed to parse the peerID: %v", req.PeerID)
		logger.FinishWithErr(spanctx, errMsg)
		logger.Error(errMsg.Error())
		return nil, errMsg
	}
	err = s.node.Network().ClosePeer(peerID)
	if err != nil {
		errMsg := fmt.Errorf("failed to close the connection to peer: %v", err)
		logger.FinishWithErr(spanctx, errMsg)
		logger.Error(errMsg.Error())
		return nil, errMsg
	}

	// TODO: consider the record in Peerstore
	logger.Debug("rpcserver:RemovePeer: finished")
	return &pbrpc.RPCPlainResponse{}, nil
}

func (s *server) Bootstrap(
	ctx context.Context,
	req *pbrpc.RPCBootstrapRequest) (*pbrpc.RPCPlainResponse, error) {
	spanctx, err := logger.StartFromParentState(ctx, "RPCServer.Bootstrap", s.serializedSpanCtx)
	if err != nil {
		logger.Debugf("Failed to deserialze the trace context. Tracer won't be able to put rpc call traces together. err: %v", err)
		spanctx = logger.Start(ctx, "RPCServer.Bootstrap")
	}
	defer logger.Finish(spanctx)

	logger.Debugf("rpcserver:Bootstrap: flag=%v, bootnodes=%v", req.Flag, req.BootnodesStr)
	if req.Flag {
		var bootnodes = []pstore.PeerInfo{}
		bootnodes, err = convertPeers(strings.Split(req.BootnodesStr, ","))
		if err != nil {
			errMsg := fmt.Errorf(
				"failed to convert bootnode address: %v, to peer info format, err: %v",
				req.BootnodesStr,
				err,
			)
			logger.FinishWithErr(spanctx, errMsg)
			logger.Error(errMsg.Error())
			return nil, errMsg
		}
		err = s.node.StartBootstrapping(s.node.ctx, bootnodes)
		if err != nil {
			errMsg := fmt.Errorf("failed to start bootstrapping: %v", err)
			logger.FinishWithErr(spanctx, errMsg)
			logger.Error(errMsg.Error())
			return nil, errMsg
		}
		logger.Debug("rpcserver:Bootstrap: started bootstrapping")
	} else {
		err = s.node.StopBootstrapping()
		if err != nil {
			errMsg := fmt.Errorf("failed to stop bootstrapping: %v", err)
			logger.FinishWithErr(spanctx, errMsg)
			logger.Error(errMsg.Error())
			return nil, errMsg
		}
		logger.Debug("rpcserver:Bootstrap: stopped bootstrapping")
	}
	logger.Debug("rpcserver:Bootstrap: finished")
	return &pbrpc.RPCPlainResponse{}, nil
}

func runRPCServer(n *Node, addr string) {
	// logging.SetLogLevel("sharding-p2p", "DEBUG")
	// Start a new trace
	ctx := context.Background()
	ctx = logger.Start(ctx, "RPCServer")
	defer logger.Finish(ctx)
	logger.SetTag(ctx, "Node ID %s", n.host.ID().Pretty())
	serializedSpanCtx, err := logger.SerializeContext(ctx)
	if err != nil {
		logger.FinishWithErr(ctx, fmt.Errorf("Failed to serialize span context, err: %v", err))
		logger.Debugf("Failed to serialze the trace context. Tracer won't be able to put rpc call traces together, err: %v", err)
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.FinishWithErr(ctx, fmt.Errorf("Failed to set up a service listening on %s, err: %v", addr, err))
		logger.Fatalf("Failed to set up a service listening on %s, err: %v", addr, err)
	}
	s := grpc.NewServer()
	pbrpc.RegisterPocServer(s, &server{node: n, serializedSpanCtx: serializedSpanCtx, rpcServer: s})

	// Catch interupt signal
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Info("Closing RPC server by Interrupt signal...")
		s.Stop()
	}()

	logger.Infof("RPC server listening to address: %v", addr)
	if err := s.Serve(lis); err != nil {
		logger.FinishWithErr(ctx, fmt.Errorf("Failed to serve the RPC server, err: %v", err))
		logger.Fatalf("Failed to serve the RPC server, err: %v", err)
	}
}
