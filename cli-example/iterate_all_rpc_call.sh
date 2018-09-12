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

# subscribe_shard {seed} {shard_id} {shard_id} ...
subscribe_shard() {
    p=$@
    seed=$1
    params=${p[@]:1}
    `cli_prompt $seed` subshard $params
}

# unsubscribe_shard {seed} {shard_id} {shard_id} ...
unsubscribe_shard() {
    p=$@
    seed=$1
    params=${p[@]:1}
    `cli_prompt $seed` unsubshard $params
}

# get_subscribe_shard {seed}
get_subscribe_shard() {
    p=$@
    seed=$1
    `cli_prompt $seed` getsubshard
}

# broadcast_collation {seed} {shard_id} {num_collation} {size} {period}
broadcast_collation() {
    p=$@
    seed=$1
    params=${p[@]:1}
    `cli_prompt $seed` broadcastcollation $params
}

# stop_server {seed}
stop_server() {
    p=$@
    seed=$1
    `cli_prompt $seed` stop
}

make partial-gx-rw
go build

for i in `seq 0 1`;
do
    spinup_node $i
done

sleep 2

# peer 0 add peer 1
add_peer 0 1

# peer 0 subscribe shard
subscribe_shard 0 1 2 3 4 5

# peer 1 subscribe shard
subscribe_shard 1 2 3 4

# get peer 0's subscribed shard
get_subscribe_shard 0

# peer 0 broadcast collations
broadcast_collation 0 2 2 100 0

# peer 1 broadcast collations
broadcast_collation 1 1 1 100 0

# peer 0 unsubscribe shard
unsubscribe_shard 0 2 4

get_subscribe_shard 0

for i in `seq 0 1`;
do
    stop_server $i
done

make gx-uw
