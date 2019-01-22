import datetime

import pytest

from simulation.logs import (
    RPCLogs,
    OperationLogs,
)
from simulation.log_aggregation import (
    Event,
    NoMatchingPattern,
    ParsingError,
    parse_line,
)


def test_parse_line_log_format_invalid_time():
    with pytest.raises(ParsingError):
        parse_line(
            'ABC123 03:53:21.346 DEBUG sharding-p: rpcserver:AddPeer: ip=192.168.0.15, port=10001, seed=1 rpcserver.go:95',  # noqa: E501
        )


@pytest.mark.parametrize(
    "line",
    (
        '03:53:21.386 DEBUG sharding-p: rpcserver:AddPeer: finished rpcserver.go:126',
        '2019-01-12T03:53:21.387236300Z DEBUG sharding-p: rpcserver:AddPeer: finished rpcserver.go:126',  # noqa: E501
        '2019-01-12T03:53:21.387236300Z 03:53:21.386 sharding-p: rpcserver:AddPeer: finished rpcserver.go:126',  # noqa: E501
        '2019-01-12T03:53:21.387236300Z 03:53:21.386 DEBUG rpcserver:AddPeer: finished rpcserver.go:126',  # noqa: E501
        '2019-01-12T03:53:21.387236300Z 03:53:21.386 DEBUG sharding-p: 123',
    ),
    ids=(
        "lack of time from docker",
        "lack of time from libp2p logger",
        "lack of log type",
        "lack of logger name",
        "random log content",
    ),
)
def test_parse_line_log_format_invalid_no_matching_pattern(line):
    with pytest.raises(NoMatchingPattern):
        assert parse_line(line)


@pytest.mark.parametrize(
    "line, expected",
    (
        (
            '2019-01-12T03:53:21.348940500Z 03:53:21.346 DEBUG sharding-p: rpcserver:AddPeer: ip=192.168.0.15, port=10001, seed=1 rpcserver.go:95',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 21, 348940, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_ADD_PEER_FMT, params=('192.168.0.15', 10001, 1)),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:21.387236300Z 03:53:21.386 DEBUG sharding-p: rpcserver:AddPeer: finished rpcserver.go:126',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 21, 387236, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_ADD_PEER_FINISHED, params=(),),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:22.619065100Z 03:53:22.614 DEBUG sharding-p: rpcserver:SubscribeShard: shardIDs=[0] rpcserver.go:183',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 22, 619065, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_SUBSCRIBE_SHARD_FMT, params=((0,),),),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:22.649928100Z 03:53:22.649 DEBUG sharding-p: rpcserver:SubscribeShard: finished rpcserver.go:195',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 22, 649928, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_SUBSCRIBE_SHARD_FINISHED, params=()),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:26.232925700Z 03:53:26.232 DEBUG sharding-p: rpcserver:UnsubscribeShard: shardIDs=[0] rpcserver.go:211',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 232925, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_UNSUBSCRIBE_SHARD_FMT, params=((0,),)),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:26.274306300Z 03:53:26.273 DEBUG sharding-p: rpcserver:UnsubscribeShard: finished rpcserver.go:223',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 274306, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_UNSUBSCRIBE_SHARD_FINISHED, params=()),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:24.634516000Z 03:53:24.634 DEBUG sharding-p: rpcserver:BroadcastCollation: broadcasting: shardID=0, numCollations=2, sizeInBytes=9900, timeInMs=100 rpcserver.go:269',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 24, 634516, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_BROADCAST_COLLATION_FMT, params=(0, 2, 9900, 100)),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:24.866050900Z 03:53:24.861 DEBUG sharding-p: rpcserver:BroadcastCollation: finished rpcserver.go:298',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 24, 866050, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_BROADCAST_COLLATION_FINISHED, params=()),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:25.561415000Z 03:53:25.560 DEBUG sharding-p: rpcserver:DiscoverShard: Shards=[shardCollations_0] rpcserver.go:151',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 25, 561415, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_DISCOVER_SHARD_FMT, params=(("shardCollations_0",),)),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:25.562048600Z 03:53:25.561 DEBUG sharding-p: rpcserver:DiscoverShard: finished rpcserver.go:164',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 25, 562048, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_DISCOVER_SHARD_FINISHED, params=()),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:25.794067600Z 03:53:25.787 DEBUG sharding-p: rpcserver:Bootstrap: flag=true, bootnodes=/ip4/192.168.0.15/tcp/10001/ipfs/QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd rpcserver.go:487',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 25, 794067, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_BOOTSTRAP_FMT, params=(True, '/ip4/192.168.0.15/tcp/10001/ipfs/QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd')),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:25.794956800Z 03:53:25.788 DEBUG sharding-p: rpcserver:Bootstrap: finished rpcserver.go:519',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 25, 794956, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_BOOTSTRAP_FINISHED, params=()),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:26.460124600Z 03:53:26.459 DEBUG sharding-p: rpcserver:RemovePeer: peerID=QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd rpcserver.go:455',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 460124, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_REMOVE_PEER_FMT, params=('QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd',)),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:26.470722700Z 03:53:26.469 DEBUG sharding-p: rpcserver:RemovePeer: finished rpcserver.go:473',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 470722, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_REMOVE_PEER_FINISHED, params=()),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:26.679785200Z 03:53:26.678 DEBUG sharding-p: rpcserver:StopServer rpcserver.go:347',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 679785, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_STOP_FMT, params=()),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:26.679827100Z 03:53:26.678 DEBUG sharding-p: rpcserver:StopServer: finished rpcserver.go:353',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 679827, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_STOP_FMT, params=()),  # noqa: E501
        ),
        (
            '2019-01-12T03:53:24.866192400Z 03:53:24.861 DEBUG sharding-p: Validating the received message: topic=shardCollations_0, from=<peer.ID Qm*zMzwwi>, dataSize=9910 shardmanager.go:364',  # noqa: E501
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 24, 866192, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=OperationLogs.LOG_RECEIVE_MSG, params=()),  # noqa: E501
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
        "bootstrap finished",
        "removepeer format",
        "removepeer finished",
        "stop format",
        "stop finished",
        "receive message",
    )
)
def test_parse_line_valid(line, expected):
    assert parse_line(line) == expected
