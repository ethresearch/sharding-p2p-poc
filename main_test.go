package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	pbevent "github.com/ethresearch/sharding-p2p-poc/pb/event"
	pbmsg "github.com/ethresearch/sharding-p2p-poc/pb/message"
	host "github.com/libp2p/go-libp2p-host"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
	// gologging "github.com/whyrusleeping/go-logging"
)

var nodeCount int

func makeUnbootstrappedNode(t *testing.T, ctx context.Context, number int) *Node {
	return makeTestingNode(t, ctx, number, false, nil)
}

func makeTestingNode(
	t *testing.T,
	ctx context.Context,
	number int,
	doBootstrapping bool,
	bootstrapPeers []pstore.PeerInfo) *Node {
	// FIXME:
	//		1. Use 20000 to avoid conflitcs with the running nodes in the environment
	//		2. Avoid reuse of listeningPort in the entire test, to avoid `dial error`s
	listeningPort := 20000 + nodeCount
	nodeCount++
	node, err := makeNode(ctx, listeningPort, number, nil, doBootstrapping, bootstrapPeers)
	if err != nil {
		t.Error("Failed to create node")
	}
	return node
}

func TestNodeListeningShards(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node := makeUnbootstrappedNode(t, ctx, 0)
	var testingShardID ShardIDType = 42
	// test `IsShardListened`
	if node.IsShardListened(testingShardID) {
		t.Errorf("Shard %v haven't been listened", testingShardID)
	}
	// test `ListenShard`
	node.ListenShard(ctx, testingShardID)
	node.PublishListeningShards(ctx)
	if !node.IsShardListened(testingShardID) {
		t.Errorf("Shard %v should have been listened", testingShardID)
	}
	if !node.IsCollationSubscribed(testingShardID) {
		t.Errorf(
			"shardCollations in shard [%v] should be subscribed",
			testingShardID,
		)
	}
	anotherShardID := testingShardID + 1
	node.ListenShard(ctx, anotherShardID)
	node.PublishListeningShards(ctx)
	if !node.IsShardListened(anotherShardID) {
		t.Errorf("Shard %v should have been listened", anotherShardID)
	}
	shardIDs := node.GetListeningShards()
	if len(shardIDs) != 2 {
		t.Errorf("We should have 2 shards being listened, instead of %v", len(shardIDs))
	}
	// test `UnlistenShard`
	node.UnlistenShard(ctx, testingShardID)
	node.PublishListeningShards(ctx)
	if node.IsShardListened(testingShardID) {
		t.Errorf("Shard %v should have been unlistened", testingShardID)
	}
	if node.IsCollationSubscribed(testingShardID) {
		t.Errorf(
			"shardCollations in shard [%v] should be already unsubscribed",
			testingShardID,
		)
	}
	node.UnlistenShard(ctx, testingShardID) // ensure no side effect
	node.PublishListeningShards(ctx)
}

func connect(t *testing.T, ctx context.Context, a, b host.Host) {
	a.Peerstore().AddAddrs(b.ID(), b.Addrs(), pstore.PermanentAddrTTL)
	b.Peerstore().AddAddrs(a.ID(), a.Addrs(), pstore.PermanentAddrTTL)
	pinfo := pstore.PeerInfo{
		ID:    a.ID(),
		Addrs: a.Addrs(),
	}
	err := b.Connect(ctx, pinfo)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Network().ConnsToPeer(b.ID())) == 0 || len(b.Network().ConnsToPeer(a.ID())) == 0 {
		t.Errorf("Fail to connect %v with %v", a.ID(), b.ID())
	}
	time.Sleep(time.Millisecond * 100)
}

func makeNodes(t *testing.T, ctx context.Context, number int) []*Node {
	nodes := make([]*Node, number)
	for i := 0; i < number; i++ {
		nodes = append(nodes, makeUnbootstrappedNode(t, ctx, i))
	}
	return nodes
}

func makePeerNodes(t *testing.T, ctx context.Context) (*Node, *Node) {
	node0 := makeUnbootstrappedNode(t, ctx, 0)
	node1 := makeUnbootstrappedNode(t, ctx, 1)
	// if node0.IsPeer(node1.ID()) || node1.IsPeer(node0.ID()) {
	// 	t.Error("Two initial nodes should not be connected without `AddPeer`")
	// }
	time.Sleep(time.Millisecond * 100)
	node0.AddPeer(ctx, node1.GetFullAddr())
	// wait until node0 receive the response from node1
	if !node0.IsPeer(node1.ID()) || !node1.IsPeer(node0.ID()) {
		t.Error("Failed to add peer")
	}
	connect(t, ctx, node0, node1)
	return node0, node1
}

func TestAddPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	makePeerNodes(t, ctx)
}
func TestBroadcastCollation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	node0, node1 := makePeerNodes(t, ctx)
	var testingShardID ShardIDType = 42
	node0.ListenShard(ctx, testingShardID)
	node0.PublishListeningShards(ctx)
	// TODO: fail: if the receiver didn't subscribe the shard, it should ignore the message

	node1.ListenShard(ctx, testingShardID)
	node1.PublishListeningShards(ctx)
	// TODO: fail: if the collation's shardID does not correspond to the protocol's shardID,
	//		 receiver should reject it

	// success
	err := node0.broadcastCollation(
		ctx,
		testingShardID,
		1,
		[]byte("123"),
	)
	if err != nil {
		t.Errorf("failed to send collation %v, %v, %v. reason=%v", testingShardID, 1, 123, err)
	}
}

func makePartiallyConnected3Nodes(t *testing.T, ctx context.Context) []*Node {
	node0, node1 := makePeerNodes(t, ctx)
	node2 := makeUnbootstrappedNode(t, ctx, 2)
	time.Sleep(time.Millisecond * 100)
	node2.AddPeer(ctx, node1.GetFullAddr())
	if !node1.IsPeer(node2.ID()) || !node2.IsPeer(node1.ID()) {
		t.Error()
	}
	connect(t, ctx, node1, node2)
	return [](*Node){node0, node1, node2}
}

func TestRequestCollation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	node0, node1 := makePeerNodes(t, ctx)
	shardID := ShardIDType(1)
	period := 42
	collation, err := node0.requestCollation(ctx, node1.ID(), shardID, period)
	if err != nil {
		t.Errorf("request collation failed: %v", err)
	}
	if collation.ShardID != shardID || int(collation.Period) != period {
		t.Errorf(
			"responded collation does not correspond to the request: collation.ShardID=%v, request.shardID=%v, collation.Period=%v, request.period=%v",
			collation.ShardID,
			shardID,
			collation.Period,
			period,
		)
	}
}

func TestRequestCollationNotFound(t *testing.T) {
	// TODO:
}

func TestRequestShardPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	node0, node1 := makePeerNodes(t, ctx)
	time.Sleep(time.Millisecond * 100)
	shardPeers, err := node0.requestShardPeer(ctx, node1.ID(), []ShardIDType{1})
	if err != nil {
		t.Errorf("Error occurred when requesting shard peer: %v", err)
	}
	peerIDs, prs := shardPeers[1]
	if !prs {
		t.Errorf("node1 should still return a empty peer list of the shard %v", 1)
	}
	if len(peerIDs) != 0 {
		t.Errorf("Wrong shard peer response %v, should be empty peerIDs", peerIDs)
	}
	node1.ListenShard(ctx, 1)
	node1.ListenShard(ctx, 2)
	// node1 listens to [1, 2], but we only request shard [1]
	shardPeers, err = node0.requestShardPeer(ctx, node1.ID(), []ShardIDType{1})
	if err != nil {
		t.Errorf("Error occurred when requesting shard peer: %v", err)
	}
	peerIDs1, prs := shardPeers[1]
	if len(peerIDs1) != 1 || peerIDs1[0] != node1.ID() {
		t.Errorf("Wrong shard peer response %v, should be node1.ID() only", peerIDs1)
	}
	// node1 only listens to [1, 2], but we request shard [1, 2, 3]
	shardPeers, err = node0.requestShardPeer(ctx, node1.ID(), []ShardIDType{1, 2, 3})
	if err != nil {
		t.Errorf("Error occurred when requesting shard peer: %v", err)
	}
	peerIDs1, prs = shardPeers[1]
	if len(peerIDs1) != 1 || peerIDs1[0] != node1.ID() {
		t.Errorf("Wrong shard peer response %v, should be node1.ID() only", peerIDs1)
	}
	peerIDs2, prs := shardPeers[2]
	if len(peerIDs2) != 1 || peerIDs2[0] != node1.ID() {
		t.Errorf("Wrong shard peer response %v, should be node1.ID() only", peerIDs2)
	}
	peerIDs3, prs := shardPeers[3]
	if !prs {
		t.Errorf("node1 should still return a empty peer list of the shard %v", 3)
	}
	if len(peerIDs3) != 0 {
		t.Errorf("Wrong shard peer response %v, should be empty peerIDs", peerIDs3)
	}
}

func TestRouting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// set the logger to DEBUG, to see the process of dht.FindPeer
	// we should be able to see something like
	// "dht: FindPeer <peer.ID d3wzD2> true routed.go:76", if successfully found the desire peer
	// golog.SetAllLoggers(gologging.DEBUG) // Change to DEBUG for extra info
	nodes := makePartiallyConnected3Nodes(t, ctx)
	time.Sleep(time.Millisecond * 100)
	if nodes[0].IsPeer(nodes[2].ID()) {
		t.Error("node0 should not be able to reach node2 before routing")
	}
	nodes[0].Connect(
		ctx,
		pstore.PeerInfo{
			ID: nodes[2].ID(),
		},
	)
	time.Sleep(time.Millisecond * 100)
	if !nodes[2].IsPeer(nodes[0].ID()) {
		t.Error("node0 should be a peer of node2 now")
	}
}

func TestPubSub(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makePartiallyConnected3Nodes(t, ctx)

	topic := "iamtopic"

	subch0, err := nodes[0].pubsubService.Subscribe(topic)
	if err != nil {
		t.Fatal(err)
	}

	// TODO: This sleep is necessary!!! Find out why
	time.Sleep(time.Millisecond * 100)

	publishMsg := "789"
	err = nodes[1].pubsubService.Publish(topic, []byte(publishMsg))
	if err != nil {
		t.Fatal(err)
	}
	msg0, err := subch0.Next(ctx)
	if string(msg0.Data) != publishMsg {
		t.Fatal(err)
	}
	if msg0.GetFrom() != nodes[1].ID() {
		t.Error("Wrong ID")
	}
}

func TestPubSubNotifyListeningShards(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makePartiallyConnected3Nodes(t, ctx)

	// wait for heartbeats to build mesh
	time.Sleep(time.Second * 2)

	// ensure notifyShards message is propagated through node1
	if len(nodes[1].shardPrefTable.GetPeerListeningShardSlice(nodes[0].ID())) != 0 {
		t.Error()
	}
	nodes[0].ListenShard(ctx, 42)
	nodes[0].PublishListeningShards(ctx)
	time.Sleep(time.Millisecond * 100)
	if len(nodes[1].shardPrefTable.GetPeerListeningShardSlice(nodes[0].ID())) != 1 {
		t.Error()
	}
	if len(nodes[2].shardPrefTable.GetPeerListeningShardSlice(nodes[0].ID())) != 1 {
		t.Error()
	}
	nodes[1].ListenShard(ctx, 42)
	nodes[1].PublishListeningShards(ctx)

	time.Sleep(time.Millisecond * 100)

	shardPeers42 := nodes[2].shardPrefTable.GetPeersInShard(42)
	if len(shardPeers42) != 2 {
		t.Errorf(
			"len(shardPeers42) should be %v, not %v. shardPeers42=%v",
			2,
			len(shardPeers42),
			shardPeers42,
		)
	}

	// test unsetShard with notifying
	nodes[0].UnlistenShard(ctx, 42)
	nodes[0].PublishListeningShards(ctx)
	time.Sleep(time.Millisecond * 100)
	if len(nodes[1].shardPrefTable.GetPeerListeningShardSlice(nodes[0].ID())) != 0 {
		t.Error()
	}
}

// test if nodes can find each other with ipfs nodes
func TestWithIPFSNodesRouting(t *testing.T) {
	// FIXME: skipped by default, since currently our boostrapping nodes are local ipfs nodes
	t.Skip()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// golog.SetAllLoggers(gologging.DEBUG) // Change to DEBUG for extra info
	ipfsPeers := convertPeers([]string{
		"/ip4/127.0.0.1/tcp/4001/ipfs/QmXa1ncfGc9RQotUAyN3Gb4ar7WXZ4DEb2wPZsSybkK1sf",
		"/ip4/104.131.131.82/tcp/4001/ipfs/QmaCpDMGvV2BGHeYERUEnRQAwe3N8SzbUtfsmvsqQLuvuJ",
		"/ip4/104.236.179.241/tcp/4001/ipfs/QmSoLPppuBtQSGwKDZT2M73ULpjvfd3aZ6ha4oFGL1KrGM",
		"/ip4/128.199.219.111/tcp/4001/ipfs/QmSoLSafTMBsPKadTEgaXctDQVcqN88CNLHXMkTNwMKPnu",
		"/ip4/178.62.158.247/tcp/4001/ipfs/QmSoLer265NRgSp2LA3dPaeykiS1J6DifTC88f5uVQKNAd",
	})
	ipfsPeer0 := ipfsPeers[0]
	ipfsPeer1 := ipfsPeers[1]

	node0 := makeUnbootstrappedNode(t, ctx, 0)
	node0.Peerstore().AddAddrs(ipfsPeer0.ID, ipfsPeer0.Addrs, pstore.PermanentAddrTTL)
	node0.Connect(context.Background(), ipfsPeer0)
	if len(node0.Network().ConnsToPeer(ipfsPeer0.ID)) == 0 {
		t.Error()
	}
	node1 := makeUnbootstrappedNode(t, ctx, 1)
	node1.Peerstore().AddAddrs(ipfsPeer1.ID, ipfsPeer1.Addrs, pstore.PermanentAddrTTL)
	node1.Connect(context.Background(), ipfsPeer1)
	if len(node1.Network().ConnsToPeer(ipfsPeer1.ID)) == 0 {
		t.Error()
	}
	node0PeerInfo := pstore.PeerInfo{
		ID:    node0.ID(),
		Addrs: []ma.Multiaddr{},
	}
	// ensure connection: node0 <-> ipfsPeer0 <-> ipfsPeer1 <-> node1
	if len(node0.Network().ConnsToPeer(ipfsPeer1.ID)) != 0 {
		t.Error()
	}
	if len(node1.Network().ConnsToPeer(ipfsPeer0.ID)) != 0 {
		t.Error()
	}
	if len(node1.Network().ConnsToPeer(node0.ID())) != 0 {
		t.Error()
	}

	node1.Connect(context.Background(), node0PeerInfo)

	if len(node1.Network().ConnsToPeer(node0.ID())) == 0 {
		t.Error()
	}
}

func TestListenShardConnectingPeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodes := makePartiallyConnected3Nodes(t, ctx)
	// wait for mesh built
	time.Sleep(time.Second * 2)
	// 0 <-> 1 <-> 2
	nodes[0].ListenShard(ctx, 0)
	nodes[0].PublishListeningShards(ctx)
	time.Sleep(time.Millisecond * 100)
	nodes[2].ListenShard(ctx, 42)
	nodes[2].PublishListeningShards(ctx)
	time.Sleep(time.Millisecond * 100)
	connWithNode2 := nodes[0].Network().ConnsToPeer(nodes[2].ID())
	if len(connWithNode2) != 0 {
		t.Error("Node 0 shouldn't have connection with node 2")
	}
	nodes[0].ListenShard(ctx, 42)
	nodes[0].PublishListeningShards(ctx)
	time.Sleep(time.Second * 1)
	connWithNode2 = nodes[0].Network().ConnsToPeer(nodes[2].ID())
	if len(connWithNode2) == 0 {
		t.Error("Node 0 should have connected to node 2 after listening to shard 42")
	}
}

func TestDHTBootstrapping(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	bootnode := makeUnbootstrappedNode(t, ctx, 0)
	piBootnode := pstore.PeerInfo{
		ID:    bootnode.ID(),
		Addrs: bootnode.Addrs(),
	}
	node1 := makeUnbootstrappedNode(t, ctx, 1)
	connect(t, ctx, bootnode, node1)
	node2 := makeTestingNode(t, ctx, 2, true, []pstore.PeerInfo{piBootnode})
	time.Sleep(time.Millisecond * 100)
	if len(node2.Peerstore().PeerInfo(node1.ID()).Addrs) == 0 {
		t.Error("node2 should have known node1 through the bootnode")
	}
}

type eventTestServer struct {
	pbevent.EventServer
	collations chan *pbmsg.Collation
}

func makeEventResponse(success bool) *pbevent.Response {
	var status pbevent.Response_Status
	if success {
		status = pbevent.Response_SUCCESS
	} else {
		status = pbevent.Response_FAILURE
	}
	return &pbevent.Response{Status: status}
}

func (s *eventTestServer) NotifyCollation(
	ctx context.Context,
	req *pbevent.NotifyCollationRequest) (*pbevent.NotifyCollationResponse, error) {
	go func() {
		s.collations <- req.Collation
	}()
	res := &pbevent.NotifyCollationResponse{
		Response: makeEventResponse(true),
		IsValid:  true,
	}
	return res, nil
}
func (s *eventTestServer) GetCollation(
	ctx context.Context,
	req *pbevent.GetCollationRequest) (*pbevent.GetCollationResponse, error) {
	collation := &pbmsg.Collation{
		ShardID: req.ShardID,
		Period:  req.Period,
		Blobs:   []byte(fmt.Sprintf("shardID=%v, period=%v", req.ShardID, req.Period)),
	}
	res := &pbevent.GetCollationResponse{
		Response:  makeEventResponse(true),
		Collation: collation,
		IsFound:   true,
	}
	return res, nil
}

func runEventServer(ctx context.Context, eventRPCAddr string) (*eventTestServer, error) {
	lis, err := net.Listen("tcp", eventRPCAddr)
	if err != nil {
		return nil, err
	}
	s := grpc.NewServer()
	ets := &eventTestServer{collations: make(chan *pbmsg.Collation)}
	pbevent.RegisterEventServer(
		s,
		ets,
	)
	go s.Serve(lis)
	go func() {
		select {
		case <-ctx.Done():
			s.Stop()
		}
	}()
	return ets, nil
}

func TestEventRPCNotifier(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	eventRPCPort := 55666
	notifierAddr := fmt.Sprintf("127.0.0.1:%v", eventRPCPort)
	_, err := runEventServer(ctx, notifierAddr)
	if err != nil {
		t.Error("failed to run event server")
	}
	time.Sleep(time.Millisecond * 100)
	eventNotifier, err := NewRpcEventNotifier(ctx, notifierAddr)
	if err != nil {
		t.Error(err)
	}

	shardID := ShardIDType(1)
	period := 2

	// NotifyCollation
	collation := &pbmsg.Collation{
		ShardID: shardID,
		Period:  PBInt(period),
		Blobs:   []byte("123"),
	}
	isValid, err := eventNotifier.NotifyCollation(collation)
	if err != nil {
		if err != nil {
			t.Errorf("something wrong when calling `eventNotifier.NotifyCollation`: %v", err)
		}
	}
	if !isValid {
		t.Errorf("wrong reponse from `eventNotifier.NotifyCollation`")
	}

	// GetCollation
	collationHash := "123"
	collationReceived, err := eventNotifier.GetCollation(shardID, period, collationHash)
	if err != nil {
		t.Errorf("something wrong when calling `eventNotifier.GetCollation`: %v", err)
	}
	if ShardIDType(collationReceived.ShardID) != shardID &&
		int(collationReceived.Period) != period {
		t.Errorf("wrong response from `eventNotifier.GetCollation`")
	}
}

func TestSubscribeCollationWithRPCNotifier(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodes := makePartiallyConnected3Nodes(t, ctx)
	shardID := ShardIDType(1)
	nodes[0].ListenShard(ctx, shardID)
	nodes[1].ListenShard(ctx, shardID)
	nodes[2].ListenShard(ctx, shardID)
	time.Sleep(time.Second * 2)

	// setup 2 event server and notifier(stub), for node1 and node2 respectively
	eventRPCPort := 55667
	notifierAddr := fmt.Sprintf("127.0.0.1:%v", eventRPCPort)
	s1, err := runEventServer(ctx, notifierAddr)
	if err != nil {
		t.Error("failed to run event server")
	}
	eventNotifier, err := NewRpcEventNotifier(ctx, notifierAddr)
	if err != nil {
		t.Error("failed to create event notifier")
	}
	// explicitly set the eventNotifier
	nodes[1].eventNotifier = eventNotifier

	eventRPCPort++
	notifierAddr = fmt.Sprintf("127.0.0.1:%v", eventRPCPort)
	s2, err := runEventServer(ctx, notifierAddr)
	if err != nil {
		t.Error("failed to run event server")
	}
	eventNotifier, err = NewRpcEventNotifier(ctx, notifierAddr)
	if err != nil {
		t.Error("failed to create event notifier")
	}
	// explicitly set the eventNotifier
	nodes[2].eventNotifier = eventNotifier

	// TODO: should be sure when is the validator executed, and how it will affect relaying
	nodes[0].broadcastCollation(ctx, shardID, 1, []byte{})
	select {
	case collation := <-s1.collations:
		if ShardIDType(collation.ShardID) != shardID || int(collation.Period) != 1 {
			t.Error("node1 received wrong collations")
		}
	case <-time.After(time.Second * 5):
		t.Error("timeout in node1")
	}
	select {
	case collation := <-s2.collations:
		if ShardIDType(collation.ShardID) != shardID || int(collation.Period) != 1 {
			t.Error("node2 received wrong collations")
		}
	case <-time.After(time.Second * 5):
		t.Error("timeout in node2")
	}
}
