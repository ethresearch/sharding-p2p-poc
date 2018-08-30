package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	pubsub "github.com/libp2p/go-floodsub"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"

	pbmsg "github.com/ethresearch/sharding-p2p-poc/pb/message"
	"github.com/golang/protobuf/proto"
	b58 "github.com/mr-tron/base58/base58"
)

type ShardManager struct {
	ctx  context.Context
	host host.Host // local host

	pubsubService *pubsub.PubSub
	subs          map[string]*pubsub.Subscription
	subsLock      sync.RWMutex

	eventNotifier EventNotifier

	peerListeningShards map[peer.ID]*ListeningShards // TODO: handle the case when peer leave
}

const listeningShardTopic = "listeningShard"
const collationTopicFmt = "shardCollations_%d"

type TopicValidator = pubsub.Validator
type TopicHandler = func(ctx context.Context, msg *pubsub.Message)

func getCollationHash(msg *pbmsg.Collation) string {
	hashBytes := keccak(msg)
	// FIXME: for convenience
	return b58.Encode(hashBytes)
}

func getCollationsTopic(shardID ShardIDType) string {
	return fmt.Sprintf(collationTopicFmt, shardID)
}

func NewShardManager(ctx context.Context, h host.Host, eventNotifier EventNotifier) *ShardManager {
	service, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		log.Fatalln(err)
	}
	p := &ShardManager{
		ctx:                 ctx,
		host:                h,
		subs:                make(map[string]*pubsub.Subscription),
		pubsubService:       service,
		subsLock:            sync.RWMutex{},
		eventNotifier:       eventNotifier,
		peerListeningShards: make(map[peer.ID]*ListeningShards),
	}
	err = p.SubscribeListeningShards()
	if err != nil {
		log.Fatalln(err)
	}
	return p
}

// management for peers' listening shards

func (n *ShardManager) AddPeerListeningShard(peerID peer.ID, shardID ShardIDType) {
	if shardID >= numShards {
		return
	}
	if n.IsPeerListeningShard(peerID, shardID) {
		return
	}
	if _, prs := n.peerListeningShards[peerID]; !prs {
		n.peerListeningShards[peerID] = NewListeningShards()
	}
	n.peerListeningShards[peerID].setShard(shardID)
}

func (n *ShardManager) RemovePeerListeningShard(peerID peer.ID, shardID ShardIDType) {
	if !n.IsPeerListeningShard(peerID, shardID) {
		return
	}
	n.peerListeningShards[peerID].unsetShard(shardID)
}

func (n *ShardManager) GetPeerListeningShard(peerID peer.ID) []ShardIDType {
	if _, prs := n.peerListeningShards[peerID]; !prs {
		return []ShardIDType{}
	}
	return n.peerListeningShards[peerID].getShards()
}

func (n *ShardManager) SetPeerListeningShard(peerID peer.ID, shardIDs []ShardIDType) {
	listeningShards := n.GetPeerListeningShard(peerID)
	for _, shardID := range listeningShards {
		n.RemovePeerListeningShard(peerID, shardID)
	}
	for _, shardID := range shardIDs {
		n.AddPeerListeningShard(peerID, shardID)
	}
}

func (n *ShardManager) IsPeerListeningShard(peerID peer.ID, shardID ShardIDType) bool {
	if _, prs := n.peerListeningShards[peerID]; !prs {
		return false
	}
	shards := n.GetPeerListeningShard(peerID)
	return inShards(shardID, shards)
}

func (n *ShardManager) GetNodesInShard(shardID ShardIDType) []peer.ID {
	peers := []peer.ID{}
	for peerID, listeningShards := range n.peerListeningShards {
		if listeningShards.isShardSet(shardID) {
			peers = append(peers, peerID)
		}
	}
	return peers
}

// listen/unlisten shards

func (n *ShardManager) connectShardNodes(shardID ShardIDType) error {
	peerIDs := n.GetNodesInShard(shardID)
	pinfos := []pstore.PeerInfo{}
	for _, peerID := range peerIDs {
		// don't connect ourselves
		if peerID == n.host.ID() {
			continue
		}
		// if already have conns, no need to add them
		connsToPeer := n.host.Network().ConnsToPeer(peerID)
		if len(connsToPeer) != 0 {
			continue
		}
		pi := n.host.Peerstore().PeerInfo(peerID)
		pinfos = append(pinfos, pi)
	}

	// borrowed from `bootstrapConnect`, should be modified/refactored and tested
	errs := make(chan error, len(pinfos))
	var wg sync.WaitGroup
	ctx := context.Background()
	for _, p := range pinfos {
		wg.Add(1)
		go func(p pstore.PeerInfo) {
			defer wg.Done()
			if err := n.host.Connect(ctx, p); err != nil {
				log.Printf(
					"Failed to connect peer %v in shard %v: %s",
					p.ID,
					shardID,
					err,
				)
				errs <- err
				return
			}
			log.Printf("Successfully connected peer %v in shard %v", p.ID, shardID)
		}(p)
	}
	wg.Wait()
	// FIXME: ignore the errors when connecting shard peers for now
	return nil
}

// TODO: return error if it fails
func (n *ShardManager) ListenShard(shardID ShardIDType) {
	if n.IsShardListened(shardID) {
		return
	}
	n.AddPeerListeningShard(n.host.ID(), shardID)

	// TODO: should set a critiria: if we have enough peers in the shard, don't connect shard nodes
	n.connectShardNodes(shardID)

	// shardCollations protocol
	n.SubscribeCollation(shardID)
}

func (n *ShardManager) UnlistenShard(shardID ShardIDType) {
	if !n.IsShardListened(shardID) {
		return
	}
	n.RemovePeerListeningShard(n.host.ID(), shardID)

	// listeningShards protocol
	// TODO: should we remove some peers in this shard?

	// shardCollations protocol
	n.UnsubscribeCollation(shardID)
}

func (n *ShardManager) GetListeningShards() []ShardIDType {
	return n.GetPeerListeningShard(n.host.ID())
}

func (n *ShardManager) IsShardListened(shardID ShardIDType) bool {
	return inShards(shardID, n.GetListeningShards())
}

func inShards(shardID ShardIDType, shards []ShardIDType) bool {
	for _, value := range shards {
		if value == shardID {
			return true
		}
	}
	return false
}

//
// PubSub related
//

// listeningShards notification

func (n *ShardManager) makeShardPrefHandler() TopicHandler {
	return func(ctx context.Context, msg *pubsub.Message) {
		shardPref := NewListeningShards().fromBytes(msg.GetData())
		peerID := msg.GetFrom()
		// if peerID == n.host.ID() {
		// 	return
		// }
		n.SetPeerListeningShard(peerID, shardPref.getShards())
	}
}

func (n *ShardManager) makeShardPrefValidator() TopicValidator {
	return func(ctx context.Context, msg *pubsub.Message) bool {
		// do nothing now
		return true
	}
}

func (n *ShardManager) SubscribeListeningShards() error {
	return n.SubscribeTopic(
		listeningShardTopic,
		n.makeShardPrefHandler(),
		n.makeShardPrefValidator(),
	)
}

func (n *ShardManager) UnsubscribeListeningShards() error {
	return n.UnsubscribeTopic(listeningShardTopic)
}

func (n *ShardManager) PublishListeningShards() {
	selfListeningShards, prs := n.peerListeningShards[n.host.ID()]
	if !prs {
		selfListeningShards = NewListeningShards()
	}
	bytes := selfListeningShards.toBytes()
	n.pubsubService.Publish(listeningShardTopic, bytes)
}

// shard collations

func (n *ShardManager) broadcastCollation(shardID ShardIDType, period int, blobs []byte) error {
	// create message data
	data := &pbmsg.Collation{
		ShardID: shardID,
		Period:  PBInt(period),
		Blobs:   blobs,
	}
	return n.broadcastCollationMessage(data)
}

func (n *ShardManager) broadcastCollationMessage(collation *pbmsg.Collation) error {
	if !n.IsCollationSubscribed(collation.GetShardID()) {
		return fmt.Errorf("broadcasting to a not subscribed shard")
	}
	collationsTopic := getCollationsTopic(collation.ShardID)
	bytes, err := proto.Marshal(collation)
	if err != nil {
		log.Fatal(err)
		return err
	}
	err = n.pubsubService.Publish(collationsTopic, bytes)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func extractProtoMsg(pubsubMsg *pubsub.Message, msg proto.Message) error {
	bytes := pubsubMsg.GetData()
	err := proto.Unmarshal(bytes, msg)
	if err != nil {
		return err
	}
	return nil
}

func (n *ShardManager) makeCollationValidator() TopicValidator {
	return func(ctx context.Context, msg *pubsub.Message) bool {
		collation := &pbmsg.Collation{}
		err := extractProtoMsg(msg, collation)
		if err != nil {
			return false
		}
		validity, err := n.eventNotifier.NotifyNewCollation(collation)
		return validity
	}
}

func (n *ShardManager) makeCollationHandler() TopicHandler {
	return func(ctx context.Context, msg *pubsub.Message) {
		// do nothing now
	}
}

func (n *ShardManager) SubscribeTopic(
	topic string,
	handler TopicHandler,
	validator TopicValidator) error {
	sub, err := n.pubsubService.Subscribe(topic)
	if err != nil {
		return err
	}
	n.subsLock.Lock()
	n.subs[topic] = sub
	n.subsLock.Unlock()
	if validator != nil {
		n.pubsubService.RegisterTopicValidator(topic, validator)
	}
	go func() {
		for {
			msg, err := sub.Next(n.ctx)
			if n.ctx.Err() != nil {
				log.Printf("%v: SubscribeTopic: n.ctx.Err()=%v", n.host.ID(), n.ctx.Err())
				return
			}
			if err != nil {
				log.Printf("%v: SubscribeTopic: err=%v", n.host.ID(), err)
				return
			}
			handler(n.ctx, msg)
		}
	}()
	return nil
}

func (n *ShardManager) UnsubscribeTopic(topic string) error {
	n.subsLock.RLock()
	sub, prs := n.subs[topic]
	n.subsLock.RUnlock()
	if !prs {
		return nil
	}
	sub.Cancel()
	n.subsLock.Lock()
	delete(n.subs, topic)
	n.subsLock.Unlock()
	return nil
}

func (n *ShardManager) SubscribeCollation(shardID ShardIDType) error {
	topic := getCollationsTopic(shardID)
	handler := n.makeCollationHandler()
	validator := n.makeCollationValidator()
	return n.SubscribeTopic(topic, handler, validator)
}

func (n *ShardManager) UnsubscribeCollation(shardID ShardIDType) error {
	topic := getCollationsTopic(shardID)
	return n.UnsubscribeTopic(topic)
}

func (n *ShardManager) IsCollationSubscribed(shardID ShardIDType) bool {
	n.subsLock.RLock()
	topic := getCollationsTopic(shardID)
	_, prs := n.subs[topic]
	n.subsLock.RUnlock()
	return prs
}
