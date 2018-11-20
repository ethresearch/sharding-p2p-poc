#!/bin/bash
# This script is expected to be executed in the root dir of the repo

COMMAND_SCRIPT="$(dirname $0)/common.sh"
. $COMMAND_SCRIPT

# make partial-gx-rw
go build

killall $EXE_NAME

for i in `seq 0 2`;
do
    spinup_node $i
done

sleep 2

add_peer 0 1
add_peer 1 2

spinup_node 3 -bootstrap -bootnodes=$(show_multiaddr 0),$(show_multiaddr 1)

# make gx-uw

