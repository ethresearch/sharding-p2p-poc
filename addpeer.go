package main

import (
	"context"
	"fmt"
	"log"

	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	ma "github.com/multiformats/go-multiaddr"
	opentracing "github.com/opentracing/opentracing-go"

	pbmsg "github.com/ethresearch/sharding-p2p-poc/pb/message"
)

const addPeerRequest = "/addPeer/request/0.0.1"

type AddPeerProtocol struct {
	node *Node // local host
}

func parseAddr(addrString string) (peerID peer.ID, protocolAddr ma.Multiaddr) {
	// The following code extracts target's the peer ID from the
	// given multiaddress
	ipfsaddr, err := ma.NewMultiaddr(addrString) // ipfsaddr=/ip4/127.0.0.1/tcp/10000/ipfs/QmVmDaabYcS3pn23KaFjkdw6hkReUUma8sBKqSDHrPYPd2
	if err != nil {
		log.Fatalln(err)
	}

	pid, err := ipfsaddr.ValueForProtocol(ma.P_IPFS) // pid=QmVmDaabYcS3pn23KaFjkdw6hkReUUma8sBKqSDHrPYPd2
	if err != nil {
		log.Fatalln(err)
	}

	peerid, err := peer.IDB58Decode(pid) // peerid=<peer.ID VmDaab>
	if err != nil {
		log.Fatalln(err)
	}

	// Decapsulate the /ipfs/<peerID> part from the target
	// /ip4/<a.b.c.d>/ipfs/<peer> becomes /ip4/<a.b.c.d>
	targetPeerAddr, _ := ma.NewMultiaddr(
		fmt.Sprintf("/ipfs/%s", peer.IDB58Encode(peerid)),
	)
	targetAddr := ipfsaddr.Decapsulate(targetPeerAddr)
	return peerid, targetAddr
}

func NewAddPeerProtocol(node *Node) *AddPeerProtocol {
	p := &AddPeerProtocol{
		node: node,
	}
	node.SetStreamHandler(addPeerRequest, p.onRequest)
	return p
}

// remote peer requests handler
func (p *AddPeerProtocol) onRequest(s inet.Stream) {
	defer inet.FullClose(s)
	// get request data
	data := &pbmsg.AddPeerRequest{}
	if err := readProtoMessage(data, s); err != nil {
		return
	}

	log.Printf(
		"%s: Received addPeer request from %s. Message: %s",
		p.node.Name(),
		s.Conn().RemotePeer(),
		data.Message,
	)

	// TODO: add logics for handshake and initialization, e.g. asking for shard peers
	//		 if nothing is wrong, accept this peer.
	resp := &pbmsg.AddPeerResponse{
		Response: &pbmsg.Response{Status: pbmsg.Response_SUCCESS},
	}

	p.node.Peerstore().AddAddr(
		s.Conn().RemotePeer(),
		s.Conn().RemoteMultiaddr(),
		pstore.PermanentAddrTTL,
	)
	sendProtoMessage(resp, s)
}

func (p *AddPeerProtocol) AddPeer(ctx context.Context, peerAddr string) error {
	// Add span for AddPeer of AddPeerProtocol
	span, _ := opentracing.StartSpanFromContext(ctx, "AddPeerProtocol.AddPeer")
	defer span.Finish()

	peerid, targetAddr := parseAddr(peerAddr)
	p.node.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)
	// create message data
	req := &pbmsg.AddPeerRequest{
		Message: fmt.Sprintf("AddPeer from %s", p.node.Name()),
	}

	s, err := p.node.NewStream(ctx, peerid, addPeerRequest)
	if err != nil {
		return err
	}

	return sendProtoMessage(req, s)
}
