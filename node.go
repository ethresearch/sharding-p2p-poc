package main

import (
	"context"
	"fmt"

	pubsub "github.com/libp2p/go-floodsub"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	ma "github.com/multiformats/go-multiaddr"
)

type Node struct {
	host.Host        // lib-p2p host
	discovery        Discovery
	*RequestProtocol // for peers to request data
	*ShardManager

	ctx context.Context
}

// NewNode creates a new node with its implemented protocols
func NewNode(ctx context.Context, h host.Host, eventNotifier EventNotifier) *Node {
	shardPrefTable := NewShardPrefTable()
	pubsubService, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		logger.Fatalf("Failed to create new pubsub service, err: %v", err)
	}
	node := &Node{
		Host:      h,
		ctx:       ctx,
		discovery: NewGlobalTable(ctx, h, pubsubService, shardPrefTable),
	}
	node.RequestProtocol = NewRequestProtocol(node)
	node.ShardManager = NewShardManager(ctx, node, pubsubService, eventNotifier, node.discovery, shardPrefTable)

	return node
}

func (n *Node) GetFullAddr() string {
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", n.ID().Pretty()))

	// Now we can build a full multiaddress to reach this host
	// by encapsulating both addresses:
	addr := n.Addrs()[0]
	fullAddr := addr.Encapsulate(hostAddr)
	return fullAddr.String()
}

// TODO: should be changed to `Knows` and `HasConnections`
func (n *Node) IsPeer(peerID peer.ID) bool {
	for _, value := range n.Peerstore().Peers() {
		if value == peerID {
			return true
		}
	}
	return false
}
