package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	pubsub "github.com/libp2p/go-floodsub"
	host "github.com/libp2p/go-libp2p-host"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	opentracing "github.com/opentracing/opentracing-go"

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

	shardPrefTable *ShardPrefTable
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
		ctx:            ctx,
		host:           h,
		subs:           make(map[string]*pubsub.Subscription),
		pubsubService:  service,
		subsLock:       sync.RWMutex{},
		eventNotifier:  eventNotifier,
		shardPrefTable: NewShardPrefTable(),
	}
	err = p.SubscribeListeningShards()
	if err != nil {
		log.Fatalln(err)
	}
	return p
}

// General

func (n *ShardManager) connectShardNodes(ctx context.Context, shardID ShardIDType) error {
	// Add span for connectShardNodes of ShardManager
	span, _ := opentracing.StartSpanFromContext(ctx, "ShardManager.connectShardNodes")
	defer span.Finish()

	peerIDs := n.shardPrefTable.GetPeersInShard(shardID)
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
	ctx = context.Background()
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
func (n *ShardManager) ListenShard(ctx context.Context, shardID ShardIDType) {
	// Add span for AddPeer of ShardManager
	span, newctx := opentracing.StartSpanFromContext(ctx, "ShardManager.ListenShard")
	defer span.Finish()
	// Set shardID info in Baggage
	span.SetBaggageItem("shardID", fmt.Sprintf("%v", shardID))

	if n.IsShardListened(shardID) {
		return
	}
	n.shardPrefTable.AddPeerListeningShard(n.host.ID(), shardID)

	// TODO: should set a critiria: if we have enough peers in the shard, don't connect shard nodes
	n.connectShardNodes(newctx, shardID)

	// shardCollations protocol
	n.SubscribeCollation(newctx, shardID)
}

func (n *ShardManager) UnlistenShard(ctx context.Context, shardID ShardIDType) {
	// Add span for UnlistenShard of ShardManager
	span, newctx := opentracing.StartSpanFromContext(ctx, "ShardManager.UnlistenShard")
	defer span.Finish()
	// Set shardID info in Baggage
	span.SetBaggageItem("shardID", fmt.Sprintf("%v", shardID))

	if !n.IsShardListened(shardID) {
		return
	}
	n.shardPrefTable.RemovePeerListeningShard(n.host.ID(), shardID)

	// listeningShards protocol
	// TODO: should we remove some peers in this shard?

	// shardCollations protocol
	n.UnsubscribeCollation(newctx, shardID)
}

func (n *ShardManager) GetListeningShards() []ShardIDType {
	return n.shardPrefTable.GetPeerListeningShardSlice(n.host.ID())
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

// General

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
				return
			}
			if err != nil {
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

// listeningShards notification

func (n *ShardManager) makeShardPrefHandler() TopicHandler {
	return func(ctx context.Context, msg *pubsub.Message) {
		shardPref := NewListeningShards().fromBytes(msg.GetData())
		peerID := msg.GetFrom()
		n.shardPrefTable.SetPeerListeningShard(peerID, shardPref)
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

func (n *ShardManager) PublishListeningShards(ctx context.Context) {
	// Add span for PublishListeningShards of AddPeerProtocol
	span, _ := opentracing.StartSpanFromContext(ctx, "ShardManager.PublishListeningShards")
	defer span.Finish()

	selfListeningShards := n.shardPrefTable.GetPeerListeningShard(n.host.ID())
	bytes := selfListeningShards.toBytes()
	n.pubsubService.Publish(listeningShardTopic, bytes)
}

// Collations

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
		// FIXME: if no eventNotifier, just skip the verification
		if n.eventNotifier != nil {
			validity, _ := n.eventNotifier.NotifyNewCollation(collation)
			return validity
		}
		return true
	}
}

func (n *ShardManager) makeCollationHandler() TopicHandler {
	return func(ctx context.Context, msg *pubsub.Message) {
		// do nothing now
	}
}

func (n *ShardManager) SubscribeCollation(ctx context.Context, shardID ShardIDType) error {
	// Add span for SubscribeShardCollations of ShardManager
	span, _ := opentracing.StartSpanFromContext(ctx, "ShardManager.SubscribeCollation")
	defer span.Finish()

	topic := getCollationsTopic(shardID)
	handler := n.makeCollationHandler()
	validator := n.makeCollationValidator()
	return n.SubscribeTopic(topic, handler, validator)
}

func (n *ShardManager) UnsubscribeCollation(ctx context.Context, shardID ShardIDType) error {
	// Add span for UnsubscribeShardCollations of ShardManager
	span, _ := opentracing.StartSpanFromContext(ctx, "ShardManager.UnsubscribeCollation")
	defer span.Finish()

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

func (n *ShardManager) broadcastCollation(
	ctx context.Context,
	shardID ShardIDType,
	period int,
	blobs []byte) error {
	// Add span for broadcastCollation of ShardManager
	span, _ := opentracing.StartSpanFromContext(ctx, "ShardManager.broadcastCollation")
	defer span.Finish()
	span.SetTag("Period", period)
	span.SetTag("Blobs", blobs)

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

// TODO: beacon chain
