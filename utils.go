package main

import (
	"strings"

	peer "github.com/libp2p/go-libp2p-peer"
	"golang.org/x/crypto/sha3"

	pbmsg "github.com/ethresearch/sharding-p2p-poc/pb/message"

	"github.com/golang/protobuf/proto"
)

func keccak(msg proto.Message) ([]byte, error) {
	dataInBytes, err := proto.Marshal(msg)
	if err != nil {
		logger.Errorf("Failed to encode protobuf message: %v, err: %v", msg, err)
		return nil, err
	}
	h := sha3.NewLegacyKeccak256()
	if _, err := h.Write(dataInBytes); err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

func peerIDToString(peerID peer.ID) string {
	return peerID.Pretty()
}
func stringToPeerID(peerIDStr string) (peer.ID, error) {
	peerID, err := peer.IDB58Decode(peerIDStr)
	if err != nil {
		return "", err
	}
	return peerID, nil
}

func peersStringToPeerIDs(peersStr []string) ([]peer.ID, error) {
	peerIDs := []peer.ID{}
	for _, peerStr := range peersStr {
		peerID, err := stringToPeerID(peerStr)
		if err != nil {
			return nil, err
		}
		peerIDs = append(peerIDs, peerID)
	}
	return peerIDs, nil
}

func peerIDsToPeersString(peerIDs []peer.ID) []string {
	peerStrs := make([]string, 0)
	for _, peerID := range peerIDs {
		peerStrs = append(peerStrs, peerIDToString(peerID))
	}
	return peerStrs
}

func pbPeersToPeerIDs(msg *pbmsg.Peers) ([]peer.ID, error) {
	return peersStringToPeerIDs(msg.Peers)
}

func peerIDsToPBPeers(peerIDs []peer.ID) *pbmsg.Peers {
	return &pbmsg.Peers{Peers: peerIDsToPeersString(peerIDs)}
}

func shardTopicToShardID(topic string) string {
	end := len(collationTopicFmt) - 2
	return strings.Replace(topic, collationTopicFmt[0:end], "", -1)
}
