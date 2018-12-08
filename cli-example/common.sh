#!/bin/bash


EXE_NAME="./sharding-p2p-poc"
IP=127.0.0.1
PORT=10000
RPCPORT=13000

go_build() {
    GIT_COMMIT=$(git rev-list -1 HEAD)
    go build -ldflags "-X main.GitCommit=$GIT_COMMIT"
}

# show_port {seed}
show_port() {
    echo $((PORT+$1))
}

# show_rpcport {seed}
show_rpcport() {
    echo $((RPCPORT+$1))
}

# spinup_node {seed} {other_params}
spinup_node() {
    seed=$1
    port=$(show_port $seed)
    rpcport=$(show_rpcport $seed)
    p=$@
    params=${@:2}
    $EXE_NAME -seed=$seed -port=$port -rpcport=$rpcport -verbose $params &
}

cli_prompt() {
    p=$@
    seed=$1
    params=${@:2}
    echo "$EXE_NAME -rpcport=$(show_rpcport $seed) -client $params"
}

# identify {seed}
identify() {
    seed=$1
    identify=$(`cli_prompt $seed` identify)
    echo $identify
}

# show_pid {seed}
show_pid() {
    seed=$1
    identify=$(identify $seed)
    re="^([A-Za-z0-9]+) "
    if [[ $identify =~ $re ]]; then
        pid=${BASH_REMATCH[1]};
    fi
    echo $pid
}

# show_multiaddr {seed}
show_multiaddr() {
    seed=$1
    port=$(show_port $seed)
    pid=$(show_pid $seed)
    echo "/ip4/127.0.0.1/tcp/$port/ipfs/$pid"
}

# add_peer {seed0} {seed1}
add_peer() {
    seed0=$1
    seed1=$2
    `cli_prompt $seed0` addpeer $IP $(show_port $seed1) $seed1
}

# subscribe_shard {seed} {num_peers} {shard_id} {shard_id} ...
subscribe_shard() {
    p=$@
    seed=$1
    params=${@:2}
    `cli_prompt $seed` subshard $params
}

# subscribe_shard_no_discover {seed} {shard_id} {shard_id} ...
# subscribe_shard_no_discover is basically subscribe_shard
# but with num_peers set to zero
subscribe_shard_no_disdover() {
    p=$@
    seed=$1
    params=${@:2}
    `cli_prompt $seed` subshard 0 $params
}

# unsubscribe_shard {seed} {shard_id} {shard_id} ...
unsubscribe_shard() {
    p=$@
    seed=$1
    params=${@:2}
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
    params=${@:2}
    `cli_prompt $seed` broadcastcollation $params
}

# stop_server {seed}
stop_server() {
    p=$@
    seed=$1
    `cli_prompt $seed` stop
}

# listpeer {seed}
list_peer() {
    p=$@
    seed=$1
    `cli_prompt $seed` listpeer
}


# listtopicpeer {seed} {topic0} {topic1} ...
list_topic_peer() {
    p=$@
    seed=$1
    `cli_prompt $seed` listtopicpeer
}

# remove_peer {seed} peerID
remove_peer() {
    p=$@
    seed=$1
    params=${@:2}
    `cli_prompt $seed` removepeer $params
}

# bootstrap {seed} {start/stop} {bootnodesStr}
bootstrap() {
    p=$@
    seed=$1
    params=${@:2}
    `cli_prompt $seed` bootstrap $params
}
