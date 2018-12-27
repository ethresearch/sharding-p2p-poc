from collections import (
    defaultdict,
)
import logging
import subprocess
import threading
import os
import time

from .config import (
    PORT_BASE,
    RPC_PORT_BASE,
)
from .exceptions import (
    WrongTopology,
)
from .node import (
    Node,
)


def get_docker_host_ip():
    sysname = os.uname().sysname
    if sysname != 'Darwin' and sysname != 'Linux':
        raise ValueError(
            "Failed to get ip in platforms other than Linux and macOS: {}".format(sysname)
        )
    cmd = 'ifconfig | grep -E "([0-9]{1,3}\\.){3}[0-9]{1,3}" | grep -v 127.0.0.1 | awk \'{ print $2 }\' | cut -f2 -d: | head -n1'
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
    nodes = []
    threads = []

    def run_node(seed, bootnodes=None):
        node = make_local_node(seed, bootnodes)
        nodes.append(node)

    for i in range(low, top):
        t = threading.Thread(target=run_node, args=(i, bootnodes))
        t.start()
        threads.append(t)
    for t in threads:
        t.join()
    nodes = sorted(nodes, key=lambda node: node.seed)

    time.sleep(5)

    for node in nodes:
        node.set_peer_id()
    return nodes


def connect_nodes(nodes, topology):
    for i, targets in topology.items():
        for j in targets:
            nodes[i].add_peer(nodes[j])


def make_peer_id_map(nodes):
    return {
        node.peer_id: node.seed
        for node in nodes
    }


def make_barbell_topology(nodes):
    topo = defaultdict(set)
    for i in range(len(nodes) - 1):
        topo[i].add(i + 1)
    return topo


def make_complete_topology(nodes):
    topo = defaultdict(set)
    for i in range(len(nodes) - 1):
        for j in range(i + 1, len(nodes)):
            topo[i].add(j)
    return topo


def ensure_topology(nodes, expected_topology):
    if len(nodes) <= 1:
        return

    threads = []

    def check_connection(nodes, i, j):
        peers_i = nodes[i].list_peer()
        peers_j = nodes[j].list_peer()
        # assume symmetric connections
        if not (nodes[j].peer_id in peers_i and nodes[i].peer_id in peers_j):
            raise WrongTopology("Nodes are not connected as expected_topology={}".format(
                expected_topology,
            ))

    for i, targets in expected_topology.items():
        for j in targets:
            t = threading.Thread(target=check_connection, args=(nodes, i, j))
            t.start()
            threads.append(t)
    for t in threads:
        t.join()


class Network:

    bootnodes = None
    normal_nodes = None
    topology = None

    logger = logging.getLogger("simulation.network")

    def __init__(self, num_bootnodes, num_normal_nodes):
        self.logger.info("Spinning up %s bootnodes", num_bootnodes)
        self.bootnodes = make_local_nodes(0, num_bootnodes)
        # TODO: confirm whether to connect bootnodes in advance
        # connect_nodes(
        #     self.bootnodes,
        #     make_barbell_topology(self.bootnodes),
        # )
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

    def _connect(self, topo_factory):
        if self.has_connected():
            self.logger.warning("connected before")
            raise Exception("connected before")
        self.logger.info("Connecting nodes")
        self.topology = topo_factory(self.nodes)
        connect_nodes(self.nodes, self.topology)
        ensure_topology(self.nodes, self.topology)

    def connect_barbell(self):
        self._connect(make_barbell_topology)

    def connect_completely(self):
        self._connect(make_complete_topology)

    def _verify_topology(self):
        if len(self.bootnodes) != 0:
            self.logger.warning("topology might change when bootnodes are used")
        self.logger.info("Verifying topogloy")
        ensure_topology(self.nodes, self.topology)

    def get_actual_topology(self):
        map_peer_id_to_seed = make_peer_id_map(self.nodes)
        topo = defaultdict(set)
        for node in self.nodes:
            peers = node.list_peer()
            for peer_id in peers:
                peer_seed = map_peer_id_to_seed[peer_id]
                topo[node.seed].add(peer_seed)
        return topo

    def kill_nodes(self):
        node_names = [n.name for n in self.nodes]
        self.logger.info("Killing nodes")
        subprocess.run(
            ["docker", "kill"] + node_names,
            stdout=subprocess.PIPE,
        )
