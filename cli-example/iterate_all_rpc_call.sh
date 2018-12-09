#!/bin/bash
# This script is expected to be executed in the root dir of the repo

COMMAND_SCRIPT="$(dirname $0)/common.sh"
. $COMMAND_SCRIPT

go_build

# check version
$EXE_NAME version

for i in `seq 0 1`;
do
    spinup_node $i
done

sleep 2

for i in `seq 0 1`;
do
    identify $i
done

# peer 0 add peer 1
add_peer 0 1

num_peers_to_connect=3

# peer 0 subscribe shard 1 2 3 4 5
subscribe_shard 0 $num_peers_to_connect 1 2 3 4 5

# peer 1 subscribe shard 2 3 with no discover
subscribe_shard_no_discover 1 2 3

# peer 1 subscribe shard 4
subscribe_shard 1 $num_peers_to_connect 4

# peer 0 discover shard peers from shard 3
discover_shard 0 3

# get peer 0's subscribed shard
get_subscribe_shard 0

# peer 0 broadcast collations
broadcast_collation 0 2 2 100 0

# peer 1 broadcast collations
broadcast_collation 1 1 1 100 0
# exit code should be 1
if [ "$?" != "1" ]
then
    exit 1
fi

# peer 0 unsubscribe shard
unsubscribe_shard 0 2 4

get_subscribe_shard 0

list_peer 0

list_topic_peer 0

remove_peer 0 $(show_pid 1)

bootstrap 0 start $(show_multiaddr 1)
bootstrap 0 stop

for i in `seq 0 1`;
do
    stop_server $i
done

sleep 1
