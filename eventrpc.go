package main

import (
	"context"
	"fmt"
	"time"

	pbevent "github.com/ethresearch/sharding-p2p-poc/pb/event"
	peer "github.com/libp2p/go-libp2p-peer"
	"google.golang.org/grpc"
)

type EventNotifier interface {
	Receive(ctx context.Context, peerID peer.ID, msgType int, data []byte) ([]byte, error)
}

type mockEventNotifier struct {
}

type rpcEventNotifier struct {
	client pbevent.EventClient
}

func NewMockEventNotifier() *mockEventNotifier {
	return &mockEventNotifier{}
}

func (notifier *mockEventNotifier) Receive(
	ctx context.Context,
	peerID peer.ID,
	msgType int,
	data []byte) ([]byte, error) {
	// Always return 1
	return []byte{1}, nil
}

func NewRpcEventNotifier(ctx context.Context, rpcAddr string) (*rpcEventNotifier, error) {
	conn, err := grpc.Dial(rpcAddr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Second*1))
	if err != nil {
		logger.Errorf("Failed to connect to the event notifier rpc server: %v", err)
		return nil, err
	}
	client := pbevent.NewEventClient(conn)
	n := &rpcEventNotifier{
		client: client,
	}
	return n, nil
}

func (notifier *rpcEventNotifier) Receive(
	ctx context.Context,
	peerID peer.ID,
	msgType int,
	data []byte) ([]byte, error) {
	req := &pbevent.ReceiveRequest{
		PeerID:  peerID.Pretty(),
		MsgType: PBInt(msgType),
		Data:    data,
	}
	res, err := notifier.client.Receive(ctx, req)
	if err != nil {
		return nil, err
	}
	if res.Response.Status != pbevent.Response_SUCCESS {
		return nil, fmt.Errorf("failure response")
	}
	return res.Data, nil
}
