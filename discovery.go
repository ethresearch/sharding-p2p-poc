package main

import (
	"context"
	"fmt"

	pubsub "github.com/libp2p/go-floodsub"
	host "github.com/libp2p/go-libp2p-host"
	pstore "github.com/libp2p/go-libp2p-peerstore"
)

type Discovery interface {
	Advertise(ctx context.Context, shardID ShardIDType) error
	FindPeers(ctx context.Context, shardID ShardIDType) ([]pstore.PeerInfo, error)
}

type GlobalTable struct {
	host           host.Host
	pubsubService  *pubsub.PubSub
	shardPrefTable *ShardPrefTable
}

func NewGlobalTable(ctx context.Context, h host.Host, pubsubService *pubsub.PubSub, shardPrefTable *ShardPrefTable) *GlobalTable {
	return &GlobalTable{
		host:           h,
		pubsubService:  pubsubService,
		shardPrefTable: shardPrefTable,
	}
}

func (gt *GlobalTable) Advertise(ctx context.Context, shardID ShardIDType) error {
	// Modify local table
	gt.shardPrefTable.AddPeerListeningShard(gt.host.ID(), shardID)

	// Publish our preference in local table
	selfListeningShards := gt.shardPrefTable.GetPeerListeningShard(gt.host.ID()).toBytes()
	if err := gt.pubsubService.Publish(listeningShardTopic, selfListeningShards); err != nil {
		logger.Error(fmt.Errorf("Failed to publish listening shards, err: %v", err))
		return err
	}

	return nil
}

func (gt *GlobalTable) FindPeers(ctx context.Context, shardID ShardIDType) ([]pstore.PeerInfo, error) {
	// Get peer ID from local table and convert to PeerInfo format
	peerIDs := gt.shardPrefTable.GetPeersInShard(shardID)
	pinfos := []pstore.PeerInfo{}
	for _, peerID := range peerIDs {
		// Exclude ourself
		if peerID == gt.host.ID() {
			continue
		}
		pi := gt.host.Peerstore().PeerInfo(peerID)
		pinfos = append(pinfos, pi)
	}
	return pinfos, nil
}
