package main

import (
	"log"

	"golang.org/x/crypto/sha3"

	"github.com/golang/protobuf/proto"
)

func keccak(msg proto.Message) []byte {
	dataInBytes, err := proto.Marshal(msg)
	if err != nil {
		log.Printf("Error occurs when hashing %v", msg)
	}
	h := sha3.NewLegacyKeccak256()
	h.Write(dataInBytes)
	return h.Sum(nil)
}
