package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	pbevent "github.com/ethresearch/sharding-p2p-poc/pb/event"
	pbmsg "github.com/ethresearch/sharding-p2p-poc/pb/message"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	"google.golang.org/grpc"
	// gologging "github.com/whyrusleeping/go-logging"
)

var nodeCount int

func makeNodes(t *testing.T, ctx context.Context, number int) []*Node {
	nodes := []*Node{}
	for i := 0; i < number; i++ {
		nodes = append(nodes, makeUnbootstrappedNode(t, ctx, i))
	}
	time.Sleep(time.Millisecond * 100)
	return nodes
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
	node, err := makeNode(ctx, defaultIP, listeningPort, number, nil, doBootstrapping, bootstrapPeers)
	if err != nil {
		t.Error("Failed to create node")
	}
	return node
}

func makeUnbootstrappedNode(t *testing.T, ctx context.Context, number int) *Node {
	return makeTestingNode(t, ctx, number, false, nil)
}

func connect(t *testing.T, ctx context.Context, a, b *Node) {
	err := a.AddPeer(ctx, b.GetFullAddr())
	if err != nil {
		t.Errorf("%v failed to `AddPeer` %v : %v", a.ID(), b.ID(), err)
	}
	if !a.IsPeer(b.ID()) || !b.IsPeer(a.ID()) {
		t.Error("AddPeer failed")
	}
	if len(a.Network().ConnsToPeer(b.ID())) == 0 || len(b.Network().ConnsToPeer(a.ID())) == 0 {
		t.Errorf("failed to connect %v with %v", a.ID(), b.ID())
	}
	time.Sleep(time.Millisecond * 100)
}

func connectBarbell(t *testing.T, ctx context.Context, nodes []*Node) {
	for i := 0; i < len(nodes)-1; i++ {
		connect(t, ctx, nodes[i], nodes[i+1])
	}
}

func connectAll(t *testing.T, ctx context.Context, nodes []*Node) {
	for i, nodei := range nodes {
		for j, nodej := range nodes {
			if i < j {
				connect(t, ctx, nodei, nodej)
			}
		}
	}
}

func waitForPubSubMeshBuilt() {
	time.Sleep(time.Second * 2)
}

func TestAddPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makeNodes(t, ctx, 2)
	connect(t, ctx, nodes[0], nodes[1])
}

func TestRouting(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makeNodes(t, ctx, 3)
	connectBarbell(t, ctx, nodes)
	waitForPubSubMeshBuilt()

	// set the logger to DEBUG, to see the process of dht.FindPeer
	// we should be able to see something like
	// "dht: FindPeer <peer.ID d3wzD2> true routed.go:76", if successfully found the desire peer
	// golog.SetAllLoggers(gologging.DEBUG) // Change to DEBUG for extra info
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

	nodes := makeNodes(t, ctx, 2)

	nodes[0].Peerstore().AddAddrs(ipfsPeer0.ID, ipfsPeer0.Addrs, pstore.PermanentAddrTTL)
	nodes[0].Connect(context.Background(), ipfsPeer0)
	if len(nodes[0].Network().ConnsToPeer(ipfsPeer0.ID)) == 0 {
		t.Error()
	}

	nodes[1].Peerstore().AddAddrs(ipfsPeer1.ID, ipfsPeer1.Addrs, pstore.PermanentAddrTTL)
	nodes[1].Connect(context.Background(), ipfsPeer1)
	if len(nodes[1].Network().ConnsToPeer(ipfsPeer1.ID)) == 0 {
		t.Error()
	}
	// ensure connection: node0 <-> ipfsPeer0 <-> ipfsPeer1 <-> node1
	if len(nodes[0].Network().ConnsToPeer(ipfsPeer1.ID)) != 0 {
		t.Error()
	}
	if len(nodes[1].Network().ConnsToPeer(ipfsPeer0.ID)) != 0 {
		t.Error()
	}
	if len(nodes[1].Network().ConnsToPeer(nodes[0].ID())) != 0 {
		t.Error()
	}

	node0PeerInfo := pstore.PeerInfo{
		ID:    nodes[0].ID(),
		Addrs: []ma.Multiaddr{},
	}
	nodes[1].Connect(context.Background(), node0PeerInfo)

	if len(nodes[1].Network().ConnsToPeer(nodes[0].ID())) == 0 {
		t.Error()
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

func TestRequestCollation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makeNodes(t, ctx, 2)
	connectBarbell(t, ctx, nodes)

	shardID := ShardIDType(1)
	period := 42
	collation, err := nodes[0].requestCollation(ctx, nodes[1].ID(), shardID, period)
	if err != nil {
		t.Errorf("request collation failed: %v", err)
	}
	if collation != nil {
		t.Errorf(
			"collation should not be returned since we didn't set an eventNotifier in nodes[1]",
		)
	}
}

func TestRequestShardPeer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makeNodes(t, ctx, 2)
	connectBarbell(t, ctx, nodes)

	shardPeers, err := nodes[0].requestShardPeer(ctx, nodes[1].ID(), []ShardIDType{1})
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

	if err := nodes[1].ListenShard(ctx, 1); err != nil {
		t.Errorf("Failed to listen to shard 1")
	}
	if err := nodes[1].ListenShard(ctx, 2); err != nil {
		t.Errorf("Failed to listen to shard 2")
	}
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)
	// node1 listens to [1, 2], but we only request shard [1]
	shardPeers, err = nodes[0].requestShardPeer(ctx, nodes[1].ID(), []ShardIDType{1})
	if err != nil {
		t.Errorf("Error occurred when requesting shard peer: %v", err)
	}
	peerIDs1, prs := shardPeers[1]
	if len(peerIDs1) != 1 || peerIDs1[0] != nodes[1].ID() {
		t.Errorf("Wrong shard peer response %v, should be nodes[1].ID() only", peerIDs1)
	}

	// node1 only listens to [1, 2], but we request shard [1, 2, 3]
	shardPeers, err = nodes[0].requestShardPeer(ctx, nodes[1].ID(), []ShardIDType{1, 2, 3})
	if err != nil {
		t.Errorf("Error occurred when requesting shard peer: %v", err)
	}
	peerIDs1, prs = shardPeers[1]
	if len(peerIDs1) != 1 || peerIDs1[0] != nodes[1].ID() {
		t.Errorf("Wrong shard peer response %v, should be nodes[1].ID() only", peerIDs1)
	}
	peerIDs2, prs := shardPeers[2]
	if len(peerIDs2) != 1 || peerIDs2[0] != nodes[1].ID() {
		t.Errorf("Wrong shard peer response %v, should be nodes[1].ID() only", peerIDs2)
	}
	peerIDs3, prs := shardPeers[3]
	if !prs {
		t.Errorf("node1 should still return a empty peer list of the shard %v", 3)
	}
	if len(peerIDs3) != 0 {
		t.Errorf("Wrong shard peer response %v, should be empty peerIDs", peerIDs3)
	}
}

func TestPubSub(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makeNodes(t, ctx, 3)
	connectBarbell(t, ctx, nodes)
	waitForPubSubMeshBuilt()

	topic := "iamtopic"

	subch0, err := nodes[0].pubsubService.Subscribe(topic)
	if err != nil {
		t.Error(err)
	}

	// TODO: This sleep is necessary!!! Find out why
	time.Sleep(time.Millisecond * 100)

	publishMsg := "789"
	err = nodes[1].pubsubService.Publish(topic, []byte(publishMsg))
	if err != nil {
		t.Error(err)
	}

	msg0, err := subch0.Next(ctx)
	if string(msg0.Data) != publishMsg {
		t.Error(err)
	}
	if msg0.GetFrom() != nodes[1].ID() {
		t.Error("Wrong ID")
	}
}

func TestListenShardAndUnlistenShard(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makeNodes(t, ctx, 2)
	connectBarbell(t, ctx, nodes)

	var testingShardID ShardIDType = 42
	// test `IsShardListened`
	if nodes[0].IsShardListened(testingShardID) {
		t.Errorf("Shard %v shoudn't have been listened", testingShardID)
	}
	// test `ListenShard`
	if err := nodes[0].ListenShard(ctx, testingShardID); err != nil {
		t.Errorf("Failed to listen to shard %v", testingShardID)
	}
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)
	if !nodes[0].IsShardListened(testingShardID) {
		t.Errorf("Shard %v should have been listened", testingShardID)
	}
	if !nodes[0].IsCollationSubscribed(testingShardID) {
		t.Errorf(
			"shardCollations in shard [%v] should be subscribed",
			testingShardID,
		)
	}
	if len(nodes[1].pubsubService.ListPeers(fmt.Sprintf("shardCollations_%v", testingShardID))) != 1 {
		t.Errorf("nodes[1] did not receive the subscription update")
	}

	// test `ListenShard` to same shard twice
	if err := nodes[0].ListenShard(ctx, testingShardID); err != nil {
		t.Errorf("Failed to listen to shard %v", testingShardID)
	}
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)
	if !nodes[0].IsShardListened(testingShardID) {
		t.Errorf("Shard %v should have been listened", testingShardID)
	}
	if !nodes[0].IsCollationSubscribed(testingShardID) {
		t.Errorf(
			"shardCollations in shard [%v] should be subscribed",
			testingShardID,
		)
	}

	// test `ListenShard` to another shard
	anotherShardID := testingShardID + 1
	if err := nodes[0].ListenShard(ctx, anotherShardID); err != nil {
		t.Errorf("Failed to listen to shard %v", anotherShardID)
	}
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)
	if !nodes[0].IsShardListened(anotherShardID) {
		t.Errorf("Shard %v should have been listened", anotherShardID)
	}
	shardIDs := nodes[0].GetListeningShards()
	if len(shardIDs) != 2 {
		t.Errorf("We should have 2 shards being listened, instead of %v", len(shardIDs))
	}
	if len(nodes[1].pubsubService.ListPeers(fmt.Sprintf("shardCollations_%v", anotherShardID))) != 1 {
		t.Errorf("nodes[1] did not receive the subscription update")
	}

	// test `UnlistenShard`
	if err := nodes[0].UnlistenShard(ctx, testingShardID); err != nil {
		t.Errorf("Failed to unlisten to shard %v", testingShardID)
	}
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)
	if nodes[0].IsShardListened(testingShardID) {
		t.Errorf("Shard %v should have been unlistened", testingShardID)
	}
	if nodes[0].IsCollationSubscribed(testingShardID) {
		t.Errorf(
			"shardCollations in shard [%v] should be already unsubscribed",
			testingShardID,
		)
	}
	if len(nodes[1].pubsubService.ListPeers(fmt.Sprintf("shardCollations_%v", testingShardID))) != 0 {
		t.Errorf("nodes[1] did not receive the subscription update")
	}

	// test `UnlistenShard` same shard twice
	if err := nodes[0].UnlistenShard(ctx, testingShardID); err != nil {
		t.Errorf("Failed to unlisten to shard %v", testingShardID)
	}
	if nodes[0].IsShardListened(testingShardID) {
		t.Errorf("Shard %v should have been unlistened", testingShardID)
	}
	if nodes[0].IsCollationSubscribed(testingShardID) {
		t.Errorf(
			"shardCollations in shard [%v] should be already unsubscribed",
			testingShardID,
		)
	}

	// test `UnlistenShard` another shard
	if err := nodes[0].UnlistenShard(ctx, anotherShardID); err != nil {
		t.Errorf("Failed to unlisten to shard %v", anotherShardID)
	}
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)
	if nodes[0].IsShardListened(anotherShardID) {
		t.Errorf("Shard %v should have been unlistened", anotherShardID)
	}
	if nodes[0].IsCollationSubscribed(anotherShardID) {
		t.Errorf(
			"shardCollations in shard [%v] should be already unsubscribed",
			anotherShardID,
		)
	}
	if len(nodes[1].pubsubService.ListPeers(fmt.Sprintf("shardCollations_%v", anotherShardID))) != 0 {
		t.Errorf("nodes[1] did not receive the subscription update")
	}
}

func TestListenShardConnectingPeers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makeNodes(t, ctx, 3)
	connectBarbell(t, ctx, nodes)
	waitForPubSubMeshBuilt()

	// 0 <-> 1 <-> 2
	if err := nodes[0].ListenShard(ctx, 0); err != nil {
		t.Errorf("Failed to listen to shard 0")
	}
	nodes[0].PublishListeningShards(ctx)

	if err := nodes[2].ListenShard(ctx, 42); err != nil {
		t.Errorf("Failed to listen to shard 42")
	}
	nodes[2].PublishListeningShards(ctx)
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)

	connWithNode2 := nodes[0].Network().ConnsToPeer(nodes[2].ID())
	if len(connWithNode2) != 0 {
		t.Error("Node 0 shouldn't have connection with node 2")
	}

	if err := nodes[0].ListenShard(ctx, 42); err != nil {
		t.Errorf("Failed to listen to shard 42")
	}
	nodes[0].PublishListeningShards(ctx)
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)

	connWithNode2 = nodes[0].Network().ConnsToPeer(nodes[2].ID())
	if len(connWithNode2) == 0 {
		t.Error("Node 0 should have connected to node 2 after listening to shard 42")
	}
}

func TestPubSubNotifyListeningShards(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makeNodes(t, ctx, 3)
	connectBarbell(t, ctx, nodes)
	waitForPubSubMeshBuilt()

	// ensure notifyShards message is propagated through node1
	if len(nodes[1].shardPrefTable.GetPeerListeningShardSlice(nodes[0].ID())) != 0 {
		t.Error()
	}

	if err := nodes[0].ListenShard(ctx, 42); err != nil {
		t.Errorf("Failed to listen to shard 42")
	}
	nodes[0].PublishListeningShards(ctx)
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)

	if len(nodes[1].shardPrefTable.GetPeerListeningShardSlice(nodes[0].ID())) != 1 {
		t.Error()
	}
	if len(nodes[2].shardPrefTable.GetPeerListeningShardSlice(nodes[0].ID())) != 1 {
		t.Error()
	}

	if err := nodes[1].ListenShard(ctx, 42); err != nil {
		t.Errorf("Failed to listen to shard 42")
	}
	nodes[1].PublishListeningShards(ctx)
	// wait for peer to receive shard subscription update
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
	if err := nodes[0].UnlistenShard(ctx, 42); err != nil {
		t.Errorf("Failed to listen to shard 42")
	}
	nodes[0].PublishListeningShards(ctx)
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)

	if len(nodes[1].shardPrefTable.GetPeerListeningShardSlice(nodes[0].ID())) != 0 {
		t.Error()
	}
}

func TestBroadcastCollation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makeNodes(t, ctx, 2)
	connectBarbell(t, ctx, nodes)

	var testingShardID ShardIDType = 42
	if err := nodes[0].ListenShard(ctx, testingShardID); err != nil {
		t.Errorf("Failed to listen to shard %v", testingShardID)
	}
	nodes[0].PublishListeningShards(ctx)
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)
	// TODO: fail: if the receiver didn't subscribe the shard, it should ignore the message

	if err := nodes[1].ListenShard(ctx, testingShardID); err != nil {
		t.Errorf("Failed to listen to shard %v", testingShardID)
	}
	nodes[1].PublishListeningShards(ctx)
	// wait for peer to receive shard subscription update
	time.Sleep(time.Millisecond * 100)
	// TODO: fail: if the collation's shardID does not correspond to the protocol's shardID,
	//		 receiver should reject it

	// success
	err := nodes[0].broadcastCollation(
		ctx,
		testingShardID,
		1,
		[]byte("123"),
	)
	if err != nil {
		t.Errorf("failed to send collation %v, %v, %v. reason=%v", testingShardID, 1, 123, err)
	}
}

type eventTestServer struct {
	pbevent.EventServer
	collations chan *pbmsg.Collation
}

func makeEventResponse(success bool) *pbevent.Response {
	var status pbevent.Response_Status
	var msg string
	if success {
		status = pbevent.Response_SUCCESS
		msg = ""
	} else {
		status = pbevent.Response_FAILURE
		msg = "failed"
	}
	return &pbevent.Response{Status: status, Message: msg}
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
	var collation *pbmsg.Collation
	// simulate the case when collation is not found
	if int(req.ShardID) == 1 && int(req.Period) == 42 {
		collation = nil
	} else {
		collation = &pbmsg.Collation{
			ShardID: req.ShardID,
			Period:  req.Period,
			Blobs:   []byte(fmt.Sprintf("shardID=%v, period=%v", req.ShardID, req.Period)),
		}
	}
	res := &pbevent.GetCollationResponse{
		Response:  makeEventResponse(true),
		Collation: collation,
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
		t.Errorf("something wrong when calling `eventNotifier.NotifyCollation`: %v", err)
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

func TestSubscribeCollationWithRPCEventNotifier(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makeNodes(t, ctx, 3)
	connectBarbell(t, ctx, nodes)

	shardID := ShardIDType(1)
	if err := nodes[0].ListenShard(ctx, shardID); err != nil {
		t.Errorf("Failed to listen to shard %v", shardID)
	}
	if err := nodes[1].ListenShard(ctx, shardID); err != nil {
		t.Errorf("Failed to listen to shard %v", shardID)
	}
	if err := nodes[2].ListenShard(ctx, shardID); err != nil {
		t.Errorf("Failed to listen to shard %v", shardID)
	}
	waitForPubSubMeshBuilt()

	// setup 2 event server and notifier(stub), for node1 and node2 respectively
	// event server and notifier for node1
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
	// event server and notifier for node2
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
	if err := nodes[0].broadcastCollation(ctx, shardID, 1, []byte{}); err != nil {
		t.Error("Failed to broadcast collation")
	}

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

func TestRequestCollationWithRPCEventNotifier(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodes := makeNodes(t, ctx, 2)
	connectBarbell(t, ctx, nodes)

	// case: without RPCEventNotifier set
	shardID := ShardIDType(1)
	period := 2
	collation, err := nodes[0].requestCollation(ctx, nodes[1].ID(), shardID, period)
	if err != nil {
		t.Errorf("request collation failed: %v", err)
	}
	if collation != nil {
		t.Errorf("collation should be nil because nodes[1] haven't setup a eventNotifier")
	}

	// case: with RPCEventNotifier set, db in the event server has the collation
	notifierAddr := fmt.Sprintf("127.0.0.1:%v", 55669)
	_, err = runEventServer(ctx, notifierAddr)
	if err != nil {
		t.Error("failed to run event server")
	}
	eventNotifier, err := NewRpcEventNotifier(ctx, notifierAddr)
	if err != nil {
		t.Error("failed to create event notifier")
	}
	// explicitly set the eventNotifier
	nodes[1].eventNotifier = eventNotifier

	collation, err = nodes[0].requestCollation(ctx, nodes[1].ID(), shardID, period)
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

	// case: with RPCEventNotifier set, db in the event server doesn't have the collation
	periodNotFound := 42
	collation, err = nodes[0].requestCollation(ctx, nodes[1].ID(), shardID, periodNotFound)
	if err != nil {
		t.Errorf("request collation failed: %v", err)
	}
	if collation != nil {
		t.Errorf("collation should be nil because it it not found in the mock event server")
	}
}
