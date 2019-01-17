import functools
import re

import pytest

from simulation.logs import (
    EventHasNoParameter,
    OperationLogs,
    REGEX_LIST,
    RPCLogs,
    boolean,
    list_ctor,
    map_log_enum_to_content_pattern,
    parse_event_params,
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
            "rpcserver:BroadcastCollation: broadcasting: shardID=0, numCollations=2, sizeInBytes=9900, timeInMs=100 rpcserver.go:269",  # noqa: E501
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
            "rpcserver:Bootstrap: flag=true, bootnodes=/ip4/192.168.0.15/tcp/10001/ipfs/QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd rpcserver.go:487",  # noqa: E501
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
            "rpcserver:RemovePeer: peerID=QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd rpcserver.go:455",  # noqa: E501
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
            "Validating the received message: topic=shardCollations_0, from=<peer.ID Qm*zMzwwi>, dataSize=9908 shardmanager.go:364",  # noqa: E501
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
    ),
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


@pytest.mark.parametrize(
    'list_str, element_type, expected',
    (
        ("[]", str, ()),
        ("[abc]", str, ('abc',)),
        ("[abc cba]", str, ('abc', 'cba')),
        ("[]", int, ()),
        ("[123]", int, (123,)),
        ("[123 321]", int, (123, 321)),
        ("[123  321 ]", int, (123, 321)),
    ),
)
def test_list_ctor_success(list_str, element_type, expected):
    assert list_ctor(list_str, element_type) == expected


@pytest.mark.parametrize(
    'list_str, element_type',
    (
        ("", str),
        ("", int),
        ("[", str),
        ("]", str),
        ("[abc]", int),
        ("[abc cba]", int),
    ),
)
def test_list_ctor_failure(list_str, element_type):
    with pytest.raises(ValueError):
        list_ctor(list_str, element_type)


@pytest.mark.parametrize(
    'input, expected',
    (
        ("true", True),
        ("True", True),
        ("false", False),
        ("False", False),
        ("FALse", False),
        ("FALSE", False),
    ),
)
def test_boolean_success(input, expected):
    assert boolean(input) == expected


@pytest.mark.parametrize(
    'input',
    (
        "",
        "123",
        "abc",
    ),
)
def test_boolean_failure(input):
    with pytest.raises(ValueError):
        boolean(input)


@pytest.mark.parametrize(
    'param_strs, param_types, expected',
    (
        (
            (),
            (),
            (),
        ),
        (
            ("1",),
            (str,),
            ("1",),
        ),
        (
            ("1",),
            (int,),
            (1,),
        ),
        (
            ("False",),
            (boolean,),
            (False,),
        ),
        (
            ("true",),
            (boolean,),
            (True,),
        ),
        (
            ("[abc cba]",),
            (functools.partial(list_ctor, element_type=str),),
            (('abc', 'cba'),),
        ),
        (
            ("[1 2]",),
            (functools.partial(list_ctor, element_type=int),),
            ((1, 2),),
        ),
    ),
)
def test_parse_event_params_types_success(param_strs, param_types, expected, monkeypatch):
    mock_event_type = 0
    mock_rpc_logs_params_ctor_map = {
        mock_event_type: param_types,
    }
    monkeypatch.setattr(
        'simulation.logs._rpc_logs_params_ctor_map',
        mock_rpc_logs_params_ctor_map,
    )
    assert parse_event_params(param_strs, mock_event_type) == expected


@pytest.mark.parametrize(
    'param_strs, param_types',
    (
        (
            ("abc",),
            (),
        ),
        (
            ("abc",),
            (int,),
        ),
        (
            ("abc",),
            (boolean,),
        ),
        (
            ("[abc]",),
            (int,),
        ),
    ),
    ids=(
        "inconsistent length between `param_strs` and `param_types`",
        "wrong types in `param_types`: should not be `int`",
        "wrong types in `param_types`: should not be `bool`",
        "wrong types in `param_types`: should be the result of `list_ctor`",
    ),
)
def test_parse_event_params_types_failure(param_strs, param_types, monkeypatch):
    mock_event_type = 0
    mock_rpc_logs_params_ctor_map = {
        mock_event_type: param_types,
    }
    monkeypatch.setattr(
        'simulation.logs._rpc_logs_params_ctor_map',
        mock_rpc_logs_params_ctor_map,
    )
    with pytest.raises(ValueError):
        parse_event_params(param_strs, mock_event_type)


@pytest.mark.parametrize(
    'param_strs, event_type, expected',
    (
        (
            ('127.0.0.1', '10000', '0'),
            RPCLogs.LOG_ADD_PEER_FMT,
            ('127.0.0.1', 10000, 0),
        ),
        (
            ("0", "1", "2", "3"),
            RPCLogs.LOG_BROADCAST_COLLATION_FMT,
            (0, 1, 2, 3),
        ),
        (
            ("[1 3 4]",),
            RPCLogs.LOG_SUBSCRIBE_SHARD_FMT,
            ((1, 3, 4),)
        ),
        (
            ("[1 3 4]",),
            RPCLogs.LOG_UNSUBSCRIBE_SHARD_FMT,
            ((1, 3, 4),)
        ),
        (
            ("[shardCollations_0]",),
            RPCLogs.LOG_DISCOVER_SHARD_FMT,
            (("shardCollations_0",),),
        ),
        (
            ("QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd",),
            RPCLogs.LOG_REMOVE_PEER_FMT,
            ("QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd",),
        ),
        (
            ("true", "/ip4/192.168.0.15/tcp/10001/ipfs/QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd,/ip4/192.168.0.15/tcp/10001/ipfs/QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd"),  # noqa: E501
            RPCLogs.LOG_BOOTSTRAP_FMT,
            (True, "/ip4/192.168.0.15/tcp/10001/ipfs/QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd,/ip4/192.168.0.15/tcp/10001/ipfs/QmXtW5fXrrvmHWPhq3FLHdm4zKnC5FZdhTRynSQT57Yrmd"),  # noqa: E501
        ),
        (
            ("false", ""),
            RPCLogs.LOG_BOOTSTRAP_FMT,
            (False, ""),
        ),
    ),
)
def test_parse_event_params_supported_events(param_strs, event_type, expected):
    assert parse_event_params(param_strs, event_type) == expected


def test_parse_event_params_unsupported_event():
    with pytest.raises(EventHasNoParameter):
        parse_event_params(
            ('1',),
            event_type=0,
        )
