package main

import (
	"bufio"
	"context"
	"fmt"
	"log"

	"github.com/gogo/protobuf/proto"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"

	pbmsg "github.com/libp2p/go-libp2p/examples/minimal/pb"

	ma "github.com/multiformats/go-multiaddr"
	protobufCodec "github.com/multiformats/go-multicodec/protobuf"
)

// pattern: /protocol-name/request-or-response-message/version
const addPeerRequest = "/addPeer/request/0.0.1"
const addPeerResponse = "/addPeer/response/0.0.1"

// AddPeerProtocol type
type AddPeerProtocol struct {
	node *Node     // local host
	done chan bool // only for demo purposes to stop main from terminating
}

func NewAddPeerProtocol(node *Node) *AddPeerProtocol {
	p := &AddPeerProtocol{
		node: node,
		done: make(chan bool),
	}
	node.SetStreamHandler(addPeerRequest, p.onRequest)
	node.SetStreamHandler(addPeerResponse, p.onResponse)
	return p
}

// remote peer requests handler
func (p *AddPeerProtocol) onRequest(s inet.Stream) {
	// get request data
	data := &pbmsg.AddPeerRequest{}
	decoder := protobufCodec.Multicodec(nil).Decoder(bufio.NewReader(s))
	err := decoder.Decode(data)
	if err != nil {
		log.Println(err)
		return
	}

	log.Printf(
		"%s: Received addPeer request from %s. Message: %s",
		p.node.Name(),
		s.Conn().RemotePeer(),
		data.Message,
	)

	// generate response message
	log.Printf(
		"%s: Sending addPeer response to %s",
		p.node.Name(),
		s.Conn().RemotePeer(),
	)

	resp := &pbmsg.AddPeerResponse{
		Success: true,
	}

	p.node.Peerstore().AddAddr(
		s.Conn().RemotePeer(),
		s.Conn().RemoteMultiaddr(),
		pstore.PermanentAddrTTL,
	)

	// send the response
	s, respErr := p.node.NewStream(context.Background(), s.Conn().RemotePeer(), addPeerResponse)
	if respErr != nil {
		log.Println(respErr)
		return
	}

	if ok := sendProtoMessage(resp, s); ok {
		log.Printf("%s: AddPeer response to %s sent.", p.node.Name(), s.Conn().RemotePeer().String())
	}
}

// remote addPeer response handler
func (p *AddPeerProtocol) onResponse(s inet.Stream) {
	data := &pbmsg.AddPeerResponse{}
	decoder := protobufCodec.Multicodec(nil).Decoder(bufio.NewReader(s))
	err := decoder.Decode(data)
	if err != nil {
		return
	}

	log.Printf(
		"%s: Received addPeer response from %s, result=%v",
		p.node.Name(),
		s.Conn().RemotePeer(),
		data.Success,
	)
	p.done <- true
}

func (p *AddPeerProtocol) AddPeer(peerAddr string) bool {
	peerid, targetAddr := parseAddr(peerAddr)
	log.Printf("%s: Sending addPeer to: %s....", p.node.Name(), peerid)
	p.node.Peerstore().AddAddr(peerid, targetAddr, pstore.PermanentAddrTTL)
	// create message data
	req := &pbmsg.AddPeerRequest{
		Message: fmt.Sprintf("AddPeer from %s", p.node.Name()),
	}

	s, err := p.node.NewStream(context.Background(), peerid, addPeerRequest)
	if err != nil {
		log.Println(err)
		return false
	}

	if ok := sendProtoMessage(req, s); !ok {
		return false
	}

	// store ref request so response handler has access to it
	// p.requests[req.MessageData.Id] = req
	// log.Printf("%s: AddPeer to: %s was sent. Message Id: %s, Message: %s", p.node.Name(), host.Name(), req.MessageData.Id, req.Message)
	return true

}

//
// utils
//

// helper method - writes a protobuf go data object to a network stream
// data: reference of protobuf go data object to send (not the object itself)
// s: network stream to write the data to
func sendProtoMessage(data proto.Message, s inet.Stream) bool {
	writer := bufio.NewWriter(s)
	enc := protobufCodec.Multicodec(nil).Encoder(writer)
	err := enc.Encode(data)
	if err != nil {
		log.Println(err)
		return false
	}
	writer.Flush()
	return true
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
