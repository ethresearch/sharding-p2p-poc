package main

import (
	"context"
	"fmt"
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

	discovery Discovery

	shardPrefTable *ShardPrefTable
}

const listeningShardTopic = "listeningShard"
const collationTopicFmt = "shardCollations_%d"

// FIXME: this should only be temporary, since this layer should not know the type of the data
//		  Currently use it to support the legacy function `broadcastCollationMessage`
const (
	typeUnknown int = iota
	typeCollation
	typeCollationRequest
)

type TopicValidator = pubsub.Validator
type TopicHandler = func(ctx context.Context, msg *pubsub.Message)

func getCollationHash(msg *pbmsg.Collation) (string, error) {
	hashBytes, err := keccak(msg)
	if err != nil {
		logger.Errorf("Failed to hash the given message: %v, err: %v", msg, err)
		return "", err
	}
	// FIXME: for convenience
	return b58.Encode(hashBytes), nil
}

func getCollationsTopic(shardID ShardIDType) string {
	return fmt.Sprintf(collationTopicFmt, shardID)
}

func NewShardManager(
	ctx context.Context,
	h host.Host,
	pubsubService *pubsub.PubSub,
	eventNotifier EventNotifier,
	discovery Discovery,
	shardPrefTable *ShardPrefTable) *ShardManager {
	p := &ShardManager{
		ctx:            ctx,
		host:           h,
		subs:           make(map[string]*pubsub.Subscription),
		pubsubService:  pubsubService,
		subsLock:       sync.RWMutex{},
		eventNotifier:  eventNotifier,
		discovery:      discovery,
		shardPrefTable: shardPrefTable,
	}
	err := p.SubscribeListeningShards()
	if err != nil {
		logger.Fatalf("Failed to subscribe to global topic 'listeningShard', err: %v", err)
	}
	return p
}

// General

func (n *ShardManager) connectShardNodes(ctx context.Context, shardID ShardIDType) error {
	// Add span for connectShardNodes of ShardManager
	spanctx := logger.Start(ctx, "ShardManager.connectShardNodes")
	defer logger.Finish(spanctx)
	logger.SetTag(spanctx, "shard", shardID)

	pinfos, err := n.discovery.FindPeers(spanctx, shardID)
	if err != nil {
		logger.SetErr(spanctx, fmt.Errorf("Failed to find peers in shard %v", shardID))
		logger.Errorf("Failed to find peers in shard %v", shardID)
		return err
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

			// Skip if we already have connection with the peer
			connsToPeer := n.host.Network().ConnsToPeer(p.ID)
			if len(connsToPeer) != 0 {
				return
			}

			if err := n.host.Connect(spanctx, p); err != nil {
				logger.SetErr(childSpanctx, fmt.Errorf("Failed to connect peer %v in shard %v, err: %v", p.ID, shardID, err))
				logger.Errorf(
					"Failed to connect peer %v in shard %v: %s",
					p.ID,
					shardID,
					err,
				)
				return
			}
			logger.Debugf("Successfully connected peer %v in shard %v", p.ID, shardID)
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

	if err := n.discovery.Advertise(spanctx, shardID); err != nil {
		logger.SetErr(spanctx, fmt.Errorf("Failed to advertise subscription to shard %v", shardID))
		logger.Errorf("Failed to advertise subscription to shard %v", shardID)
		return err
	}

	// TODO: should set a critiria: if we have enough peers in the shard, don't connect shard nodes
	if err := n.connectShardNodes(spanctx, shardID); err != nil {
		logger.SetErr(spanctx, fmt.Errorf("Failed to connect to nodes in shard %v", shardID))
		logger.Errorf("Failed to connect to nodes in shard %v", shardID)
	}

	// shardCollations protocol
	if err := n.SubscribeCollation(spanctx, shardID); err != nil {
		logger.FinishWithErr(spanctx, fmt.Errorf("Failed to subscribe to collation in shard %v", shardID))
		logger.Errorf("Failed to subscribe to collation in shard %v", shardID)
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
	if err := n.discovery.Advertise(spanctx, shardID); err != nil {
		logger.SetErr(spanctx, fmt.Errorf("Failed to advertise subscription to shard %v", shardID))
		logger.Errorf("Failed to advertise subscription to shard %v", shardID)
		return err
	}

	// listeningShards protocol
	// TODO: should we remove some peers in this shard?

	// shardCollations protocol
	n.UnsubscribeCollation(spanctx, shardID)

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
		logger.Errorf("Failed to subscribe to topic: %s", topic)
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

func (n *ShardManager) UnsubscribeTopic(ctx context.Context, topic string) {
	// Add span for UnsubscribeTopic of ShardManager
	spanctx := logger.Start(ctx, "ShardManager.UnsubscribeTopic")
	defer logger.Finish(spanctx)

	n.subsLock.RLock()
	sub, prs := n.subs[topic]
	n.subsLock.RUnlock()
	if !prs {
		return
	}
	// Tag topic to unsubscribe to
	logger.SetTag(spanctx, "Topic", topic)

	sub.Cancel()
	n.subsLock.Lock()
	delete(n.subs, topic)
	n.subsLock.Unlock()
	return
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

func (n *ShardManager) UnsubscribeListeningShards() {
	n.UnsubscribeTopic(context.Background(), listeningShardTopic)
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

func (n *ShardManager) SubscribeCollation(ctx context.Context, shardID ShardIDType) error {
	// Add span for SubscribeCollation of ShardManager
	spanctx := logger.Start(ctx, "ShardManager.SubscribeCollation")
	defer logger.Finish(spanctx)

	topic := getCollationsTopic(shardID)
	return n.SubscribeGeneral(spanctx, topic)
}

func (n *ShardManager) makeGeneralValidator(topic string) TopicValidator {
	return func(ctx context.Context, msg *pubsub.Message) bool {
		typedMessage := &pbmsg.MessageWithType{}
		err := extractProtoMsg(msg, typedMessage)
		if err != nil {
			return false
		}
		// FIXME: if no eventNotifier, just skip the verification
		if n.eventNotifier != nil {
			validityBytes, err := n.eventNotifier.Receive(
				msg.GetFrom(),
				int(typedMessage.MsgType),
				typedMessage.Data,
			)
			if err != nil {
				return false
			}
			// TODO: `retVal` from `n.eventNotifier.Receive` should be a bool.
			//		 validityByte == b"\x00" means false, otherwise true.
			if len(validityBytes) != 1 || validityBytes[0] == 0 {
				return false
			}
		}
		return true
	}
}

func (n *ShardManager) makeGeneralHandler() TopicHandler {
	return func(ctx context.Context, msg *pubsub.Message) {
		// do nothing now
	}
}

func (n *ShardManager) SubscribeGeneral(ctx context.Context, topic string) error {
	handler := n.makeGeneralHandler()
	validator := n.makeGeneralValidator(topic)
	return n.SubscribeTopic(ctx, topic, handler, validator)
}

func (n *ShardManager) UnsubscribeCollation(ctx context.Context, shardID ShardIDType) {
	// Add span for UnsubscribeCollation of ShardManager
	spanctx := logger.Start(ctx, "ShardManager.UnsubscribeCollation")
	defer logger.Finish(spanctx)

	topic := getCollationsTopic(shardID)
	n.UnsubscribeTopic(spanctx, topic)
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
		logger.Errorf("Failed to broadcast collation message")
		return err
	}

	// Tag collation period and blobs if nothing goes wrong
	logger.SetTag(spanctx, "Period", period)
	logger.SetTag(spanctx, "Blobs", blobs)
	return nil
}

// FIXME: in this go layer, we shouldn't have knowledge that what we are broadcasting
//		  This should be handled in the upper side, e.g. the Python host.
// 		  However, this code makes us easier to test without spinning up the "remote host"
//		  We should remove these "collation" specific messages when the host side is ready.
func (n *ShardManager) broadcastCollationMessage(collation *pbmsg.Collation) error {
	if !n.IsCollationSubscribed(collation.GetShardID()) {
		return fmt.Errorf("broadcasting to a not subscribed shard")
	}
	collationsTopic := getCollationsTopic(collation.ShardID)
	dataBytes, err := proto.Marshal(collation)
	// FIXME: we shouldn't know MsgType in Go code.
	typedMessage := &pbmsg.MessageWithType{
		MsgType: PBInt(typeCollation),
		Data:    dataBytes,
	}
	msgBytes, err := proto.Marshal(typedMessage)
	if err != nil {
		logger.Errorf("Failed to encode protobuf message: %v, err: %v", collation, err)
		return err
	}
	err = n.pubsubService.Publish(collationsTopic, msgBytes)
	if err != nil {
		logger.Errorf(
			"Failed to publish data '%v' to topic '%v', err: %v",
			msgBytes,
			collationsTopic,
			err,
		)
		return err
	}
	return nil
}

// TODO: beacon chain
