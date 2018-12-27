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


sys.path.append("{}/..".format(Path(__file__).parent))


from simulation.constants import (
    LOG_BROADCASTCOLLATION,
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

    network = Network(0, 30)
    network.connect_barbell()

    nodes = network.nodes
    for node in nodes:
        node.subscribe_shard([0])
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
    time_broadcast = nodes[broadcasting_node].get_log_time(LOG_BROADCASTCOLLATION, 0)
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
    network.kill_nodes()


def test_joining_through_bootnodes():
    num_bootnodes = 1
    num_normal_nodes = 10
    n = Network(1, 10)
    print("Spinning up {} bootnodes...".format(num_bootnodes), end='')
    bootnodes = make_local_nodes(0, num_bootnodes)
    print("done")
    print("Connecting bootnodes...", end='')
    topo = make_barbell_topology(bootnodes)
    connect_nodes(bootnodes, topo)
    print("done")
    print("Checking the connections...", end='')
    ensure_topology(bootnodes, topo)
    print("done")

    print("Spinning up {} nodes...".format(num_normal_nodes), end='')
    nodes = make_local_nodes(num_bootnodes, num_bootnodes + num_normal_nodes, bootnodes_multiaddr)
    print("done")

    print("Sleeping for seconds...", end='')
    time.sleep(3)
    print("done")

    all_nodes = bootnodes + nodes
    actual_topo = get_actual_topology(all_nodes)
    print("actual_topo =", actual_topo)

    print("Cleaning up the nodes...", end='')
    kill_nodes(all_nodes)
    print("done")


def test_reproduce_bootstrapping_issue():
    network = Network(num_bootnodes=1, num_normal_nodes=5)

    print("Sleeping for seconds...", end='')
    time.sleep(3)
    print("done")

    all_nodes = network.nodes
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

    network.kill_nodes()


if __name__ == "__main__":
    test_time_broadcasting_data_single_shard()
    test_joining_through_bootnodes()
    test_reproduce_bootstrapping_issue()
