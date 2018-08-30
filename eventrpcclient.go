package main

import (
	"context"
	"fmt"
	"log"

	pbevent "github.com/ethresearch/sharding-p2p-poc/pb/event"
	pbmsg "github.com/ethresearch/sharding-p2p-poc/pb/message"
	pubsub "github.com/libp2p/go-floodsub"
	"google.golang.org/grpc"
)

var (
	eventRPCPort = 15000
	eventRPCAddr = fmt.Sprintf("127.0.0.1:%v", eventRPCPort)
)

type EventNotifier interface {
	NotifyNewCollation(collation *pbmsg.Collation) (bool, error)
}

type rpcEventNotifier struct {
	client pbevent.EventClient
	ctx    context.Context
}

func NewRpcEventNotifier(ctx context.Context, rpcAddr string) (*rpcEventNotifier, error) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
		return nil, err
	}
	client := pbevent.NewEventClient(conn)
	n := &rpcEventNotifier{
		client: client,
		ctx:    ctx,
	}
	return n, nil
}

func (notifier *rpcEventNotifier) TestValidator(ctx context.Context, msg *pubsub.Message) bool {
	return false
}

func (notifier *rpcEventNotifier) NotifyNewCollation(collation *pbmsg.Collation) (bool, error) {
	newCollationNotifier := &pbevent.NewCollationNotifier{
		MetaMsg:   &pbevent.MetaMsg{},
		Collation: collation,
	}
	res, err := notifier.client.NewCollation(notifier.ctx, newCollationNotifier)
	if err != nil {
		return false, err
	}
	return res.IsValid, nil
}

func callEventRPCNewCollation(eventRPCAddr string, collation *pbmsg.Collation) (bool, error) {
	notifier, err := NewRpcEventNotifier(context.Background(), eventRPCAddr)
	if err != nil {
		return false, err
	}
	res, err := notifier.NotifyNewCollation(collation)
	if err != nil {
		return false, err
	}
	return res, nil
}
