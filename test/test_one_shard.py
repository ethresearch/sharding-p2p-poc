#!/usr/bin/env python

import json
import logging
import re
import subprocess
import sys
import threading
import os
import time

# --i <image>
# --n <number of nodes>
# --I <interface> eno4
# example:
# os.system("./umba --i sharding --I eno4 --n 20")
# os.system("docker exec -it whiteblock-node0 -port=8080")
# ip eample: 10.1.0.2 would be whiteblock-node0
# 10.1.0.6 would be whiteblock-node1 ++
# just add the flags and commands that are needed
# we will handle logic of collecting data and configuring umba

#TEST SERIES A
#LATENCY

RPC_PORT_BASE = 13000
PORT_BASE = 10000


logger = logging.getLogger("test_one_shard")


def get_docker_host_ip():
    sysname = os.uname().sysname
    if sysname != 'Darwin' and sysname != 'Linux':
        raise ValueError(
            "Failed to get ip in platforms other than Linux and macOS: {}".format(sysname)
        )
    cmd = 'ifconfig | grep -E "([0-9]{1,3}\\.){3}[0-9]{1,3}" | grep -v 127.0.0.1 | awk \'{ print $2 }\' | cut -f2 -d: | head -n1'
    res = subprocess.run(cmd, shell=True, check=True, stdout=subprocess.PIPE, encoding='utf-8')
    return res.stdout.rstrip()


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
        cmd = "docker run -d --name {} -p {}:10000 -p {}:13000 ethereum/sharding-p2p:latest sh -c \"./sharding-p2p-poc -loglevel=DEBUG -ip=0.0.0.0 -seed={} {}\"".format(
            self.name,
            self.port,
            self.rpc_port,
            self.seed,
            bootnodes_cmd,
        )
        subprocess.run(cmd, shell=True, stdout=subprocess.PIPE, check=True)

    def cli(self, cmd, **kwargs):
        cmd_list = cmd.split(' ')
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
                "./sharding-p2p-poc '-loglevel=DEBUG' '-client' {}".format(cmd_quoted_param_str),
            ],
            stdout=subprocess.PIPE,
            encoding='utf-8',
            **kwargs,
        )

    def cli_safe(self, cmd):
        res = self.cli(cmd, check=True)
        return res.stdout.rstrip()

    def add_peer(self, node):
        self.cli_safe("addpeer {} {} {}".format(node.ip, node.port, node.seed))

    def remove_peer(self, peer_id):
        self.cli_safe("removepeer {}".format(peer_id))

    def list_peer(self):
        return self.cli_safe("listpeer")

    def list_topic_peer(self, topics):
        return self.cli_safe("listtopic {}".format(' '.join(topics)))

    def subscribe_shard(self, shard_ids):
        return self.cli_safe("subshard {}".format(' '.join(map(str, shard_ids))))

    def unsubscribe_shard(self, shard_ids):
        return self.cli_safe("unsubshard {}".format(' '.join(map(str, shard_ids))))

    def get_subscribed_shard(self):
        return self.cli_safe("getsubshard")

    def broadcast_collation(self, shard_id, num_collations, collation_size, collation_time):
        return self.cli_safe("broadcastcollation {} {} {} {}".format(
            shard_id,
            num_collations,
            collation_size,
            collation_time,
        ))

    def stop(self):
        return self.cli_safe("stop")

    def grep_log(self, pattern):
        res = subprocess.run(
            [
                "docker logs {} 2>&1 | grep '{}'".format(self.name, pattern),
            ],
            shell=True,
            stdout=subprocess.PIPE,
            encoding='utf-8',
        )
        return res.stdout.rstrip()

    def set_peer_id(self):
        grep_res = self.grep_log('Node is listening')
        match = re.search('peerID=([a-zA-Z0-9]+) ', grep_res)
        if match is None:
            raise ValueError("failed to grep the peer_id from docker logs")
        self.peer_id = match[1]


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
    sorted(nodes, key=lambda node: node.seed)

    time.sleep(2)
    threads = []
    for node in nodes:
        t = threading.Thread(target=node.set_peer_id, args=())
        t.start()
        threads.append(t)
    for t in threads:
        t.join()
    return nodes


def connect_barbell(nodes):
    threads = []
    for i in range(len(nodes) - 1):
        t = threading.Thread(target=nodes[i].add_peer, args=(nodes[i + 1],))
        t.start()
        threads.append(t)
    for t in threads:
        t.join()


def connect_fully(nodes):
    threads = []
    for i in range(len(nodes) - 1):
        for j in range(i + 1, len(nodes)):
            t = threading.Thread(target=nodes[i].add_peer, args=(nodes[j],))
            t.start()
            threads.append(t)
    for t in threads:
        t.join()


if __name__ == "__main__":
    num_bootnodes = 5
    num_normal_nodes = 4
    print("Spinning up {} bootnodes...".format(num_bootnodes), end='')
    bootnodes = make_local_nodes(0, num_bootnodes)
    print("done")
    print("Connecting bootnodes...", end='')
    connect_barbell(bootnodes)
    print("done")

    for node in bootnodes:
        node.subscribe_shard([0])
    bootnodes[0].broadcast_collation(0, 10, 100, 100)
    print("Waiting for messages broadcasted...", end='')
    time.sleep(2)
    print("done")
    log = bootnodes[-1].grep_log('Validating the received message')
    print(log)
    sys.exit(1)


    bootnodes_multiaddr = [node.multiaddr for node in bootnodes]
    print("Spinning up {} nodes...".format(num_normal_nodes), end='')
    nodes = make_local_nodes(num_bootnodes, num_bootnodes + num_normal_nodes, bootnodes_multiaddr)
    print("done")

    print("Sleeping for seconds...", end='')
    time.sleep(3)
    print("done")

    print(nodes[-1].list_peer())
    # print(nodes)
