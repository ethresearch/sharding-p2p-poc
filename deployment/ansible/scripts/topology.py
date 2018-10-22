import sys
import random
from copy import deepcopy
import yaml
from itertools import groupby

RPC_PORT_GAP = 100


class Peer:
    def __init__(self, host, _id):
        self.container_name = f"peer_{_id}"
        self.host = host
        self.public_ip = host.split("@")[1]
        self.listen_port = _id
        self.rpc_port = _id + RPC_PORT_GAP
        self.seed = _id

    def __repr__(self):
        return f"<{self.container_name}: {self.public_ip}, l:{self.listen_port}, r:{self.rpc_port}, s:{self.seed}>"


def parse_inventory(line):
    host, _start, _end = line.strip().split(" ")
    start = int(_start.split("=")[1])
    end = int(_end.split("=")[1])
    return host, start, end


def hosts_to_peers(lines):
    peers = []
    for line in lines:
        host, start, end = parse_inventory(line)
        for i in range(start, end + 1):
            peer = Peer(host, i)
            peers.append(peer)
    return peers


def read_inventories(path):
    with open(path, "r") as f:
        lines = f.readlines()
    key = lines[0].strip()
    inventories = {key: []}
    for _line in lines[1:]:
        line = _line.strip()
        if not line.startswith("["):
            inventories[key].append(line)
        else:
            key = line
            inventories[key] = []
    print(inventories)
    peers = hosts_to_peers(inventories["[nodes]"])
    return peers


def barbell_topology(_inventories):
    inventories = deepcopy(_inventories)
    random.shuffle(inventories)
    n = len(inventories)
    topology = {peer: [inventories[(i + 1) % n]]
                for i, peer in enumerate(inventories)}
    return topology


def topology_to_yaml(topology):
    results = {}
    for me, others in topology.items():
        commands = [
            f"docker exec -t {me.container_name} sh -c './sharding-p2p-poc -loglevel=DEBUG -client addpeer {peer.public_ip} {peer.listen_port} {peer.seed}'"
            for peer in others
        ]
        if me.host in results:
            results[me.host].extend(commands)
        else:
            results[me.host] = commands

    return yaml.dump(results, default_flow_style=False, width=1000)


if __name__ == '__main__':
    inventories = read_inventories(sys.argv[1])
    topology = barbell_topology(inventories)
    for p, l in topology.items():
        print(p.container_name, p.public_ip, l)
    print("\n")
    yml = topology_to_yaml(topology)
    print(yml)
    with open("topology.yml", "w") as f:
        f.write(yml)
