import re

import pytest

from simulation.logs import (
    OperationLogs,
    REGEX_LIST,
    RPCLogs,
    map_log_enum_to_content_pattern,
)


@pytest.mark.parametrize(
    "log_enum, log",
    (
        (
            RPCLogs.LOG_ADD_PEER_FMT,
            "rpcserver:AddPeer: ip=192.168.0.15, port=10001, seed=1 rpcserver.go:95",
        ),
        (
            RPCLogs.LOG_ADD_PEER_FINISHED,
            "rpcserver:AddPeer: finished rpcserver.go:126",
        ),
        (
            RPCLogs.LOG_SUBSCRIBE_SHARD_FMT,
            "rpcserver:SubscribeShard: shardIDs=[0] rpcserver.go:183",
        ),
        (
            RPCLogs.LOG_SUBSCRIBE_SHARD_FINISHED,
            "rpcserver:SubscribeShard: finished rpcserver.go:195",
        ),
        (
            RPCLogs.LOG_UNSUBSCRIBE_SHARD_FMT,
            "rpcserver:UnsubscribeShard: shardIDs=[0] rpcserver.go:211",
        ),
        (
            RPCLogs.LOG_UNSUBSCRIBE_SHARD_FINISHED,
            "rpcserver:UnsubscribeShard: finished rpcserver.go:223",
        ),
        (
            RPCLogs.LOG_BROADCAST_COLLATION_FMT,
            "rpcserver:BroadcastCollation: broadcasting: shardID=0, numCollations=2, sizeInBytes=9900, timeInMs=100 rpcserver.go:269",
        ),
        (
            RPCLogs.LOG_BROADCAST_COLLATION_FINISHED,
            "rpcserver:BroadcastCollation: finished rpcserver.go:298",
        ),
        (
            RPCLogs.LOG_DISCOVER_SHARD_FMT,
            "rpcserver:DiscoverShard: Shards=[shardCollations_0] rpcserver.go:151",
        ),
        (
            RPCLogs.LOG_DISCOVER_SHARD_FINISHED,
            "rpcserver:DiscoverShard: finished rpcserver.go:164",
        ),
        (
            RPCLogs.LOG_BOOTSTRAP_FMT,
            "rpcserver:Bootstrap: flag=true, bootnodes=/ip4/192.168.0.15/tcp/10001/ipfs/QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd rpcserver.go:487",
        ),
        (
            RPCLogs.LOG_BOOTSTRAP_FMT,
            "rpcserver:Bootstrap: flag=false, bootnodes= rpcserver.go:487",
        ),
        (
            RPCLogs.LOG_BOOTSTRAP_FINISHED,
            "rpcserver:Bootstrap: finished rpcserver.go:519",
        ),
        (
            RPCLogs.LOG_REMOVE_PEER_FMT,
            "rpcserver:RemovePeer: peerID=QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd rpcserver.go:455",
        ),
        (
            RPCLogs.LOG_REMOVE_PEER_FINISHED,
            "rpcserver:RemovePeer: finished rpcserver.go:473",
        ),
        (
            RPCLogs.LOG_STOP_FMT,
            "rpcserver:StopServer rpcserver.go:347",
        ),
        (
            RPCLogs.LOG_STOP_FINISHED,
            "rpcserver:StopServer: finished rpcserver.go:353",
        ),
        (
            OperationLogs.LOG_RECEIVE_MSG,
            "Validating the received message: topic=shardCollations_0, from=<peer.ID Qm*zMzwwi>, dataSize=9908 shardmanager.go:364",
        ),
    ),
    ids=(
        "addpeer format",
        "addpeer finished",
        "subscribeshard format",
        "subscribeshard finished",
        "unsubscribeshard format",
        "unsubscribeshard finished",
        "broadcastcollation format",
        "broadcastcollation finished",
        "discovershard format",
        "discovershard finished",
        "bootstrap format start",
        "bootstrap format stop",
        "bootstrap finished",
        "removepeer format",
        "removepeer finished",
        "stop format",
        "stop finished",
        "receive message",
    )
)
def test_log_patterns(log_enum, log):
    assert log_enum in map_log_enum_to_content_pattern
    pattern = map_log_enum_to_content_pattern[log_enum]
    match = re.search(pattern, log)
    assert match is not None


@pytest.mark.parametrize(
    'log, parsed',
    (
        (
            "[]",
            "[]",
        ),
        (
            "[0]",
            "[0]",
        ),
        (
            "[0 1 2]",
            '[0 1 2]',
        ),
    ),
)
def test_regex_list_valid(log, parsed):
    match = re.search(REGEX_LIST, log)
    assert match is not None


@pytest.mark.parametrize(
    'log',
    (
        "",
        "[",
        "]",
        "[ ]"
        "[0,1,2]",
        "[0, 1, 2]",
        "[ 0]",
        "[0 ]",
    ),
)
def test_regex_list_invalid(log):
    match = re.search(REGEX_LIST, log)
    assert match is None
