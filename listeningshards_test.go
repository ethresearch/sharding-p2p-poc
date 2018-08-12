package main

import "testing"

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
	// test `ToBytes` and `ListeningShardsFromBytes`
	bytes := ls.ToBytes()
	lsNew := ListeningShardsFromBytes(bytes)
	if len(ls.getShards()) != len(lsNew.getShards()) {
		t.Error()
	}
	for index, value := range ls.getShards() {
		if value != lsNew.getShards()[index] {
			t.Error()
		}
	}
}
