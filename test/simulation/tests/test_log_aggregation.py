import datetime

import pytest

from simulation.logs import (
    RPCLogs,
    OperationLogs,
)
from simulation.log_aggregation import (
    Event,
    parse_line,
    parse_logs,
)


logs_with_rpc_events = """
2019-01-12T03:53:18.561866700Z 03:53:18.561  INFO sharding-p: Node is listening: seed=0, addr=0.0.0.0:10000, peerID=Qmd8vXqRiFVkcXcB1nigjCDuUyHcFkja6fQcU9w2zMzwwi main.go:213
2019-01-12T03:53:18.562849400Z 03:53:18.562  INFO sharding-p: RPC server listening to address: 127.0.0.1:13000 rpcserver.go:552
2019-01-12T03:53:20.874775100Z 03:53:20.874 DEBUG sharding-p: rpcserver:IdentifyRequest: receive= rpcserver.go:70
2019-01-12T03:53:20.875025800Z 03:53:20.874 DEBUG sharding-p: rpcserver:Identify: finished rpcserver.go:80
2019-01-12T03:53:21.348940500Z 03:53:21.346 DEBUG sharding-p: rpcserver:AddPeer: ip=192.168.0.15, port=10001, seed=1 rpcserver.go:95
2019-01-12T03:53:21.387236300Z 03:53:21.386 DEBUG sharding-p: rpcserver:AddPeer: finished rpcserver.go:126
2019-01-12T03:53:22.619065100Z 03:53:22.614 DEBUG sharding-p: rpcserver:SubscribeShard: shardIDs=[0] rpcserver.go:183
2019-01-12T03:53:22.649928100Z 03:53:22.649 DEBUG sharding-p: rpcserver:SubscribeShard: finished rpcserver.go:195
2019-01-12T03:53:24.634516000Z 03:53:24.634 DEBUG sharding-p: rpcserver:BroadcastCollation: broadcasting: shardID=0, numCollations=2, sizeInBytes=9900, timeInMs=100 rpcserver.go:269
2019-01-12T03:53:24.749219300Z 03:53:24.746 DEBUG sharding-p: Validating the received message: topic=shardCollations_0, from=<peer.ID Qm*zMzwwi>, dataSize=9908 shardmanager.go:364
2019-01-12T03:53:24.866050900Z 03:53:24.861 DEBUG sharding-p: rpcserver:BroadcastCollation: finished rpcserver.go:298
2019-01-12T03:53:24.866192400Z 03:53:24.861 DEBUG sharding-p: Validating the received message: topic=shardCollations_0, from=<peer.ID Qm*zMzwwi>, dataSize=9910 shardmanager.go:364
2019-01-12T03:53:25.561415000Z 03:53:25.560 DEBUG sharding-p: rpcserver:DiscoverShard: Shards=[shardCollations_0] rpcserver.go:151
2019-01-12T03:53:25.562048600Z 03:53:25.561 DEBUG sharding-p: rpcserver:DiscoverShard: finished rpcserver.go:164
2019-01-12T03:53:25.794067600Z 03:53:25.787 DEBUG sharding-p: rpcserver:Bootstrap: flag=true, bootnodes=/ip4/192.168.0.15/tcp/10001/ipfs/QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd rpcserver.go:487
2019-01-12T03:53:25.794517600Z 03:53:25.787 DEBUG sharding-p: <peer.ID Qm*zMzwwi> bootstrapping to <peer.ID Qm*57Yrmd> bootstrap.go:46
2019-01-12T03:53:25.794822300Z 03:53:25.787 DEBUG sharding-p: context.Background bootstrapDialSuccess <peer.ID Qm*57Yrmd> bootstrap.go:54
2019-01-12T03:53:25.794860400Z 03:53:25.788 DEBUG sharding-p: Bootstrapped with <peer.ID Qm*57Yrmd> bootstrap.go:55
2019-01-12T03:53:25.794887600Z 03:53:25.788 DEBUG sharding-p: context.Background bootstrapDial <peer.ID Qm*zMzwwi> <peer.ID Qm*57Yrmd> bootstrap.go:56
2019-01-12T03:53:25.794921500Z 03:53:25.788 DEBUG sharding-p: rpcserver:Bootstrap: started bootstrapping rpcserver.go:508
2019-01-12T03:53:25.794956800Z 03:53:25.788 DEBUG sharding-p: rpcserver:Bootstrap: finished rpcserver.go:519
2019-01-12T03:53:26.014750200Z 03:53:26.014 DEBUG sharding-p: rpcserver:Bootstrap: flag=false, bootnodes= rpcserver.go:487
2019-01-12T03:53:26.014787100Z 03:53:26.014 DEBUG sharding-p: rpcserver:Bootstrap: stopped bootstrapping rpcserver.go:517
2019-01-12T03:53:26.014812000Z 03:53:26.014 DEBUG sharding-p: rpcserver:Bootstrap: finished rpcserver.go:519
2019-01-12T03:53:26.232925700Z 03:53:26.232 DEBUG sharding-p: rpcserver:UnsubscribeShard: shardIDs=[0] rpcserver.go:211
2019-01-12T03:53:26.274306300Z 03:53:26.273 DEBUG sharding-p: rpcserver:UnsubscribeShard: finished rpcserver.go:223
2019-01-12T03:53:26.460124600Z 03:53:26.459 DEBUG sharding-p: rpcserver:RemovePeer: peerID=QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd rpcserver.go:455
2019-01-12T03:53:26.470722700Z 03:53:26.469 DEBUG sharding-p: rpcserver:RemovePeer: finished rpcserver.go:473
2019-01-12T03:53:26.679785200Z 03:53:26.678 DEBUG sharding-p: rpcserver:StopServer rpcserver.go:347
2019-01-12T03:53:26.679827100Z 03:53:26.678 DEBUG sharding-p: rpcserver:StopServer: finished rpcserver.go:353
2019-01-12T03:53:27.688521600Z 03:53:27.687  INFO sharding-p: Closing RPC server by rpc call... rpcserver.go:350
"""






@pytest.mark.parametrize(
    "line, expected",
    (
        (
            '2019-01-12T03:53:21.387236300Z 03:53:21.386 DEBUG sharding-p: rpcserver:AddPeer: finished rpcserver.go:126',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 21, 387236, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_ADD_PEER_FINISHED, params=()),
        ),
    ),
    ids=(
        "existing log",
    ),
)
def test_parse_line_log_format_valid(line, expected):
    assert parse_line(line) == expected


@pytest.mark.parametrize(
    "line",
    (
        '03:53:21.386 DEBUG sharding-p: rpcserver:AddPeer: finished rpcserver.go:126',
        '2019-01-12T03:53:21.387236300Z DEBUG sharding-p: rpcserver:AddPeer: finished rpcserver.go:126',
        '2019-01-12T03:53:21.387236300Z 03:53:21.386 sharding-p: rpcserver:AddPeer: finished rpcserver.go:126',
        '2019-01-12T03:53:21.387236300Z 03:53:21.386 DEBUG rpcserver:AddPeer: finished rpcserver.go:126',
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
def test_parse_line_log_format_invalid(line):
    assert parse_line(line) is None


@pytest.mark.parametrize(
    "line, expected",
    (
        (
            '2019-01-12T03:53:21.348940500Z 03:53:21.346 DEBUG sharding-p: rpcserver:AddPeer: ip=192.168.0.15, port=10001, seed=1 rpcserver.go:95',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 21, 348940, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_ADD_PEER_FMT, params=('192.168.0.15', '10001', '1')),
        ),
        (
            '2019-01-12T03:53:21.387236300Z 03:53:21.386 DEBUG sharding-p: rpcserver:AddPeer: finished rpcserver.go:126',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 21, 387236, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_ADD_PEER_FINISHED, params=()),
        ),
        (
            '2019-01-12T03:53:22.619065100Z 03:53:22.614 DEBUG sharding-p: rpcserver:SubscribeShard: shardIDs=[0] rpcserver.go:183',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 22, 619065, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_SUBSCRIBE_SHARD_FMT, params=('[0]',)),
        ),
        (
            '2019-01-12T03:53:22.649928100Z 03:53:22.649 DEBUG sharding-p: rpcserver:SubscribeShard: finished rpcserver.go:195',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 22, 649928, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_SUBSCRIBE_SHARD_FINISHED, params=()),
        ),
        (
            '2019-01-12T03:53:26.232925700Z 03:53:26.232 DEBUG sharding-p: rpcserver:UnsubscribeShard: shardIDs=[0] rpcserver.go:211',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 232925, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_UNSUBSCRIBE_SHARD_FMT, params=('[0]',)),
        ),
        (
            '2019-01-12T03:53:26.274306300Z 03:53:26.273 DEBUG sharding-p: rpcserver:UnsubscribeShard: finished rpcserver.go:223',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 274306, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_UNSUBSCRIBE_SHARD_FINISHED, params=()),
        ),
        (
            '2019-01-12T03:53:24.634516000Z 03:53:24.634 DEBUG sharding-p: rpcserver:BroadcastCollation: broadcasting: shardID=0, numCollations=2, sizeInBytes=9900, timeInMs=100 rpcserver.go:269',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 24, 634516, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_BROADCAST_COLLATION_FMT, params=('0', '2', '9900', '100')),
        ),
        (
            '2019-01-12T03:53:24.866050900Z 03:53:24.861 DEBUG sharding-p: rpcserver:BroadcastCollation: finished rpcserver.go:298',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 24, 866050, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_BROADCAST_COLLATION_FINISHED, params=()),
        ),
        (
            '2019-01-12T03:53:25.561415000Z 03:53:25.560 DEBUG sharding-p: rpcserver:DiscoverShard: Shards=[shardCollations_0] rpcserver.go:151',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 25, 561415, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_DISCOVER_SHARD_FMT, params=('[shardCollations_0]',)),
        ),
        (
            '2019-01-12T03:53:25.562048600Z 03:53:25.561 DEBUG sharding-p: rpcserver:DiscoverShard: finished rpcserver.go:164',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 25, 562048, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_DISCOVER_SHARD_FINISHED, params=()),
        ),
        (
            '2019-01-12T03:53:25.794067600Z 03:53:25.787 DEBUG sharding-p: rpcserver:Bootstrap: flag=true, bootnodes=/ip4/192.168.0.15/tcp/10001/ipfs/QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd rpcserver.go:487',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 25, 794067, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_BOOTSTRAP_FMT, params=('true', '/ip4/192.168.0.15/tcp/10001/ipfs/QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd')),
        ),
        (
            '2019-01-12T03:53:25.794956800Z 03:53:25.788 DEBUG sharding-p: rpcserver:Bootstrap: finished rpcserver.go:519',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 25, 794956, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_BOOTSTRAP_FINISHED, params=()),
        ),
        (
            '2019-01-12T03:53:26.460124600Z 03:53:26.459 DEBUG sharding-p: rpcserver:RemovePeer: peerID=QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd rpcserver.go:455',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 460124, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_REMOVE_PEER_FMT, params=('QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd',)),
        ),
        (
            '2019-01-12T03:53:26.470722700Z 03:53:26.469 DEBUG sharding-p: rpcserver:RemovePeer: finished rpcserver.go:473',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 470722, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_REMOVE_PEER_FINISHED, params=()),
        ),
        (
            '2019-01-12T03:53:26.679785200Z 03:53:26.678 DEBUG sharding-p: rpcserver:StopServer rpcserver.go:347',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 679785, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_STOP_FMT, params=()),
        ),
        (
            '2019-01-12T03:53:26.679827100Z 03:53:26.678 DEBUG sharding-p: rpcserver:StopServer: finished rpcserver.go:353',
            Event(time=datetime.datetime(2019, 1, 12, 3, 53, 26, 679827, tzinfo=datetime.timezone.utc), log_type='DEBUG', logger_name='sharding-p', event_type=RPCLogs.LOG_STOP_FMT, params=()),
        ),
    ),
)
def test_parse_line_success(line, expected):
    assert parse_line(line) == expected
    print(parse_line(line))


def test_over_all():
    # event = parse_line(example_add_peer_log)
    for line in logs_with_rpc_events.split('\n'):
        parsed = parse_line(line)
        if parsed is not None:
            print("line={!r}, event={!r}".format(line, parsed))

