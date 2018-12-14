from collections import (
    defaultdict,
)
import json
import re
import subprocess
import sys
import threading
import os
import time

from dateutil import (
    parser,
)


RPC_PORT_BASE = 13000
PORT_BASE = 10000


def get_docker_host_ip():
    sysname = os.uname().sysname
    if sysname != 'Darwin' and sysname != 'Linux':
        raise ValueError(
            "Failed to get ip in platforms other than Linux and macOS: {}".format(sysname)
        )
    cmd = 'ifconfig | grep -E "([0-9]{1,3}\\.){3}[0-9]{1,3}" | grep -v 127.0.0.1 | awk \'{ print $2 }\' | cut -f2 -d: | head -n1'
    res = subprocess.run(cmd, shell=True, check=True, stdout=subprocess.PIPE, encoding='utf-8')
    return res.stdout.rstrip()


class CLIFailure(Exception):
    pass


class Node:

    ip = None
    port = None
    rpc_port = None
    peer_id = None
    seed = None

    def __init__(self, ip, port, rpc_port, seed):
        self.ip = ip
        self.port = port
        self.rpc_port = rpc_port
        self.seed = seed

    def __repr__(self):
        return "<Node seed={} peer_id={}>".format(
            self.seed,
            None if self.peer_id is None else self.peer_id[2:8],
        )

    @property
    def name(self):
        return f"whiteblock-node{self.seed}"

    @property
    def multiaddr(self):
        return f"/ip4/{self.ip}/tcp/{self.port}/ipfs/{self.peer_id}"

    def close(self):
        subprocess.run(f"docker kill {self.name}", shell=True, stdout=subprocess.PIPE)
        subprocess.run(f"docker rm -f {self.name}", shell=True, stdout=subprocess.PIPE)

    def run(self, bootnodes=None):
        """`bootnodes` should be a list of string. Each string should be a multiaddr.
        """
        self.close()
        bootnodes_cmd = ""
        if bootnodes is not None:
            bootnodes_cmd = "-bootstrap -bootnodes={}".format(
                ",".join(bootnodes),
            )
        cmd = "docker run -d --name {} -p {}:10000 -p {}:13000 ethresearch/sharding-p2p:dev sh -c \"./sharding-p2p-poc -verbose -ip=0.0.0.0 -seed={} {}\"".format(
            self.name,
            self.port,
            self.rpc_port,
            self.seed,
            bootnodes_cmd,
        )
        subprocess.run(cmd, shell=True, stdout=subprocess.PIPE, check=True)

    def cli(self, cmd_list):
        cmd_quoted_param_list = ["'{}'".format(i) for i in cmd_list]
        cmd_quoted_param_str = " ".join(cmd_quoted_param_list)
        return subprocess.run(
            [
                "docker",
                "exec",
                "-t",
                self.name,
                "sh",
                "-c",
                "./sharding-p2p-poc '-client' {}".format(cmd_quoted_param_str),
            ],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            encoding='utf-8',
        )

    def cli_safe(self, cmd):
        res = self.cli(cmd)
        if res.returncode != 0:
            raise CLIFailure(
                "exit_code={}, stdout={!r}, stderr={!r}".format(
                    res.returncode,
                    res.stdout,
                    res.stderr,
                )
            )
        out = res.stdout.rstrip()
        # assume CLIs only reply data in JSON
        if out != '':
            return json.loads(out)
        return out

    def identify(self):
        return self.cli_safe(["identify"])

    def add_peer(self, node):
        self.cli_safe([
            "addpeer",
            node.ip,
            node.port,
            node.seed,
        ])

    def remove_peer(self, peer_id):
        self.cli_safe([
            "removepeer",
            peer_id,
        ])

    def list_peer(self):
        return self.cli_safe(["listpeer"])

    def list_topic_peer(self, topics=[]):
        return self.cli_safe(["listtopicpeer"] + topics)

    def subscribe_shard(self, shard_ids):
        self.cli_safe(["subshard"] + shard_ids)

    def unsubscribe_shard(self, shard_ids):
        self.cli_safe(["unsubshard"] + shard_ids)

    def get_subscribed_shard(self):
        return self.cli_safe(["getsubshard"])

    def broadcast_collation(self, shard_id, num_collations, collation_size, collation_time):
        self.cli_safe([
            "broadcastcollation",
            shard_id,
            num_collations,
            collation_size,
            collation_time,
        ])

    def bootstrap(self, if_start):
        self.cli_safe([
            "bootstrap",
            "start" if if_start else "stop",
        ])

    def stop(self):
        self.cli_safe(["stop"])

    def grep_log(self, pattern):
        res = subprocess.run(
            [
                "docker logs {} 2>&1 | grep '{}'".format(
                    self.name,
                    pattern,
                ),
            ],
            shell=True,
            stdout=subprocess.PIPE,
            encoding='utf-8',
        )
        return res.stdout.rstrip()

    def wait_for_log(self, pattern, k_th):
        """Wait for the `k_th` log in the format of `pattern`
        """
        # TODO: should set a `timeout`?
        cmd = "docker logs {} -t -f 2>&1 | grep --line-buffered '{}' -m {}".format(
            self.name,
            pattern,
            k_th + 1,
        )
        res = subprocess.run(
            [cmd],
            shell=True,
            stdout=subprocess.PIPE,
            encoding='utf-8',
        )
        logs = res.stdout.rstrip()
        log = logs.split('\n')[k_th]
        return log

    def set_peer_id(self):
        pinfo = self.identify()
        self.peer_id = pinfo["peerID"]

    def get_log_time(self, pattern, k_th):
        log = self.wait_for_log(pattern, k_th)
        time_str_iso8601 = log.split(' ')[0]
        return parser.parse(time_str_iso8601)


def make_node(seed, bootnodes=None):
    n = Node(
        get_docker_host_ip(),
        seed + PORT_BASE,
        seed + RPC_PORT_BASE,
        seed,
    )
    n.run(bootnodes)
    return n


def make_local_nodes(low, top, bootnodes=None):
    nodes = []
    threads = []

    def run_node(seed, bootnodes=None):
        node = make_node(seed, bootnodes)
        nodes.append(node)

    for i in range(low, top):
        t = threading.Thread(target=run_node, args=(i, bootnodes))
        t.start()
        threads.append(t)
    for t in threads:
        t.join()
    nodes = sorted(nodes, key=lambda node: node.seed)

    time.sleep(5)

    threads = []
    for node in nodes:
        t = threading.Thread(target=node.set_peer_id, args=())
        t.start()
        threads.append(t)
    for t in threads:
        t.join()
    return nodes


def connect_nodes(nodes, topology):
    for i, targets in topology.items():
        for j in targets:
            nodes[i].add_peer(nodes[j])


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
        assert nodes[j].peer_id in peers_i
        assert nodes[i].peer_id in peers_j

    for i, targets in expected_topology.items():
        for j in targets:
            t = threading.Thread(target=check_connection, args=(nodes, i, j))
            t.start()
            threads.append(t)
    for t in threads:
        t.join()


def make_peer_id_map(nodes):
    return {
        node.peer_id: node.seed
        for node in nodes
    }


def get_actual_topology(nodes):
    map_peer_id_to_seed = make_peer_id_map(nodes)
    topo = defaultdict(set)
    for node in nodes:
        peers = node.list_peer()
        for peer_id in peers:
            peer_seed = map_peer_id_to_seed[peer_id]
            topo[node.seed].add(peer_seed)
    return topo


def kill_nodes(nodes):
    node_names = [n.name for n in nodes]
    subprocess.run(
        ["docker", "kill"] + node_names,
        stdout=subprocess.PIPE,
    )
