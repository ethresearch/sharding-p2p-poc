package main

import (
	"context"

	pstore "github.com/libp2p/go-libp2p-peerstore"
)

type Discovery interface {
	Advertise(ctx context.Context, shardID ShardIDType) error
	FindPeers(ctx context.Context, shardID ShardIDType) ([]pstore.PeerInfo, error)
}
