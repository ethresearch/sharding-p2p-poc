package main

import (
	"context"
	"log"

	pbevent "github.com/ethresearch/sharding-p2p-poc/pb/event"
	pbmsg "github.com/ethresearch/sharding-p2p-poc/pb/message"
	"google.golang.org/grpc"
)

const EVENT_RPC_PORT = 15000

func callEventRPC(eventRPCAddr string, collation *pbmsg.Collation) (bool, error) {
	conn, err := grpc.Dial(eventRPCAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return false, err
	}
	defer conn.Close()
	client := pbevent.NewEventClient(conn)
	newCollationNotifier := &pbevent.NewCollationNotifier{
		MetaMsg:   &pbevent.MetaMsg{},
		Collation: collation,
	}
	log.Printf("eventrpcclient:NewCollation: sending=%v", newCollationNotifier)
	res, err := client.NewCollation(context.Background(), newCollationNotifier)
	if err != nil {
		log.Fatal(err)
		return false, err
	}
	return res.IsValid, nil
}
