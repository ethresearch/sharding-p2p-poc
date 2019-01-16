import json
import subprocess

from dateutil import (
    parser,
)

from .exceptions import (
    CLIFailure,
)
from .log_aggregation import (
    NoMatchingPattern,
    ParsingError,
    parse_line,
)

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
        if self.peer_id is None:
            raise ValueError("`peer_id` has not been set")
        return f"/ip4/{self.ip}/tcp/{self.port}/ipfs/{self.peer_id}"

    def close(self):
        subprocess.run(
            ["docker", "kill", self.name],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
        subprocess.run(
            ["docker", "rm", "-f", self.name],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )

    def run(self, bootnodes=None):
        """`bootnodes` should be a list of string. Each string should be a multiaddr.
        """
        self.close()
        bootnodes_cmd = ""
        if (bootnodes is not None) and (len(bootnodes) != 0):
            bootnodes_cmd = "-bootstrap -bootnodes={}".format(
                ",".join(bootnodes),
            )
        cmd = "docker run -d --name {} -p {}:10000 -p {}:13000 ethresearch/sharding-p2p:dev sh -c \"./sharding-p2p-poc -verbose -ip=0.0.0.0 -seed={} {}\"".format(  # noqa: E501
            self.name,
            self.port,
            self.rpc_port,
            self.seed,
            bootnodes_cmd,
        )
        subprocess.run(cmd, shell=True, stdout=subprocess.PIPE, check=True)

    def cli(self, cmd_list):
        if not isinstance(cmd_list, list):
            raise CLIFailure("`cmd_list` should be of `list` type: cmd_list={!r}".format(cmd_list))
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

    def list_topic_peer(self, topics=None):
        if topics is None:
            topics = []
        return self.cli_safe(["listtopicpeer"] + topics)

    def list_shard_peer(self, shards=None):
        if shards is None:
            shards = []
        return self.cli_safe(["listshardpeer"] + shards)

    def subscribe_shard(self, shard_ids, num_shard_peer_to_connect=0):
        self.cli_safe(["subshard", num_shard_peer_to_connect] + shard_ids)

    def unsubscribe_shard(self, shard_ids):
        self.cli_safe(["unsubshard"] + shard_ids)

    def get_subscribed_shard(self):
        return self.cli_safe(["getsubshard"])

    def discover_shard(self, shard_ids):
        return self.cli_safe(["discovershard"] + shard_ids)

    def broadcast_collation(self, shard_id, num_collations, collation_size, collation_time):
        self.cli_safe([
            "broadcastcollation",
            shard_id,
            num_collations,
            collation_size,
            collation_time,
        ])

    def bootstrap(self, if_start, bootnodes_str=None):
        if if_start:
            if bootnodes_str is None:
                raise CLIFailure("if `bootstrap start`, `bootnodes_str` should not be `None`")
            self.cli_safe([
                "bootstrap",
                "start",
                bootnodes_str,
            ])
        else:
            self.cli_safe([
                "bootstrap",
                "stop",
            ])

    def stop(self):
        self.cli_safe(["stop"])

    def wait_for_log(self, pattern, k_th):
        """Wait for the `k_th` log in the format of `pattern`
        """
        # TODO: should set a `timeout`?

        # This `sed` removes the color control code, to make `grep` work easier
        # FIXME: it seems `wait_for_log` gets far slower after adding `cmd_rm_color_control`,
        #        possibly because of the buffering in `sed`? But the situation didn't get better
        #        even though added `-l`(line buffering) to `sed` in macOS.
        cmd_rm_color_control = "sed -l $'s/\\x1b\\\\[[0-9;]*[a-zA-Z]//g'"
        cmd = "docker logs {} -t -f 2>&1 | {} | grep --line-buffered -E '{}' -m {}".format(
            self.name,
            cmd_rm_color_control,
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

    def get_logs(self):
        cmd_rm_color_control = "sed $'s/\\x1b\\\\[[0-9;]*[a-zA-Z]//g'"
        cmd = "docker logs {} -t 2>&1 | {}".format(
            self.name,
            cmd_rm_color_control,
        )
        res = subprocess.run(
            [cmd],
            shell=True,
            stdout=subprocess.PIPE,
            encoding='utf-8',
        )
        logs = res.stdout.rstrip().split('\n')
        for line in logs:
            yield line

    def get_events(self):
        for line in self.get_logs():
            try:
                event = parse_line(line)
                yield event
            except (NoMatchingPattern, ParsingError):
                continue
