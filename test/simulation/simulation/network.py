import logging
from multiprocessing.pool import (
    ThreadPool,
)
import os
import subprocess
import time

from .config import (
    PORT_BASE,
    RPC_PORT_BASE,
)
from .exceptions import (
    InvalidTopology,
    WrongTopology,
)
from .node import (
    Node,
)


THREAD_POOL_SIZE = 30


def get_docker_host_ip():
    sysname = os.uname().sysname
    if sysname != 'Darwin' and sysname != 'Linux':
        raise ValueError(
            "Failed to get ip in platforms other than Linux and macOS: {}".format(sysname)
        )
    cmd = 'ifconfig | grep -E "([0-9]{1,3}\\.){3}[0-9]{1,3}" | grep -v 127.0.0.1 | awk \'{ print $2 }\' | cut -f2 -d: | head -n1'  # noqa: E501
    res = subprocess.run(cmd, shell=True, check=True, stdout=subprocess.PIPE, encoding='utf-8')
    return res.stdout.rstrip()


def make_local_node(seed, bootnodes=None):
    n = Node(
        get_docker_host_ip(),
        seed + PORT_BASE,
        seed + RPC_PORT_BASE,
        seed,
    )
    if bootnodes is None:
        bootnodes_multiaddr = None
    else:
        bootnodes_multiaddr = [node.multiaddr for node in bootnodes]
    n.run(bootnodes_multiaddr)
    return n


def make_local_nodes(low, top, bootnodes=None):

    def run_node(seed):
        return make_local_node(seed, bootnodes)

    pool = ThreadPool(THREAD_POOL_SIZE)
    nodes = pool.map(
        run_node,
        range(low, top),
    )
    nodes_sorted = sorted(nodes, key=lambda node: node.seed)

    time.sleep(5)

    for node in nodes_sorted:
        node.set_peer_id()
    return nodes_sorted


def _make_conn_tuple(a, b):
    if a > b:
        return b, a
    return a, b


def connect_nodes(nodes, topology):
    """Topology should be in the form of `{(node0, node1), (node0, node2)}`
    """
    for conn in topology:
        try:
            index0, index1 = conn
        except:
            raise InvalidTopology("conn={}, topology={}".format(conn, topology))
        nodes[index0].add_peer(nodes[index1])
    time.sleep(1)


def make_peer_id_map(nodes):
    return {
        node.peer_id: node.seed
        for node in nodes
    }


# FIXME: change topology to `set` of `set`
def make_barbell_topology(nodes):
    topology = set()
    for i in range(len(nodes) - 1):
        topology.add(_make_conn_tuple(i, i + 1))
    return topology


def make_complete_topology(nodes):
    topology = set()
    for i in range(len(nodes) - 1):
        for j in range(i + 1, len(nodes)):
            topology.add(_make_conn_tuple(i, j))
    return topology


def contains_topology(nodes, expected_topology):
    if len(nodes) <= 1:
        return

    def check_connection(conn):
        try:
            i, j = conn
        except:
            raise InvalidTopology("conn={}, expected_topology={}".format(conn, expected_topology))
        peers_i = nodes[i].list_peer()
        peers_j = nodes[j].list_peer()
        # assume symmetric connections
        if not (nodes[j].peer_id in peers_i and nodes[i].peer_id in peers_j):
            raise WrongTopology("Nodes are not connected as expected_topology={}".format(
                expected_topology,
            ))

    pool = ThreadPool(THREAD_POOL_SIZE)
    pool.map(
        check_connection,
        expected_topology,
    )


class Network:

    bootnodes = None
    normal_nodes = None
    topology = None

    logger = logging.getLogger("simulation.Network")

    def __init__(self, num_bootnodes, num_normal_nodes, connect_bootnodes=True):
        self.logger.setLevel(logging.DEBUG)
        self.logger.info("Spinning up %s bootnodes", num_bootnodes)
        self.bootnodes = make_local_nodes(0, num_bootnodes)
        # connect bootnodes with full topology
        if connect_bootnodes:
            connect_nodes(
                self.bootnodes,
                make_complete_topology(self.bootnodes),
            )
        self.logger.info("Spinning up %s normal nodes", num_normal_nodes)
        self.normal_nodes = make_local_nodes(
            num_bootnodes,
            num_bootnodes + num_normal_nodes,
            bootnodes=self.bootnodes,
        )

    @property
    def nodes(self):
        return self.bootnodes + self.normal_nodes

    def has_connected(self):
        return self.topology is not None

    def verify_topology(self):
        if len(self.bootnodes) != 0:
            self.logger.warning("topology might change when bootnodes are used")
        self.logger.info("Verifying topogloy")
        actual_topology = self.get_actual_topology()
        if self.topology != actual_topology:
            raise WrongTopology(
                "Topology mismatch: actual_topology={}, expected_topology={}".format(
                    actual_topology,
                    self.topology,
                )
            )

    def _connect(self, topo_factory):
        if self.has_connected():
            self.logger.warning("connected before")
            raise Exception("connected before")
        self.logger.info("Connecting nodes")
        self.topology = topo_factory(self.nodes)
        connect_nodes(self.nodes, self.topology)

    def connect_barbell(self):
        self._connect(make_barbell_topology)

    def connect_completely(self):
        self._connect(make_complete_topology)

    def get_actual_topology(self):
        map_peer_id_to_seed = make_peer_id_map(self.nodes)
        topology = set()
        for node in self.nodes:
            peers = node.list_peer()
            for peer_id in peers:
                peer_seed = map_peer_id_to_seed[peer_id]
                topology.add(_make_conn_tuple(node.seed, peer_seed))
        return topology

    def kill_nodes(self):
        node_names = [n.name for n in self.nodes]
        self.logger.info("Killing nodes")
        subprocess.run(
            ["docker", "kill"] + node_names,
            stdout=subprocess.PIPE,
        )

    # TODO: log aggregation?
