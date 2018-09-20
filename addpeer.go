package main

import (
	"context"
	"fmt"
	"log"

	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	protocol "github.com/libp2p/go-libp2p-protocol"
	ma "github.com/multiformats/go-multiaddr"
)

const addPeerRequestProtocol = protocol.ID("/addPeer/request/0.0.1")

type AddPeerProtocol struct {
	node *Node // local host
}

func parseAddr(addrString string) (peer.ID, ma.Multiaddr, error) {
	// The following code extracts target's the peer ID from the
	// given multiaddress
	ipfsaddr, err := ma.NewMultiaddr(addrString) // ipfsaddr=/ip4/127.0.0.1/tcp/10000/ipfs/QmVmDaabYcS3pn23KaFjkdw6hkReUUma8sBKqSDHrPYPd2
	if err != nil {
		return "", nil, err
	}

	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS) // pid=QmVmDaabYcS3pn23KaFjkdw6hkReUUma8sBKqSDHrPYPd2
	if err != nil {
		return "", nil, err
	}

	peerid, err := peer.IDB58Decode(pid) // peerid=<peer.ID VmDaab>
	if err != nil {
		return "", nil, err
	}

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)),
	)
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)
	return peerid, targetAddr, nil
}

func NewAddPeerProtocol(node *Node) *AddPeerProtocol {
	p := &AddPeerProtocol{
		node: node,
	}
	return p
}

func (p *AddPeerProtocol) AddPeer(ctx context.Context, peerAddr string) error {
	// Add span for AddPeer of AddPeerProtocol
	spanctx := logger.Start(ctx, "AddPeerProtocol.AddPeer")
	defer logger.Finish(spanctx)

	peerid, targetAddr, err := parseAddr(peerAddr)
	if err != nil {
		logger.FinishWithErr(spanctx, fmt.Errorf("Failed to parse peer address: %s, err: %v", peerAddr, err))
		log.Println(err)
		return err
	}
	p.node.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)

	if err := p.node.Connect(ctx, p.node.Peerstore().PeerInfo(peerid)); err != nil {
		logger.FinishWithErr(spanctx, fmt.Errorf("Failed to connect to peer %v, err: %v", peerid, err))
		log.Println(err)
		return err
	}

	// TODO: Execute protocol required when connection with new peer is built, e.g.
	// s, err := p.node.NewStream(ctx, peerid, exchangeTopicsProtocol)

	return nil
}
