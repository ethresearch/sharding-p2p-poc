package main

import (
	"context"

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
