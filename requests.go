package main

import (
	"bufio"
	"context"
	"fmt"
	"log"

	pbmsg "github.com/ethresearch/sharding-p2p-poc/pb/message"
	"github.com/golang/protobuf/proto"

	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"

	protocol "github.com/libp2p/go-libp2p-protocol"
	protobufCodec "github.com/multiformats/go-multicodec/protobuf"
)

// RequestProtocol type
type RequestProtocol struct {
	node *Node
}

const collationRequestProtocol = protocol.ID("/collationRequest/1.0.0")
const shardPeerRequestProtocol = protocol.ID("/shardPeerRequest/1.0.0")

// NewRequestProtocol defines the request protocol, which allows others to query data
func NewRequestProtocol(node *Node) *RequestProtocol {
	p := &RequestProtocol{
		node: node,
	}
	node.SetStreamHandler(collationRequestProtocol, p.onCollationRequest)
	node.SetStreamHandler(shardPeerRequestProtocol, p.onShardPeerRequest)
	return p
}

// helper method - reads a protobuf go data object from a network stream
// data: reference of protobuf go data object(not the object itself)
// s: network stream to read the data from
func readProtoMessage(data proto.Message, s inet.Stream) bool {
	decoder := protobufCodec.Multicodec(nil).Decoder(bufio.NewReader(s))
	err := decoder.Decode(data)
	if err != nil {
		log.Println("readProtoMessage: ", err)
		return false
	}
	return true
}

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

func (p *RequestProtocol) onShardPeerRequest(s inet.Stream) {
	defer inet.FullClose(s)
	req := &pbmsg.ShardPeerRequest{}
	if !readProtoMessage(req, s) {
		return
	}
	shardPeers := make(map[ShardIDType]*pbmsg.ShardPeerResponse_Peers)
	for _, shardID := range req.ShardIDs {
		peerIDs := p.node.shardPrefTable.GetPeersInShard(shardID)
		peerIDStrings := []string{}
		for _, peerID := range peerIDs {
			peerIDStrings = append(peerIDStrings, peerID.Pretty())
		}
		shardPeers[shardID] = &pbmsg.ShardPeerResponse_Peers{
			Peers: peerIDStrings,
		}
	}

	res := &pbmsg.ShardPeerResponse{
		Response:   &pbmsg.Response{Status: pbmsg.Response_SUCCESS},
		ShardPeers: shardPeers,
	}
	if !sendProtoMessage(res, s) {
		log.Printf("onShardPeerRequest: failed to send proto message %v", res)
		return
	}
}

func (p *RequestProtocol) requestShardPeer(
	ctx context.Context,
	peerID peer.ID,
	shardIDs []ShardIDType) (map[ShardIDType][]peer.ID, error) {
	s, err := p.node.NewStream(
		ctx,
		peerID,
		shardPeerRequestProtocol,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open new stream")
	}
	req := &pbmsg.ShardPeerRequest{
		ShardIDs: shardIDs,
	}
	if !sendProtoMessage(req, s) {
		return nil, fmt.Errorf("failed to send request")
	}
	res := &pbmsg.ShardPeerResponse{}
	if !readProtoMessage(res, s) {
		s.Close()
		return nil, fmt.Errorf("failed to read response proto")
	}
	shardPeers := make(map[ShardIDType][]peer.ID)
	for shardID, peers := range res.ShardPeers {
		peerIDs := []peer.ID{}
		for _, peerString := range peers.Peers {
			peerID, err := peer.IDB58Decode(peerString)
			if err != nil {
				return nil, fmt.Errorf("error occurred when parsing peerIDs")
			}
			peerIDs = append(peerIDs, peerID)
		}
		shardPeers[shardID] = peerIDs
	}
	return shardPeers, nil
}

// collation request
func (p *RequestProtocol) onCollationRequest(s inet.Stream) {
	defer inet.FullClose(s)
	// reject if the sender is not a peer
	data := &pbmsg.CollationRequest{}
	if !readProtoMessage(data, s) {
		return
	}
	// FIXME: add checks
	var collation *pbmsg.Collation
	collation, err := p.node.getCollation(
		ShardIDType(data.GetShardID()),
		int(data.GetPeriod()),
		data.GetHash(),
	)
	var collationResp *pbmsg.CollationResponse
	if err != nil {
		collationResp = &pbmsg.CollationResponse{
			Response:  &pbmsg.Response{Status: pbmsg.Response_FAILURE},
			Collation: nil,
		}
	} else {
		collationResp = &pbmsg.CollationResponse{
			Response:  &pbmsg.Response{Status: pbmsg.Response_SUCCESS},
			Collation: collation,
		}
	}
	if !sendProtoMessage(collationResp, s) {
		log.Printf("onCollationRequest: failed to send proto message %v", collationResp)
		return
	}
}

func (p *RequestProtocol) requestCollation(
	ctx context.Context,
	peerID peer.ID,
	shardID ShardIDType,
	period int) (*pbmsg.Collation, error) {
	s, err := p.node.NewStream(
		ctx,
		peerID,
		collationRequestProtocol,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open new stream %v", err)
	}
	req := &pbmsg.CollationRequest{
		ShardID: PBInt(shardID),
		Period:  PBInt(period),
	}
	if !sendProtoMessage(req, s) {
		return nil, fmt.Errorf("failed to send request")
	}
	data := &pbmsg.CollationResponse{}
	if !readProtoMessage(data, s) {
		return nil, fmt.Errorf("failed to read response proto")
	}
	return data.Collation, nil
}
