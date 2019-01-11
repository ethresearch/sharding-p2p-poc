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


class OperationLogs(Enum):
    LOG_RECEIVE_MSG = auto()


_rpc_logs_map = {
    RPCLogs.LOG_ADD_PEER_FMT: 'rpcserver:AddPeer: ip={}, port={}, seed={}',
    RPCLogs.LOG_ADD_PEER_FINISHED: 'rpcserver:AddPeer: finished',
# LOG_BROADCAST_COLLATION = 'rpcserver:BroadcastCollation: broadcasting: shardID=0, numCollations=1, sizeInBytes=900, timeInMs=123'  # noqa: E501
    RPCLogs.LOG_BROADCAST_COLLATION_FMT: 'rpcserver:BroadcastCollation: broadcasting: shardID={}, numCollations={}, sizeInBytes={}, timeInMs={}',  # noqa: E501
    RPCLogs.LOG_BROADCAST_COLLATION_FINISHED: 'rpcserver:BroadcastCollation: finished',
# LOG_SUBSCRIBE_SHARD = 'rpcserver:SubscribeShard: shardIDs=[0 1 2]'
    RPCLogs.LOG_SUBSCRIBE_SHARD_FMT: 'rpcserver:SubscribeShard: shardIDs={}',
    RPCLogs.LOG_SUBSCRIBE_SHARD_FINISHED: 'rpcserver:SubscribeShard: finished',
# LOG_UNSUBSCRIBE_SHARD = 'rpcserver:UnsubscribeShard: shardIDs=[0]'
    RPCLogs.LOG_UNSUBSCRIBE_SHARD_FMT: 'rpcserver:UnsubscribeShard: shardIDs={}',
    RPCLogs.LOG_UNSUBSCRIBE_SHARD_FINISHED: 'rpcserver:UnsubscribeShard: finished',
}


_operation_logs_map = {
    OperationLogs.LOG_RECEIVE_MSG: 'Validating the received message',
}


logs_map = {
    **_rpc_logs_map,
    **_operation_logs_map,
}

str_to_logs = {
    key: value
    for key, value in logs_map.items()
}

