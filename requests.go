package main

import (
	"bufio"
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"

	pbmsg "github.com/ethresearch/sharding-p2p-poc/pb/message"

	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"

	protocol "github.com/libp2p/go-libp2p-protocol"

	protobufCodec "github.com/multiformats/go-multicodec/protobuf"
)

// RequestProtocol type
type RequestProtocol struct {
	node *Node
}

const shardPeerRequestProtocol = protocol.ID("/shardPeerRequest/1.0.0")
const generalRequestProtocol = protocol.ID("/generalRequest/1.0.0")

// NewRequestProtocol defines the request protocol, which allows others to query data
func NewRequestProtocol(node *Node) *RequestProtocol {
	p := &RequestProtocol{
		node: node,
	}
	node.SetStreamHandler(shardPeerRequestProtocol, p.onShardPeerRequest)
	node.SetStreamHandler(generalRequestProtocol, p.onGeneralRequest)
	return p
}

// helper method - reads a protobuf go data object from a network stream
// data: reference of protobuf go data object(not the object itself)
// s: network stream to read the data from
func readProtoMessage(data proto.Message, s inet.Stream) error {
	decoder := protobufCodec.Multicodec(nil).Decoder(bufio.NewReader(s))
	return decoder.Decode(data)
}

// helper method - writes a protobuf go data object to a network stream
// data: reference of protobuf go data object to send (not the object itself)
// s: network stream to write the data to
func sendProtoMessage(data proto.Message, s inet.Stream) error {
	writer := bufio.NewWriter(s)
	enc := protobufCodec.Multicodec(nil).Encoder(writer)
	err := enc.Encode(data)
	if err != nil {
		return err
	}
	writer.Flush()
	return nil
}

func (p *RequestProtocol) onShardPeerRequest(s inet.Stream) {
	defer inet.FullClose(s)
	req := &pbmsg.ShardPeerRequest{}
	if err := readProtoMessage(req, s); err != nil {
		return
	}
	shardPeers := make(map[ShardIDType]*pbmsg.Peers)
	for _, shardID := range req.ShardIDs {
		peerIDs := p.node.shardPrefTable.GetPeersInShard(shardID)
		shardPeers[shardID] = peerIDsToPBPeers(peerIDs)
	}

	res := &pbmsg.ShardPeerResponse{
		Response:   &pbmsg.Response{Status: pbmsg.Response_SUCCESS},
		ShardPeers: shardPeers,
	}
	if err := sendProtoMessage(res, s); err != nil {
		logger.Errorf("onShardPeerRequest: failed to send proto message '%v', err: %v", res, err)
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
	defer s.Close()

	req := &pbmsg.ShardPeerRequest{
		ShardIDs: shardIDs,
	}
	if err := sendProtoMessage(req, s); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	res := &pbmsg.ShardPeerResponse{}
	if err := readProtoMessage(res, s); err != nil {
		return nil, fmt.Errorf("failed to read response proto")
	}

	shardPeers := make(map[ShardIDType][]peer.ID)
	for shardID, peers := range res.ShardPeers {
		peerIDs, err := pbPeersToPeerIDs(peers)
		if err != nil {
			return nil, fmt.Errorf(
				"Failed to convert peer ID from PB string to peerID format, err: %v",
				err,
			)
		}
		shardPeers[shardID] = peerIDs
	}
	return shardPeers, nil
}

func (p *RequestProtocol) onGeneralRequest(s inet.Stream) {
	defer inet.FullClose(s)
	req := &pbmsg.GeneralRequest{}
	if err := readProtoMessage(req, s); err != nil {
		logger.Errorf(
			"onGeneralRequest: failed to read proto message, reason=%v",
			err,
		)
		return
	}
	if p.node.eventNotifier == nil {
		logger.Error("onGeneralRequest: no eventNotifier set")
		return
	}
	peerID := s.Conn().RemotePeer()
	dataBytes, err := p.node.eventNotifier.Receive(peerID, int(req.MsgType), req.Data)
	if err != nil {
		logger.Errorf(
			"onGeneralRequest: failed to read proto message, reason=%v",
			err,
		)
		return
	}
	resp := &pbmsg.GeneralResponse{
		Data: dataBytes,
	}
	if err := sendProtoMessage(resp, s); err != nil {
		logger.Errorf(
			"onGeneralRequest: failed to send proto message %v, reason=%v",
			resp,
			err,
		)
	}
}

func (p *RequestProtocol) generalRequest(
	ctx context.Context,
	peerID peer.ID,
	msgType int,
	data []byte) ([]byte, error) {
	s, err := p.node.NewStream(
		ctx,
		peerID,
		generalRequestProtocol,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open new stream %v", err)
	}
	req := &pbmsg.GeneralRequest{
		MsgType: PBInt(msgType),
		Data:    data,
	}
	if err := sendProtoMessage(req, s); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	resp := &pbmsg.GeneralResponse{}
	if err := readProtoMessage(resp, s); err != nil {
		return nil, fmt.Errorf("failed to read response proto")
	}
	return resp.Data, nil
}
