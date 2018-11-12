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
N = 100


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
        subprocess.run(f"docker kill {self.name}", shell=True)
        subprocess.run(f"docker rm -f {self.name}", shell=True)

    def run_docker(self, bootnodes=None):
        """`bootnodes` should be a list of string. Each string should be a multiaddr.
        """
        self.close()
        bootnodes_cmd = ""
        if bootnodes is not None:
            bootnodes_cmd = "-bootstrap -bootnodes={}".format(
            ",".join(bootnodes),
        )
        cmd = "docker run -d --name {} -p {}:10000 -p {}:13000 ethereum/sharding-p2p:dev sh -c \"./sharding-p2p-poc -loglevel=DEBUG -ip=0.0.0.0 -seed={} {}\"".format(
            self.name,
            self.port,
            self.rpc_port,
            self.seed,
            bootnodes_cmd,
        )
        print(cmd)
        subprocess.run(cmd, shell=True, check=True)

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
                "./sharding-p2p-poc '-client' {}".format(cmd_quoted_param_str),
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
    n.run_docker(bootnodes)
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
    # nodes = []
    # for i in range(low, top):
    #     nodes.append(make_node(i, bootnodes))
    time.sleep(4)
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
    bootnodes = make_local_nodes(0, 3)
    connect_fully(bootnodes)
    print(bootnodes[-1].list_peer())

    bootnodes_multiaddr = [node.multiaddr for node in bootnodes]

    nodes = make_local_nodes(3, 5, bootnodes_multiaddr)

    time.sleep(3)
    # nodes = make_local_nodes(0, 25)
    # connect_barbell(nodes)
    # time.sleep(2)

    print(nodes[-2].list_peer())




    import sys
    sys.exit(1)

    #A1
    #0ms
    print("A1 Start")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000") 
    # os.system("docker exec -it whiteblock-node0 -port=8080")

    print("A1 Complete")
    print("Going To Sleep")
    time.sleep(1200)

    print("Parsing & Saving Data")

    os.system("")

    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x A1")

    print("Shutting Down")

    #A2
    #50ms 
    print("A2 Start")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0 --latency 12") 


    print("A2 Complete")

    print("Going To Sleep")
    time.sleep(1800)

    print("Parsing & Saving Data")
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x A2")

    print("Shutting Down")

    #A3
    #100ms
    print("A3 Start")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0 --latency 25") 

    print("A3 Complete")

    print("Going To Sleep")
    time.sleep(1800)

    print("Parsing & Saving Data")
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x A3")

    print("Shutting Down")

    #TEST SERIES B
    #TRANSACTION VOLUME

    #B1
    #8000 tx
    print("B1 Start")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 8000") 
    print("B1 Complete")

    print("Going To Sleep")
    time.sleep(1800)

    print("Parsing & Saving Data")
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x B1")

    #B2
    #16000 tx
    print("B2 Start")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 16000") 
    print("B2 Complete")

    print("Going To Sleep")
    time.sleep(1800)

    print("Parsing & Saving Data")
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x B2")
    print("Shutting Down")

    #B3
    #20000tx
    print("B3 Start")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 20000") 
    print("B3 Complete")

    print("Going To Sleep")
    time.sleep(1800)

    print("Parsing & Saving Data")
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x B3")
    print("Shutting Down")

    #TEST SERIES C
    #TRANSACTION SIZE

    #C1
    #500B
    print("C1 Start")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --tx-size 259") 
    print("C1 Complete")

    print("Going To Sleep")
    time.sleep(1200)

    print("Parsing & Saving Data")
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x C1")
    print("Shutting Down")

    #C2
    #1000B
    print("C2 Start")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --tx-size 344") 
    print("C2 Complete")

    print("Going To Sleep")
    time.sleep(1200)

    print("Parsing & Saving Data")
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x C2")
    print("Shutting Down")

    #C3
    #2000B
    print("C3 Start")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --tx-size 429") 
    print("C3 Complete")

    print("Going To Sleep")
    time.sleep(1200)

    print("Parsing & Saving Data")
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x C3")
    print("Shutting Down")

    #TEST SERIES D
    #HIGHER LATENCY

    #D1
    #200ms
    print("D1 Start")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --latency 50") 
    print("D1 Complete")

    time.sleep(1800)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x D1")

    #D2
    #300ms
    print("D2")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22 --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --latency 75") 


    time.sleep(1800)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x D2")


    #D3
    #400ms
    print("D3")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0 --latency 100") 


    time.sleep(1800)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x D3")


    #TEST SERIES E
    #PACKET LOSS

    #E1
    #0.01%
    print("E1")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0.0025 --latency 0") 

    time.sleep(1800)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x E1")


    #E2
    #0.1%
    print("E2")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0.025 --latency 0") 


    time.sleep(1800)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x E2")


    #E3
    #1%
    print("E3")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 24 --number-of-tx 4000 --emulate --loss 0.25 --latency 0") 


    time.sleep(1800)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x E3")

    #TEST SERIES F
    #NUMBER OF BLOCK PRODUCERS

    #F1
    #3BP
    print("F1")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 4  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 4 --clients 24 --number-of-tx 4000") 

    time.sleep(1200)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x F1")

    #F2
    #11BP
    print("F2")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 12  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 12 --clients 24 --number-of-tx 4000") 

    time.sleep(1200)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x F2")

    #F3
    #15BP
    print("F3")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 16  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 16 --clients 24 --number-of-tx 4000") 

    time.sleep(1200)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x F3")

    #TEST SERIES G
    #HIGHER NUMBER OF CLIENTS

    #G1
    #100
    print("G1")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 100 --number-of-tx 4000") 

    time.sleep(1200)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x G1")

    #G2
    #200
    print("G2")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 200 --number-of-tx 4000") 

    time.sleep(1200)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x G2")

    #G3
    #300
    print("G3")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 300 --number-of-tx 4000") 

    time.sleep(1200)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x G3")

    #TEST SERIES H
    #H1 DONKEY KONG
    print("H1 Start")
    os.chdir("/home/appo/umba")
    os.system("./umba -i sharding -n 22  --servers alpha,bravo,charlie,echo,foxtrot --verbose --bp 22 --clients 3000 --number-of-tx 4000 --emulate --loss 0.5 --latency 25") 


    time.sleep(7200)
    os.system("~/rpc/rpc --ip 10.2.0.2 -S $(cat block_num.txt) -x E3")
