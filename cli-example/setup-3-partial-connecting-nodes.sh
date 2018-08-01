#!/bin/bash

gx-go rw
go build

killall sharding-p2p-poc

PORT=10000
RPCPORT=13000
SEED=0
./sharding-p2p-poc -seed=$SEED -port=$((PORT+SEED)) -rpcport=$((RPCPORT+SEED)) &
SEED=1
./sharding-p2p-poc -seed=$SEED -port=$((PORT+SEED)) -rpcport=$((RPCPORT+SEED)) &
SEED=2
./sharding-p2p-poc -seed=$SEED -port=$((PORT+SEED)) -rpcport=$((RPCPORT+SEED)) &
sleep 2
./sharding-p2p-poc -rpcport=$((RPCPORT+1)) -client addpeer 127.0.0.1 $((PORT)) 0
./sharding-p2p-poc -rpcport=$((RPCPORT+1)) -client addpeer 127.0.0.1 $((PORT+2)) 2

gx-go uw
