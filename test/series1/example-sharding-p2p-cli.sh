#!/bin/bash
# This script is expected to be executed in the root dir of the repo

EXE_NAME="./sharding-p2p-poc"
IP=127.0.0.1
PORT=10000
RPCPORT=13000

# spinup_node {seed} {other_params}
spinup_node() {
    port=$((PORT+$1))
    rpcport=$((RPCPORT+$1))
    p=$@
    params=${p[@]:1}
    $EXE_NAME -seed=$1 -port=$port -rpcport=$rpcport $params &
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

# subscribe_shard {seed} {shard_id0} {shard_id1} ...
subscribe_shard() {
    p=$@
    seed=$1
    params=${p[@]:1}
    `cli_prompt $seed` subshard $params
}

# unsubscribe_shard {seed} {shard_id0} {shard_id1} ...
unsubscribe_shard() {
    p=$@
    seed=$1
    params=${p[@]:1}
    # subshard {shard_id0} {shard_id1} ...
    `cli_prompt $seed` unsubshard $params
}

# get_subscribed_shard {seed}
get_subscribed_shard() {
    p=$@
    seed=$1
    `cli_prompt $seed` getsubshard
}

# broadcast_collation {seed} {shard_id} {num_collations} {collation_size} {time_in_ms}
broadcast_collation() {
    p=$@
    seed=$1
    params=${p[@]:1}
    `cli_prompt $seed` broadcastcollation $params
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

subscribe_shard 0 42 56  # node 0 subscribe shards 42, 56
subscribe_shard 1 42 56  # node 1 subscribe shards 42, 56
unsubscribe_shard 1 56  # node 1 unsubscribe shards 56
get_subscribed_shard 1  # list node 1's subscribed shards

# node 1 broadcasts 123 5566-byte-sized collations in shard 42 every 789 ms
broadcast_collation 1 42 123 5566 789

make gx-uw