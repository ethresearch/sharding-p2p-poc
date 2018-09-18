package main

import (
	"fmt"
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"
)

const byteSize = 8 // in bits

type ListeningShards struct {
	shardBits []byte
}

func NewListeningShards() *ListeningShards {
	return &ListeningShards{
		shardBits: make([]byte, (numShards/8)+1),
	}
}

func shardIDToBitIndex(shardID ShardIDType) (byte, byte, error) {
	if shardID >= numShards {
		return 0, 0, fmt.Errorf("Wrong shardID %v", shardID)
	}
	byteIndex := byte(shardID / byteSize)
	bitIndex := byte(shardID % byteSize)
	return byteIndex, bitIndex, nil
}

func (ls *ListeningShards) unsetShard(shardID ShardIDType) error {
	byteIndex, bitIndex, err := shardIDToBitIndex(shardID)
	if err != nil {
		return fmt.Errorf("")
	}
	if int(byteIndex) >= len(ls.shardBits) {
		return fmt.Errorf(
			"(byteIndex=%v) >= (len(shardBits)=%v)",
			byteIndex,
			len(ls.shardBits),
		)
	}
	ls.shardBits[byteIndex] &= (^(1 << bitIndex))
	return nil
}

func (ls *ListeningShards) setShard(shardID ShardIDType) error {
	byteIndex, bitIndex, err := shardIDToBitIndex(shardID)
	if err != nil {
		return fmt.Errorf("")
	}
	if int(byteIndex) >= len(ls.shardBits) {
		return fmt.Errorf(
			"(byteIndex=%v) >= (len(shardBits)=%v)",
			byteIndex,
			len(ls.shardBits),
		)
	}
	ls.shardBits[byteIndex] |= (1 << bitIndex)
	return nil
}

func (ls *ListeningShards) isShardSet(shardID ShardIDType) bool {
	byteIndex, bitIndex, err := shardIDToBitIndex(shardID)
	if err != nil {
		fmt.Errorf("")
	}
	index := ls.shardBits[byteIndex] & (1 << bitIndex)
	return index != 0
}

func (ls *ListeningShards) getShards() []ShardIDType {
	shards := []ShardIDType{}
	for shardID := ShardIDType(0); shardID < numShards; shardID++ {
		if ls.isShardSet(shardID) {
			shards = append(shards, shardID)
		}
	}
	return shards
}

func (ls *ListeningShards) fromSlice(shards []ShardIDType) *ListeningShards {
	for _, shardID := range shards {
		ls.setShard(shardID)
	}
	return ls
}

// TODO: need checks against the format of bytes
func (ls *ListeningShards) fromBytes(bytes []byte) *ListeningShards {
	ls.shardBits = bytes
	return ls
}

func (ls *ListeningShards) toBytes() []byte {
	return ls.shardBits
}

// ShardPrefTable manages peers' shard preference
type ShardPrefTable struct {
	shardPrefMap map[peer.ID]*ListeningShards
	lock         sync.RWMutex
}

func NewShardPrefTable() *ShardPrefTable {
	return &ShardPrefTable{
		shardPrefMap: make(map[peer.ID]*ListeningShards),
		lock:         sync.RWMutex{},
	}
}

func (n *ShardPrefTable) isPeerRecorded(peerID peer.ID) bool {
	n.lock.RLock()
	defer n.lock.RUnlock()
	_, prs := n.shardPrefMap[peerID]
	return prs
}

func (n *ShardPrefTable) IsPeerListeningShard(peerID peer.ID, shardID ShardIDType) bool {
	if !n.isPeerRecorded(peerID) {
		return false
	}
	n.lock.RLock()
	shardPref := n.shardPrefMap[peerID]
	n.lock.RUnlock()
	return shardPref.isShardSet(shardID)
}

func (n *ShardPrefTable) AddPeerListeningShard(peerID peer.ID, shardID ShardIDType) {
	if shardID >= numShards {
		return
	}
	n.lock.RLock()
	shardPref, prs := n.shardPrefMap[peerID]
	n.lock.RUnlock()
	if !prs {
		shardPref = NewListeningShards()
	}
	shardPref.setShard(shardID)
	n.lock.Lock()
	defer n.lock.Unlock()
	n.shardPrefMap[peerID] = shardPref
}

func (n *ShardPrefTable) RemovePeerListeningShard(peerID peer.ID, shardID ShardIDType) {
	if !n.isPeerRecorded(peerID) {
		return
	}
	if !n.IsPeerListeningShard(peerID, shardID) {
		return
	}
	n.lock.Lock()
	defer n.lock.Unlock()
	n.shardPrefMap[peerID].unsetShard(shardID)
}

func (n *ShardPrefTable) GetPeerListeningShard(peerID peer.ID) *ListeningShards {
	if !n.isPeerRecorded(peerID) {
		return NewListeningShards()
	}
	n.lock.RLock()
	defer n.lock.RUnlock()
	return n.shardPrefMap[peerID]
}

func (n *ShardPrefTable) GetPeerListeningShardSlice(peerID peer.ID) []ShardIDType {
	if !n.isPeerRecorded(peerID) {
		return []ShardIDType{}
	}
	n.lock.RLock()
	defer n.lock.RUnlock()
	return n.shardPrefMap[peerID].getShards()
}

func (n *ShardPrefTable) SetPeerListeningShard(peerID peer.ID, ls *ListeningShards) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.shardPrefMap[peerID] = ls
	return
}

func (n *ShardPrefTable) GetPeersInShard(shardID ShardIDType) []peer.ID {
	n.lock.RLock()
	defer n.lock.RUnlock()
	peers := []peer.ID{}
	for peerID, listeningShards := range n.shardPrefMap {
		if listeningShards.isShardSet(shardID) {
			peers = append(peers, peerID)
		}
	}
	return peers
}
