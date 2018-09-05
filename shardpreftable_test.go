package main

import (
	"testing"

	peer "github.com/libp2p/go-libp2p-peer"
)

func TestListeningShards(t *testing.T) {
	ls := NewListeningShards()
	lsSlice := ls.getShards()
	if len(lsSlice) != 0 {
		t.Error()
	}

	ls.setShard(1)
	lsSlice = ls.getShards()
	if (len(lsSlice) != 1) || lsSlice[0] != ShardIDType(1) {
		t.Error()
	}
	ls.setShard(42)
	if len(ls.getShards()) != 2 {
		t.Error()
	}
	// test `toBytes` and `fromBytes`
	bytes := ls.toBytes()
	lsNew := ls.fromBytes(bytes)
	if len(ls.getShards()) != len(lsNew.getShards()) {
		t.Error()
	}
	lsNewSlice := lsNew.getShards()
	for index, value := range ls.getShards() {
		if value != lsNewSlice[index] {
			t.Error()
		}
	}
}

func TestShardPrefTable(t *testing.T) {
	shardPrefTable := NewShardPrefTable()
	arbitraryPeerID := peer.ID("123456")

	var testingShardID ShardIDType = 42
	if len(shardPrefTable.GetPeerListeningShardSlice(arbitraryPeerID)) != 0 {
		t.Errorf("Peer %v should not be listening to any shard", arbitraryPeerID)
	}
	if shardPrefTable.IsPeerListeningShard(arbitraryPeerID, testingShardID) {
		t.Errorf("Peer %v should not be listening to shard %v", arbitraryPeerID, testingShardID)
	}
	shardPrefTable.AddPeerListeningShard(arbitraryPeerID, testingShardID)
	if !shardPrefTable.IsPeerListeningShard(arbitraryPeerID, testingShardID) {
		t.Errorf("Peer %v should be listening to shard %v", arbitraryPeerID, testingShardID)
	}
	shardPrefTable.AddPeerListeningShard(arbitraryPeerID, numShards)
	if shardPrefTable.IsPeerListeningShard(arbitraryPeerID, numShards) {
		t.Errorf(
			"Peer %v should not be able to listen to shardID bigger than %v",
			arbitraryPeerID,
			numShards,
		)
	}
	// listen to multiple shards
	anotherShardID := testingShardID + 1 // notice that it should be less than `numShards`
	shardPrefTable.AddPeerListeningShard(arbitraryPeerID, anotherShardID)
	if !shardPrefTable.IsPeerListeningShard(arbitraryPeerID, anotherShardID) {
		t.Errorf("Peer %v should be listening to shard %v", arbitraryPeerID, testingShardID)
	}
	if len(shardPrefTable.GetPeerListeningShardSlice(arbitraryPeerID)) != 2 {
		t.Errorf(
			"Peer %v should be listening to %v shards, not %v",
			arbitraryPeerID,
			2,
			len(shardPrefTable.GetPeerListeningShardSlice(arbitraryPeerID)),
		)
	}
	shardPrefTable.RemovePeerListeningShard(arbitraryPeerID, anotherShardID)
	if shardPrefTable.IsPeerListeningShard(arbitraryPeerID, anotherShardID) {
		t.Errorf("Peer %v should be listening to shard %v", arbitraryPeerID, testingShardID)
	}
	if len(shardPrefTable.GetPeerListeningShardSlice(arbitraryPeerID)) != 1 {
		t.Errorf(
			"Peer %v should be only listening to %v shards, not %v",
			arbitraryPeerID,
			1,
			len(shardPrefTable.GetPeerListeningShardSlice(arbitraryPeerID)),
		)
	}

	// see if it is still correct with multiple peers
	anotherPeerID := peer.ID("9547")
	if len(shardPrefTable.GetPeerListeningShardSlice(anotherPeerID)) != 0 {
		t.Errorf("Peer %v should not be listening to any shard", anotherPeerID)
	}
	shardPrefTable.AddPeerListeningShard(anotherPeerID, testingShardID)
	if len(shardPrefTable.GetPeerListeningShardSlice(anotherPeerID)) != 1 {
		t.Errorf("Peer %v should be listening to 1 shard", anotherPeerID)
	}
	// make sure not affect other peers
	if len(shardPrefTable.GetPeerListeningShardSlice(arbitraryPeerID)) != 1 {
		t.Errorf("Peer %v should be listening to 1 shard", arbitraryPeerID)
	}
	shardPrefTable.RemovePeerListeningShard(anotherPeerID, testingShardID)
	if len(shardPrefTable.GetPeerListeningShardSlice(anotherPeerID)) != 0 {
		t.Errorf("Peer %v should be listening to 0 shard", anotherPeerID)
	}
	if len(shardPrefTable.GetPeerListeningShardSlice(arbitraryPeerID)) != 1 {
		t.Errorf("Peer %v should be listening to 1 shard", arbitraryPeerID)
	}
}
