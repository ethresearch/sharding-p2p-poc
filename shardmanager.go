package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	pubsub "github.com/libp2p/go-floodsub"
	host "github.com/libp2p/go-libp2p-host"
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
	spanctx := logger.Start(ctx, "ShardManager.connectShardNodes")
	defer logger.Finish(spanctx)
	logger.SetTag(spanctx, "shard", shardID)

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
	var wg sync.WaitGroup
	for _, p := range pinfos {
		wg.Add(1)
		go func(p pstore.PeerInfo) {
			// Add span for Connect of ShardManager.connectShardNodes
			childSpanctx := logger.Start(spanctx, "ShardManager.connectShardNodes.Connect")
			defer logger.Finish(childSpanctx)

			defer wg.Done()
			if err := n.host.Connect(spanctx, p); err != nil {
				logger.SetErr(childSpanctx, fmt.Errorf("Failed to connect peer %v in shard %v, err: %v", p.ID, shardID, err))
				log.Printf(
					"Failed to connect peer %v in shard %v: %s",
					p.ID,
					shardID,
					err,
				)
				return
			}
			log.Printf("Successfully connected peer %v in shard %v", p.ID, shardID)
			logger.SetTag(childSpanctx, "peer", p.ID)
		}(p)
	}
	wg.Wait()
	// FIXME: ignore the errors when connecting shard peers for now
	return nil
}

func (n *ShardManager) ListenShard(ctx context.Context, shardID ShardIDType) error {
	// Add span for ListenShard of ShardManager
	spanctx := logger.Start(ctx, "ShardManager.ListenShard")
	defer logger.Finish(spanctx)

	if n.IsShardListened(shardID) {
		return nil
	}
	n.shardPrefTable.AddPeerListeningShard(n.host.ID(), shardID)

	// TODO: should set a critiria: if we have enough peers in the shard, don't connect shard nodes
	if err := n.connectShardNodes(spanctx, shardID); err != nil {
		logger.FinishWithErr(spanctx, fmt.Errorf("Failed to connect to nodes in shard %v", shardID))
		log.Printf("Failed to connect to nodes in shard %v", shardID)
	}

	// shardCollations protocol
	if err := n.SubscribeCollation(spanctx, shardID); err != nil {
		logger.FinishWithErr(spanctx, fmt.Errorf("Failed to subscribe to collation in shard %v", shardID))
		log.Printf("Failed to subscribe to collation in shard %v", shardID)
		return err
	}

	// Tag shardID info if nothing goes wrong
	logger.SetTag(spanctx, "shardID", fmt.Sprintf("%v", shardID))
	return nil
}

func (n *ShardManager) UnlistenShard(ctx context.Context, shardID ShardIDType) error {
	// Add span for UnlistenShard of ShardManager
	spanctx := logger.Start(ctx, "ShardManager.UnlistenShard")
	defer logger.Finish(spanctx)

	if !n.IsShardListened(shardID) {
		return nil
	}
	n.shardPrefTable.RemovePeerListeningShard(n.host.ID(), shardID)

	// listeningShards protocol
	// TODO: should we remove some peers in this shard?

	// shardCollations protocol
	if err := n.UnsubscribeCollation(spanctx, shardID); err != nil {
		logger.FinishWithErr(spanctx, fmt.Errorf("Failed to unsubscribe to collation in shard %v", shardID))
		log.Printf("Failed to unsubscribe to collation in shard %v", shardID)
		return err
	}

	// Tag shardID info if nothing goes wrong
	logger.SetTag(spanctx, "shardID", fmt.Sprintf("%v", shardID))
	return nil
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
	ctx context.Context,
	topic string,
	handler TopicHandler,
	validator TopicValidator) error {
	// Add span for SubscribeTopic of ShardManager
	spanctx := logger.Start(ctx, "ShardManager.SubscribeTopic")
	defer logger.Finish(spanctx)

	sub, err := n.pubsubService.Subscribe(topic)
	if err != nil {
		log.Printf("Failed to subscribe to topic: %s", topic)
		logger.FinishWithErr(spanctx, fmt.Errorf("Failed to subscribe to topic: %s, err: %v", topic, err))
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
	// Tag topic to subscribe to if nothing goes wrong
	logger.SetTag(spanctx, "Topic", topic)
	return nil
}

func (n *ShardManager) UnsubscribeTopic(ctx context.Context, topic string) error {
	// Add span for UnsubscribeTopic of ShardManager
	spanctx := logger.Start(ctx, "ShardManager.UnsubscribeTopic")
	defer logger.Finish(spanctx)

	n.subsLock.RLock()
	sub, prs := n.subs[topic]
	n.subsLock.RUnlock()
	if !prs {
		return nil
	}
	// Tag topic to unsubscribe to
	logger.SetTag(spanctx, "Topic", topic)

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
		context.Background(),
		listeningShardTopic,
		n.makeShardPrefHandler(),
		n.makeShardPrefValidator(),
	)
}

func (n *ShardManager) UnsubscribeListeningShards() error {
	return n.UnsubscribeTopic(context.Background(), listeningShardTopic)
}

func (n *ShardManager) PublishListeningShards(ctx context.Context) {
	// Add span for PublishListeningShards of ShardManager
	spanctx := logger.Start(ctx, "ShardManager.PublishListeningShards")
	defer logger.Finish(spanctx)

	selfListeningShards := n.shardPrefTable.GetPeerListeningShard(n.host.ID())
	bytes := selfListeningShards.toBytes()
	if err := n.pubsubService.Publish(listeningShardTopic, bytes); err != nil {
		logger.SetErr(spanctx, fmt.Errorf("Failed to publish listening shards, err: %v", err))
		log.Println("Failed to publish listening shards")
	}
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
			validity, _ := n.eventNotifier.NotifyCollation(collation)
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
	// Add span for SubscribeCollation of ShardManager
	spanctx := logger.Start(ctx, "ShardManager.SubscribeCollation")
	defer logger.Finish(spanctx)

	topic := getCollationsTopic(shardID)
	handler := n.makeCollationHandler()
	validator := n.makeCollationValidator()
	return n.SubscribeTopic(spanctx, topic, handler, validator)
}

func (n *ShardManager) UnsubscribeCollation(ctx context.Context, shardID ShardIDType) error {
	// Add span for UnsubscribeCollation of ShardManager
	spanctx := logger.Start(ctx, "ShardManager.UnsubscribeCollation")
	defer logger.Finish(spanctx)

	topic := getCollationsTopic(shardID)
	return n.UnsubscribeTopic(spanctx, topic)
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
	spanctx := logger.Start(ctx, "ShardManager.broadcastCollation")
	defer logger.Finish(spanctx)

	// create message data
	data := &pbmsg.Collation{
		ShardID: shardID,
		Period:  PBInt(period),
		Blobs:   blobs,
	}
	err := n.broadcastCollationMessage(data)
	if err != nil {
		logger.SetErr(spanctx, fmt.Errorf("Failed to broadcast collation message, err: %v", err))
		log.Println("Failed to broadcast collation message")
		return err
	}

	// Tag collation period and blobs if nothing goes wrong
	logger.SetTag(spanctx, "Period", period)
	logger.SetTag(spanctx, "Blobs", blobs)
	return nil
}

func (n *ShardManager) broadcastCollationMessage(collation *pbmsg.Collation) error {
	if !n.IsCollationSubscribed(collation.GetShardID()) {
		return fmt.Errorf("broadcasting to a not subscribed shard")
	}
	collationsTopic := getCollationsTopic(collation.ShardID)
	bytes, err := proto.Marshal(collation)
	if err != nil {
		log.Println(err)
		return err
	}
	err = n.pubsubService.Publish(collationsTopic, bytes)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// TODO: beacon chain

// notifier related

func (n *ShardManager) getCollation(
	shardID ShardIDType,
	period int,
	collationHash string) (*pbmsg.Collation, error) {
	// get collations from remote clients only when `n.eventNotifier` is set
	if n.eventNotifier != nil {
		collation, err := n.eventNotifier.GetCollation(shardID, period, collationHash)
		if err != nil {
			return nil, err
		}
		return collation, nil
	}
	return nil, nil
}
