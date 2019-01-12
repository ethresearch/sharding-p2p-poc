# LOG_ADD_PEER = 'rpcserver:AddPeer: ip=192.168.11.69, port=10001, seed=1'
from enum import (
    Enum,
    auto,
)

# RPC_LOGS = namedtuple('')
class RPCLogs(Enum):
    LOG_ADD_PEER_FMT = auto()
    LOG_ADD_PEER_FINISHED = auto()
    LOG_BROADCAST_COLLATION_FMT = auto()
    LOG_BROADCAST_COLLATION_FINISHED = auto()
    LOG_SUBSCRIBE_SHARD_FMT = auto()
    LOG_SUBSCRIBE_SHARD_FINISHED = auto()
    LOG_UNSUBSCRIBE_SHARD_FMT = auto()
    LOG_UNSUBSCRIBE_SHARD_FINISHED = auto()
    LOG_DISCOVER_SHARD_FMT = auto()
    LOG_DISCOVER_SHARD_FINISHED = auto()
    LOG_REMOVE_PEER_FMT = auto()
    LOG_REMOVE_PEER_FINISHED = auto()
    LOG_STOP_FMT = auto()
    LOG_STOP_FINISHED = auto()
    LOG_BOOTSTRAP_FMT = auto()
    LOG_BOOTSTRAP_FINISHED = auto()


class OperationLogs(Enum):
    LOG_RECEIVE_MSG = auto()



LIST_REGEX = r'\[[^\]]*\]'


_rpc_logs_map = {
    RPCLogs.LOG_ADD_PEER_FMT: r'rpcserver:AddPeer: ip=((?:\d{1,3}\.){3}\d{1,3}), port=(\d+), seed=(\d+)',  # noqa: E501
    # FIXME: possibly let all `finished` to be one kind of log?
    RPCLogs.LOG_ADD_PEER_FINISHED: r'rpcserver:AddPeer: finished',
# LOG_BROADCAST_COLLATION = 'rpcserver:BroadcastCollation: broadcasting: shardID=0, numCollations=1, sizeInBytes=900, timeInMs=123'  # noqa: E501
    RPCLogs.LOG_BROADCAST_COLLATION_FMT: r'rpcserver:BroadcastCollation: broadcasting: shardID=(\d+), numCollations=(\d+), sizeInBytes=(\d+), timeInMs=(\d+)',  # noqa: E501
    RPCLogs.LOG_BROADCAST_COLLATION_FINISHED: r'rpcserver:BroadcastCollation: finished',
    # FIXME: should change the list from [0 1 2] to JSON [0, 1, 2], for a more precise parsing
# LOG_SUBSCRIBE_SHARD = 'rpcserver:SubscribeShard: shardIDs=[0 1 2]'
    RPCLogs.LOG_SUBSCRIBE_SHARD_FMT: r'rpcserver:SubscribeShard: shardIDs=({})'.format(LIST_REGEX),
    RPCLogs.LOG_SUBSCRIBE_SHARD_FINISHED: r'rpcserver:SubscribeShard: finished',
# LOG_UNSUBSCRIBE_SHARD = 'rpcserver:UnsubscribeShard: shardIDs=[0]'
    RPCLogs.LOG_UNSUBSCRIBE_SHARD_FMT: r'rpcserver:UnsubscribeShard: shardIDs=({})'.format(LIST_REGEX),
    RPCLogs.LOG_UNSUBSCRIBE_SHARD_FINISHED: r'rpcserver:UnsubscribeShard: finished',
    RPCLogs.LOG_DISCOVER_SHARD_FMT: r'rpcserver:DiscoverShard: Shards=({})'.format(LIST_REGEX),
    RPCLogs.LOG_DISCOVER_SHARD_FINISHED: r'rpcserver:DiscoverShard: finished',
    RPCLogs.LOG_REMOVE_PEER_FMT: r'rpcserver:RemovePeer: peerID=(\w+)',
    RPCLogs.LOG_REMOVE_PEER_FINISHED: r'rpcserver:RemovePeer: finished',
    RPCLogs.LOG_STOP_FMT: r'rpcserver:StopServer',
    RPCLogs.LOG_STOP_FINISHED: r'rpcserver:StopServer: finished',
    RPCLogs.LOG_BOOTSTRAP_FMT: r'rpcserver:Bootstrap: flag=(\w+), bootnodes=(\S*)',
    RPCLogs.LOG_BOOTSTRAP_FINISHED: r'rpcserver:Bootstrap: finished',
}


_operation_logs_map = {
    OperationLogs.LOG_RECEIVE_MSG: 'Validating the received message',
}


_log_enum_to_content_pattern = {
    **_rpc_logs_map,
    **_operation_logs_map,
}




# {docker_time} {time} {log_type} {logger_name} {log_content}
# _example_log = "2019-01-12T03:53:20.874775100Z 03:53:20.874 DEBUG sharding-p: rpcserver:IdentifyRequest: receive= rpcserver.go:70"
# _example_add_peer_log = "03:57:49.742 DEBUG sharding-p: rpcserver:AddPeer: ip=192.168.0.15, port=10001, seed=1 rpcserver.go:95"
# identify = "rpcserver:Identify: finished"
LOG_PATTERN = r"^([\w:\-\.]+)\s+[\d:\.]+\s+(\w+)\s+([^:]+):\s+{}"


map_log_enum_pattern = {
    log_enum: LOG_PATTERN.format(log_content_pattern)
    for log_enum, log_content_pattern in _log_enum_to_content_pattern.items()
}
