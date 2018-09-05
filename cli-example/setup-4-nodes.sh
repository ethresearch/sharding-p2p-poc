#!/bin/bash
# This script is expected to be executed in the root dir of the repo

EXE_NAME="./sharding-p2p-poc"
IP=127.0.0.1
PORT=10000
RPCPORT=13000
EVENTRPCPORT=35566

# spinup_node {seed} {other_params}
spinup_node() {
    port=$((PORT+$1))
    rpcport=$((RPCPORT+$1))
    eventrpcport=$((EVENTRPCPORT+$1))
    p=$@
    params=${p[@]:1}
    $EXE_NAME -seed=$1 -port=$port -rpcport=$rpcport -notifierport=$eventrpcport $params &
}

cli_prompt() {
    p=$@
    seed=$1
    params=${p[@]:1}
    echo "$EXE_NAME -rpcport=$((RPCPORT+seed)) -client $params"
}

# add_peer {seed0} {seed1}
add_peer() {
    seed0=$1
    seed1=$2
    `cli_prompt $seed0` addpeer $IP $((PORT+seed1)) $seed1
}

make partial-gx-rw
go build

killall $EXE_NAME

for i in `seq 0 2`;
do
    spinup_node $i
done

sleep 2

add_peer 0 1
add_peer 1 2

multiaddr0=/ip4/127.0.0.1/tcp/10000/ipfs/QmS5QmciTXXnCUCyxud5eWFenUMAmvAWSDa1c7dvdXRMZ7
multiaddr1=/ip4/127.0.0.1/tcp/10001/ipfs/QmexAnfpHrhMmAC5UNQVS8iBuUUgDrMbMY17Cck2gKrqeX

spinup_node 3 -bootstrap -bootnodes=$multiaddr0,$multiaddr1

make gx-uw

