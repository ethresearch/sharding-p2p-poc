package main

import (
	"context"
	"fmt"
	"log"

	pbevent "github.com/ethresearch/sharding-p2p-poc/pb/event"
	pbmsg "github.com/ethresearch/sharding-p2p-poc/pb/message"
	"google.golang.org/grpc"
)

var (
	defulatEventRPCPort = 35566
)

type EventNotifier interface {
	NotifyCollation(collation *pbmsg.Collation) (bool, error)
	NotifyPubSub(string, []byte) (bool, error)
	GetCollation(shardID ShardIDType, period int, hash string) (*pbmsg.Collation, error)
}

type rpcEventNotifier struct {
	client pbevent.EventClient
	ctx    context.Context
}

func NewRpcEventNotifier(ctx context.Context, rpcAddr string) (*rpcEventNotifier, error) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Printf("failed to connect to the rpc server: %v", err)
		return nil, err
	}
	client := pbevent.NewEventClient(conn)
	n := &rpcEventNotifier{
		client: client,
		ctx:    ctx,
	}
	return n, nil
}

func (notifier *rpcEventNotifier) NotifyCollation(collation *pbmsg.Collation) (bool, error) {
	notifyCollationReq := &pbevent.NotifyCollationRequest{
		MetaMsg:   &pbevent.MetaMsg{},
		Collation: collation,
	}
	res, err := notifier.client.NotifyCollation(notifier.ctx, notifyCollationReq)
	if err != nil {
		return false, err
	}
	return res.IsValid, nil
}

func (notifier *rpcEventNotifier) NotifyPubSub(topic string, data []byte) (bool, error) {
	notifyPubSubReq := &pbevent.NotifyPubSubRequest{
		Topic: topic,
		Data: data,
	}
	res, err := notifier.client.NotifyPubSub(notifier.ctx, notifyPubSubReq)
	if err != nil {
		return false, err
	}
	return res.IsValid, nil
}

func (notifier *rpcEventNotifier) GetCollation(
	shardID ShardIDType,
	period int,
	hash string) (*pbmsg.Collation, error) {
	getCollationReq := &pbevent.GetCollationRequest{
		MetaMsg: &pbevent.MetaMsg{},
		ShardID: PBInt(shardID),
		Period:  PBInt(period),
		Hash:    hash,
	}
	res, err := notifier.client.GetCollation(notifier.ctx, getCollationReq)
	if err != nil {
		return nil, err
	}
	if res.Response.Status != pbevent.Response_SUCCESS {
		return nil, fmt.Errorf("request failed: %v", res.Response.Message)
	}
	return res.Collation, nil
}
