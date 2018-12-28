from datetime import (
    datetime,
)
import logging
import math
import os
from pathlib import (
    Path,
)
import sys
import time

from simulation.logs import (
    LOG_BROADCAST_COLLATION_FINISHED,
    LOG_RECEIVE_MSG,
)
from simulation.network import (
    Network,
    connect_nodes,
    ensure_topology,
    make_barbell_topology,
    make_local_nodes,
)


def test_time_broadcasting_data_single_shard():
    num_collations = 10
    collation_size = 1000000  # 1MB
    collation_time = 50  # broadcast 1 collation every 50 milliseconds
    percent = 0.9

    n = Network(0, 30)
    n.connect_barbell()

    nodes = n.nodes
    for node in nodes:
        node.subscribe_shard([0])

    print("Sleeping for seconds...", end='')
    time.sleep(3)
    print("done")

    broadcasting_node = 0
    print(
        "Broadcasting {} collations(size={} bytes) from node{} in the barbell topology...".format(
            num_collations,
            collation_size,
            broadcasting_node,
        ),
        end='',
    )
    nodes[broadcasting_node].broadcast_collation(0, num_collations, collation_size, collation_time)
    print("done")

    # TODO: maybe we can have a list of broadcasted data, and use them to grep in the nodes' logs
    #       for precision, instead of using only numbers
    # wait until all nodes receive the broadcasted data, and gather the time
    print("Gathering time...", end='')
    time_broadcast = nodes[broadcasting_node].get_log_time(LOG_BROADCAST_COLLATION_FINISHED, 0)
    time_received_list = []
    for i in range(len(nodes)):
        time_received = nodes[i].get_log_time(LOG_RECEIVE_MSG, num_collations - 1,)
        time_received_list.append(time_received)
    print("done")

    # sort the time, find the last node in the first percent% nodes who received the data
    time_received_sorted = sorted(time_received_list, key=lambda t: t)
    index_last = math.ceil(len(nodes) * percent - 1)

    print(
        "time to broadcast all data to {} percent nodes: \x1b[0;37m{}\x1b[0m".format(
            percent,
            time_received_sorted[index_last] - time_broadcast,
        )
    )
    n.kill_nodes()


def test_joining_through_bootnodes():
    n = Network(num_bootnodes=1, num_normal_nodes=10)

    print("Sleeping for seconds...", end='')
    time.sleep(3)
    print("done")

    actual_topo = n.get_actual_topology()
    print("actual_topo =", actual_topo)

    n.kill_nodes()


def test_reproduce_bootstrapping_issue():
    n = Network(num_bootnodes=1, num_normal_nodes=5)

    all_nodes = n.nodes
    for node in all_nodes:
        node.subscribe_shard([1])

    print("Sleeping for seconds...", end='')
    time.sleep(3)
    print("done")

    for node in all_nodes:
        peers = node.list_peer()
        topic_peers = node.list_topic_peer([])
        print("{}: summary: len(peers)={}, len_per_topic={}".format(
            node,
            len(peers),
            {key: len(value) for key, value in topic_peers.items()},
        ))
        print(f"{node}: peers={peers}")
        print(f"{node}: topic_peers={topic_peers}")

    n.kill_nodes()


if __name__ == "__main__":
    l = logging.getLogger("simulation.Network")
    h = logging.StreamHandler()
    h.setLevel(logging.DEBUG)
    l.addHandler(h)
    test_time_broadcasting_data_single_shard()
    test_joining_through_bootnodes()
    test_reproduce_bootstrapping_issue()
