import argparse
import configparser
import io
import json
import os
import subprocess

import yaml


LISTENPORT_BASE = 10000
RPCPORT_BASE = 20000

_server_sno = 0
_seed = 0


def inc_seed():
    global _seed
    temp = _seed
    _seed += 1
    return temp


def inc_server_sno():
    global _server_sno
    temp = _server_sno
    _server_sno += 1
    return temp


def make_listenport(seed):
    return seed + LISTENPORT_BASE


def make_rpcport(seed):
    return seed + RPCPORT_BASE


def get_docker_host_ip():
    sysname = os.uname().sysname
    if sysname != 'Darwin' and sysname != 'Linux':
        raise ValueError(
            "Failed to get ip in platforms other than Linux and macOS: {}".format(sysname)
        )
    cmd = 'ifconfig | grep -E "([0-9]{1,3}\\.){3}[0-9]{1,3}" | grep -v 127.0.0.1 | awk \'{ print $2 }\' | cut -f2 -d: | head -n1'
    res = subprocess.run(cmd, shell=True, stdout=subprocess.PIPE)
    return res.stdout.decode().replace('\n', '')


def distribute_number_to_portions(number, portions):
    accummed = 0
    num_list = []
    if len(portions) == 0:
        raise ValueError("len(portions) should >= 1")
    if sum(portions) > 1:
        raise ValueError("sum of portions should be less than 1, portions={}".format(portions))
    for idx, portion in enumerate(portions):
        num_in_this_portion = int(number * portion)
        if idx == (len(portions) - 1):
            num_in_this_portion = number - accummed
        num_list.append(num_in_this_portion)
        accummed += num_in_this_portion
    assert sum(num_list) == number
    return num_list


def make_node_name(server_name, node_index, seed):
    return "{}-{}-{}".format(server_name, node_index, seed)


def is_local(server_name):
    return server_name == 'local'


def extract(config):
    docker_host_ip = get_docker_host_ip()
    server_list = []
    node_list = []
    server_portions = tuple((
        server['portion']
        for region_obj in config['servers']
        for server in region_obj['list']
    ))
    num_nodes = config['nodes']['number']
    num_nodes_in_servers = distribute_number_to_portions(num_nodes, server_portions)

    for region_obj in config['servers']:
        for server_idx, server in enumerate(region_obj['list']):
            if is_local(server['name']):
                server['ip'] = docker_host_ip  # use local ip instead of the loopback ip
            # add to server list
            server_list.append(server)
            # add nodes in this server to node list
            # seed, listenport, rpcport, targets
            for node_idx in range(num_nodes_in_servers[server_idx]):
                seed = inc_seed()
                node_info = {
                    'name': make_node_name(server['name'], node_idx, seed),
                    'server_info': server,
                    'ip': server['ip'],
                    'seed': seed,
                    'listenport': make_listenport(seed),
                    'rpcport': make_rpcport(seed),
                    'targets': [],  # FIXME: need topology map to do it
                }
                node_list.append(node_info)
    return server_list, node_list


def make_addr_info_str(server_info):
    if is_local(server_info['name']):
        addr_info = "ansible_connection=local"
    else:
        addr_info = "ansible_host={} ansible_port={}".format(
            server_info["ip"],
            server_info["port"],
        )
        if "user" in server_info:
            addr_info += " ansible_user={}".format(server_info["user"])
        if "pass" in server_info:
            addr_info += " ansible_pass={}".format(server_info["pass"])
    return addr_info


def write_inv(server_list, node_list, f_ini):
    f_ini.write("[servers]\n")
    for server_info in server_list:
        addr_info = make_addr_info_str(server_info)
        f_ini.write("{} {}\n".format(server_info["name"], addr_info))

    f_ini.write("\n[nodes]\n")
    for node_info in node_list:
        addr_info = make_addr_info_str(node_info['server_info'])
        targets_str = json.dumps(node_info['targets'])
        f_ini.write(
            ' '.join((
                '{}'.format(node_info['name']),
                '{}'.format(addr_info),
                'listenport={}'.format(node_info['listenport']),
                'rpcport={}'.format(node_info['rpcport']),
                'seed={}'.format(node_info['seed']),
                'targets={}'.format(targets_str),
            )) + '\n'
        )


def gen_inv(f_config, f_ini):
    config = yaml.load(f_config)
    server_list, node_list = extract(config)
    write_inv(server_list, node_list, f_ini)


def test_gen_inv():
    config = """servers:
  - region: local
    list:
      - name: local
        ip: 192.168.0.15
        portion: 1
nodes:
  number: 5
"""
    f_config = io.StringIO(config)
    f_ini = io.StringIO()
    gen_inv(f_config, f_ini)
    f_ini.seek(0, 0)
    ini = f_ini.read()
    assert ini == """[servers]
local ansible_connection=local

[nodes]
local-0-0 ansible_connection=local listenport=10000 rpcport=20000 seed=0 targets=[]
local-1-1 ansible_connection=local listenport=10001 rpcport=20001 seed=1 targets=[]
local-2-2 ansible_connection=local listenport=10002 rpcport=20002 seed=2 targets=[]
local-3-3 ansible_connection=local listenport=10003 rpcport=20003 seed=3 targets=[]
local-4-4 ansible_connection=local listenport=10004 rpcport=20004 seed=4 targets=[]
"""


def main():
    parser = argparse.ArgumentParser(
        description="Generate an ansible inventory file from the yaml config file",
    )
    parser.add_argument('config', type=str, help="yaml config file")
    parser.add_argument('out', type=str, help="output yaml file")
    args = parser.parse_args()
    with open(args.config, 'r') as f_config, open(args.out, 'w') as f_ini:
        gen_inv(f_config, f_ini)


if __name__ == '__main__':
    main()
