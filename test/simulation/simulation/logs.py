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


class EventHasNoParameter(Exception):
    pass


# The regex for the list of elements: empty list, or 1 or more words, delimited by whitespaces
# E.g. [], [1], [1 2], [shardCollations_2]
LIST_DELIMITER = r' '
REGEX_LIST = r'\[(?:|\w+(?:{}\w+)*)\]'.format(LIST_DELIMITER)


def boolean(param_str):
    param_str_lower = param_str.lower()
    if param_str_lower == "true":
        return True
    elif param_str_lower == "false":
        return False
    raise ValueError(
        "Invalid `param_str`, whose `.lower` should be either `true` or `false`, "
        "param_str={}".format(param_str)
    )


def list_ctor(list_str, element_type):
    if len(list_str) < 2:
        raise ValueError("len(list_str) <= 2, list_str={}".format(list_str))
    splitted_list = list_str[1:-1].split(LIST_DELIMITER)
    # filter empty strings
    filtered_list = (
        x
        for x in splitted_list
        if x != ''
    )
    return tuple(
        element_type(i)
        for i in filtered_list
    )


# TODO: better way to handle types? like typing?
_rpc_logs_params_ctor_map = {
    RPCLogs.LOG_ADD_PEER_FMT: (str, int, int),
    RPCLogs.LOG_BROADCAST_COLLATION_FMT: (int, int, int, int),
    RPCLogs.LOG_SUBSCRIBE_SHARD_FMT: (partial(list_ctor, element_type=int),),
    RPCLogs.LOG_UNSUBSCRIBE_SHARD_FMT: (partial(list_ctor, element_type=int),),
    RPCLogs.LOG_DISCOVER_SHARD_FMT: (partial(list_ctor, element_type=str),),
    RPCLogs.LOG_REMOVE_PEER_FMT: (str,),
    RPCLogs.LOG_BOOTSTRAP_FMT: (boolean, str),
}


def parse_event_params(param_strs, event_type):
    """
    Args:
        params_strs: A tuple of strings, derived from the regex.
        event_type: The type of the log event.

    Returns:
        A tuple of parsed parameters with proper types, indicated by `_rpc_logs_params_ctor_map`

    Raises:
        ValueError: Raised when the length of `param_strs` is not correct.
    """
    if event_type not in _rpc_logs_params_ctor_map:
        raise EventHasNoParameter(
            "`event_type` has no parameters: "
            "event_type={}, supported_types={}".format(
                event_type,
                _rpc_logs_params_ctor_map.keys(),
            )
        )
    params_ctors = _rpc_logs_params_ctor_map[event_type]
    if len(param_strs) != len(params_ctors):
        raise ValueError(
            "length of `params_ctors` and `param_strs` should be the same: "
            "params_ctors={}, param_strs={}".format(params_ctors, param_strs)
        )
    return tuple(
        ctor(param_str)
        for ctor, param_str in zip(params_ctors, param_strs)
    )


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
