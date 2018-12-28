import pytest
import subprocess
import time

from simulation.config import (
    PORT_BASE,
    RPC_PORT_BASE,
)
from simulation.logs import (
    LOG_BROADCAST_COLLATION_FINISHED,
    LOG_RECEIVE_MSG,
    LOG_SUBSCRIBE_SHARD_FINISHED,
    LOG_UNSUBSCRIBE_SHARD_FINISHED,
)
from simulation.exceptions import (
    CLIFailure,
)
from simulation.network import (
    get_docker_host_ip,
    make_local_node,
    make_local_nodes,
)
from simulation.node import (
    Node,
)


@pytest.yield_fixture("module")
def unchanged_node():
    """Unchanged when tested, to save the time initializing new nodes every test.
    """
    n = make_local_node(12345)
    n.set_peer_id()
    yield n
    n.close()


@pytest.yield_fixture
def nodes():
    ns = make_local_nodes(0, 3)
    yield ns
    for n in ns:
        n.close()


def test_name(unchanged_node):
    assert unchanged_node.name == "whiteblock-node{}".format(unchanged_node.seed)


def is_node_running(node):
    res = subprocess.run(
        ["docker ps | grep {} | wc -l".format(node.name)],
        shell=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True,
        encoding="utf-8",
    )
    num_lines = int(res.stdout.strip())
    if num_lines > 1:
        raise Exception("Should not grep more than 1 running container named {}".format(
            node.name,
        ))
    return num_lines != 0


def test_close():
    n = make_local_node(0)
    assert is_node_running(n)
    n.close()
    assert not is_node_running(n)


def test_run():
    unrun_node = Node(
        get_docker_host_ip(),
        PORT_BASE,
        RPC_PORT_BASE,
        0,
    )
    assert not is_node_running(unrun_node)
    unrun_node.run()
    assert is_node_running(unrun_node)
    unrun_node.close()


def test_wait_for_log(unchanged_node):
    time.sleep(0.2)
    # XXX: here assume we at least have two lines in the format of "INFO: xxx"
    #      if the log style changes in the container and we get fewer lines, this will block
    #      forever
    log = unchanged_node.wait_for_log("INFO", 1)
    # just confirm the grep'ed log is correct
    assert "INFO" in log.split(' ')


def test_get_log_time(unchanged_node):
    time.sleep(0.2)
    unchanged_node.get_log_time("INFO", 1)


def test_set_peer_id():
    n = make_local_node(0)
    assert n.peer_id is None
    n.set_peer_id()
    assert n.peer_id is not None


def test_multiaddr(nodes):
    unrun_node = Node(
        get_docker_host_ip(),
        PORT_BASE,
        RPC_PORT_BASE,
        0,
    )
    with pytest.raises(ValueError):
        unrun_node.multiaddr
    assert nodes[0].multiaddr == "/ip4/{}/tcp/{}/ipfs/{}".format(
        nodes[0].ip,
        nodes[0].port,
        nodes[0].peer_id
    )
    unrun_node.close()


def test_cli(unchanged_node):
    inexisting_command = "123456"
    res = unchanged_node.cli([inexisting_command])
    assert res.returncode != 0
    existing_command = "listpeer"
    res = unchanged_node.cli([existing_command])
    assert res.returncode == 0
    # wrong type: should be `list` instead of the `string`
    with pytest.raises(CLIFailure):
        unchanged_node.cli(existing_command)


def test_cli_safe(unchanged_node):
    inexisting_command = "123456"
    with pytest.raises(CLIFailure):
        unchanged_node.cli_safe([inexisting_command])
    unchanged_node.cli_safe(["listpeer"])


def test_identify(unchanged_node):
    pinfo = unchanged_node.identify()
    assert "peerID" in pinfo
    assert "multiAddrs" in pinfo


def test_list_peer(unchanged_node):
    peers = unchanged_node.list_peer()
    assert peers == []


def test_add_peer(nodes):
    assert len(nodes[0].list_peer()) == 0
    assert len(nodes[1].list_peer()) == 0
    nodes[0].add_peer(nodes[1])
    # symmetric connection
    assert len(nodes[0].list_peer()) == 1
    assert len(nodes[1].list_peer()) == 1
    # set the wrong `ip`/`port`/`seed` of nodes[2], and nodes[0] try to add it
    ip_2 = nodes[2].ip
    port_2 = nodes[2].port
    seed_2 = nodes[2].seed
    # ip
    nodes[2].ip = "123.456.789.012"
    with pytest.raises(CLIFailure):
        nodes[0].add_peer(nodes[2])
    nodes[2].ip = ip_2
    # port
    nodes[2].port = 123
    with pytest.raises(CLIFailure):
        nodes[0].add_peer(nodes[2])
    nodes[2].port = port_2
    # seed
    nodes[2].seed = 32767
    with pytest.raises(CLIFailure):
        nodes[0].add_peer(nodes[2])
    nodes[2].seed = seed_2
    # add 2 peers(avoid nodes[0] adding nodes[2] again because of `dial backoff `)
    nodes[2].add_peer(nodes[0])


def test_remove_peer(nodes):
    # test: wrong format `peer_id`
    with pytest.raises(CLIFailure):
        nodes[0].remove_peer("123")
    # test: remove non-peer
    nodes[0].remove_peer(nodes[2].peer_id)
    nodes[0].add_peer(nodes[1])
    nodes[0].remove_peer(nodes[1].peer_id)
    # symmetric connection
    assert len(nodes[0].list_peer()) == 0
    assert len(nodes[1].list_peer()) == 0
    # test: remove twice
    nodes[0].remove_peer(nodes[1].peer_id)


def test_list_topic_peer(nodes):
    topic_peers = nodes[0].list_topic_peer()
    # assert the default listening topic is present
    assert "listeningShard" in topic_peers
    assert len(topic_peers["listeningShard"]) == 0
    # test: add_peer and should be one more peer in `listeningShard`
    nodes[0].add_peer(nodes[1])
    assert len(nodes[0].list_topic_peer()["listeningShard"]) == 1
    # test: remove_peer and should be one less peer in `listeningShard`
    nodes[0].remove_peer(nodes[1].peer_id)
    assert len(nodes[0].list_topic_peer()["listeningShard"]) == 0


def test_get_subscribed_shard(unchanged_node):
    shards = unchanged_node.get_subscribed_shard()
    assert shards == []


def test_subscribe_shard(nodes):
    # test: wrong type, should be list of int
    with pytest.raises(TypeError):
        nodes[0].subscribe_shard(0)
    # test: normally subscribe
    nodes[0].subscribe_shard([0])
    assert nodes[0].get_subscribed_shard() == [0]
    assert nodes[1].list_shard_peer([0])["0"] == []
    # test: after adding peers, the node should know the shards its peer subscribes
    nodes[1].add_peer(nodes[0])
    time.sleep(0.2)
    assert nodes[0].peer_id in nodes[1].list_shard_peer([0])["0"]
    # test: subscribe "0" twice, and shards "1", "2"
    nodes[0].subscribe_shard([0, 1, 2])
    assert nodes[0].get_subscribed_shard() == [0, 1, 2]
    # test: ensure there are two logs `LOG_SUBSCRIBE_SHARD_FINISHED` because of `subscribe_shard`
    nodes[0].wait_for_log(LOG_SUBSCRIBE_SHARD_FINISHED, 1)


def test_unsubscribe_shard(nodes):
    # test: wrong type, should be list of int
    with pytest.raises(TypeError):
        nodes[0].unsubscribe_shard(0)
    # test: unsubscribe before subscribe
    nodes[0].unsubscribe_shard([0])
    # test: normally subscribe
    nodes[0].subscribe_shard([0])
    nodes[0].add_peer(nodes[1])
    nodes[0].unsubscribe_shard([0])
    assert nodes[0].get_subscribed_shard() == []
    # test: ensure the peer knows its peer unsubscribes
    assert nodes[1].list_shard_peer([0])["0"] == []
    # test: ensure there are two logs `LOG_UNSUBSCRIBE_SHARD_FINISHED` because of
    #       `unsubscribe_shard`
    nodes[0].wait_for_log(LOG_UNSUBSCRIBE_SHARD_FINISHED, 1)


def test_broadcast_collation(nodes):
    # test: broadcast to a non-subscribed shard
    with pytest.raises(CLIFailure):
        nodes[0].broadcast_collation(0, 1, 100, 123)
    nodes[0].add_peer(nodes[1])
    nodes[1].add_peer(nodes[2])
    nodes[0].subscribe_shard([0])
    nodes[1].subscribe_shard([0])
    nodes[2].subscribe_shard([0])
    time.sleep(2)
    # test: see if all nodes receive the broadcasted collation. Use a bigger size of collation to
    #       avoid the possible small time difference. E.g. sometimes t2 < t1, which does not make
    #       sense.
    nodes[0].broadcast_collation(0, 1, 1000000, 123)
    t0 = nodes[0].get_log_time(LOG_BROADCAST_COLLATION_FINISHED, 0)
    t1 = nodes[0].get_log_time(LOG_RECEIVE_MSG, 0)
    t2 = nodes[1].get_log_time(LOG_RECEIVE_MSG, 0)
    t3 = nodes[2].get_log_time(LOG_RECEIVE_MSG, 0)
    assert t0 < t1
    assert t1 < t2
    assert t2 < t3


def test_bootstrap(nodes):
    # test: stop before start
    nodes[0].bootstrap(False)
    nodes[0].add_peer(nodes[1])
    nodes[1].add_peer(nodes[2])
    time.sleep(0.2)
    # test: nodes[0] bootstrap and get the nodes[2] as peer with nodes[1] as the bootnode
    nodes[0].bootstrap(True, nodes[1].multiaddr)
    time.sleep(2)
    assert nodes[2].peer_id in nodes[0].list_peer()
    # TODO: grep logs


def test_stop():
    n = make_local_node(0)
    assert is_node_running(n)
    n.stop()
    time.sleep(2)  # FIXME: we need to wait for a fairly long time to wait for its stop
    assert not is_node_running(n)
