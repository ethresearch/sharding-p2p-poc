package main

import (
	"fmt"
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
	// log.Printf("shardID=%v, byteIndex=%v, bitIndex=%v", shardID, byteIndex, bitIndex)
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
	// log.Printf("shardID=%v, byteIndex=%v, bitIndex=%v", shardID, byteIndex, bitIndex)
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

func ListeningShardsFromSlice(shards []ShardIDType) *ListeningShards {
	listeningShards := NewListeningShards()
	for _, shardID := range shards {
		listeningShards.setShard(shardID)
	}
	return listeningShards
}

// TODO: need checks against the format of bytes
func ListeningShardsFromBytes(bytes []byte) *ListeningShards {
	listeningShards := NewListeningShards()
	listeningShards.shardBits = bytes
	return listeningShards
}

func (ls *ListeningShards) ToBytes() []byte {
	return ls.shardBits
}
