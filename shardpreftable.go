package main

import (
	"sync"

	peer "github.com/libp2p/go-libp2p-peer"
)

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
