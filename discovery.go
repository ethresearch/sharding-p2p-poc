package main

import (
	"context"

	pstore "github.com/libp2p/go-libp2p-peerstore"
)

type Discovery interface {
	Advertise(ctx context.Context, topic string) error
	FindPeers(ctx context.Context, topic string) ([]pstore.PeerInfo, error)
}
