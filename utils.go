package main

import (
	"golang.org/x/crypto/sha3"

	"github.com/golang/protobuf/proto"
)

func keccak(msg proto.Message) []byte {
	dataInBytes, err := proto.Marshal(msg)
	if err != nil {
		logger.Panicf("Failed to encode protobuf message: %v, err: %v", msg, err)
	}
	h := sha3.NewLegacyKeccak256()
	h.Write(dataInBytes)
	return h.Sum(nil)
}
