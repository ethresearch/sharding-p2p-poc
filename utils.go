package main

import (
	"golang.org/x/crypto/sha3"

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
