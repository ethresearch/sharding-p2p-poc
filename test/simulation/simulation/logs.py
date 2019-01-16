from functools import (
    partial,
)
from enum import (
    Enum,
    auto,
)


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


# The regex for the list of elements: empty list, or 1 or more words, delimited by whitespaces
# E.g. [], [1], [1 2], [shardCollations_2]
LIST_DELIMITER = r' '
REGEX_LIST = r'\[(?:|\w+(?:{}\w+)*)\]'.format(LIST_DELIMITER)


def list_ctor(list_str, element_type):
    list_content = list_str[1:-1]
    return tuple(
        element_type(i)
        for i in list_content.split(LIST_DELIMITER)
    )


# TODO: typing!!!
_rpc_logs_params_ctor_map = {
    RPCLogs.LOG_ADD_PEER_FMT: (str, int, int),
    RPCLogs.LOG_BROADCAST_COLLATION_FMT: (int, int, int, int),
    RPCLogs.LOG_SUBSCRIBE_SHARD_FMT: (partial(list_ctor, element_type=int),),
    RPCLogs.LOG_UNSUBSCRIBE_SHARD_FMT: (partial(list_ctor, element_type=int),),
    RPCLogs.LOG_DISCOVER_SHARD_FMT: (partial(list_ctor, element_type=str),),
    RPCLogs.LOG_REMOVE_PEER_FMT: (str,),
    RPCLogs.LOG_BOOTSTRAP_FMT: (bool, str),
}


def convert_type(param_strs, event_type):
    if event_type not in _rpc_logs_params_ctor_map:
        # do nothing
        return param_strs
    params_ctors = _rpc_logs_params_ctor_map[event_type]
    if len(param_strs) != len(params_ctors):
        raise ValueError(
            "length of `params_ctors` and `param_strs` should be the same: "
            "params_ctors={}, param_strs={}".format(params_ctors, param_strs)
        )
    return tuple(
        params_ctors[index](param_str)
        for index, param_str in enumerate(param_strs)
    )


a = ('1', '2', '3')
b = convert_type(a, RPCLogs.LOG_ADD_PEER_FMT)
print(b)
c = ("[1 3 4]",)
d = convert_type(c, RPCLogs.LOG_SUBSCRIBE_SHARD_FMT)
print(d)



_rpc_logs_map = {
    RPCLogs.LOG_ADD_PEER_FMT: r'rpcserver:AddPeer: ip=((?:[0-9]{1,3}\.){3}[0-9]{1,3}), port=([0-9]+), seed=([0-9]+)',  # noqa: E501
    RPCLogs.LOG_ADD_PEER_FINISHED: r'rpcserver:AddPeer: finished',
    RPCLogs.LOG_BROADCAST_COLLATION_FMT: r'rpcserver:BroadcastCollation: broadcasting: shardID=([0-9]+), numCollations=([0-9]+), sizeInBytes=([0-9]+), timeInMs=([0-9]+)',  # noqa: E501
    RPCLogs.LOG_BROADCAST_COLLATION_FINISHED: r'rpcserver:BroadcastCollation: finished',
    RPCLogs.LOG_SUBSCRIBE_SHARD_FMT: r'rpcserver:SubscribeShard: shardIDs=({})'.format(REGEX_LIST),
    RPCLogs.LOG_SUBSCRIBE_SHARD_FINISHED: r'rpcserver:SubscribeShard: finished',
    RPCLogs.LOG_UNSUBSCRIBE_SHARD_FMT: r'rpcserver:UnsubscribeShard: shardIDs=({})'.format(REGEX_LIST),  # noqa: E501
    RPCLogs.LOG_UNSUBSCRIBE_SHARD_FINISHED: r'rpcserver:UnsubscribeShard: finished',
    RPCLogs.LOG_DISCOVER_SHARD_FMT: r'rpcserver:DiscoverShard: Shards=({})'.format(REGEX_LIST),
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


map_log_enum_to_content_pattern = {
    **_rpc_logs_map,
    **_operation_logs_map,
}




# {docker_time} {time} {log_type} {logger_name} {log_content}
# _example_log = "2019-01-12T03:53:20.874775100Z 03:53:20.874 DEBUG sharding-p: rpcserver:IdentifyRequest: receive= rpcserver.go:70"
# _example_add_peer_log = "03:57:49.742 DEBUG sharding-p: rpcserver:AddPeer: ip=192.168.0.15, port=10001, seed=1 rpcserver.go:95"
LOG_PATTERN = r"^([A-Z0-9:\.\-]+) +[0-9:\.]+ +(\w+) +([^:]+): +{}"


map_log_enum_pattern = {
    log_enum: LOG_PATTERN.format(log_content_pattern)
    for log_enum, log_content_pattern in map_log_enum_to_content_pattern.items()
}
