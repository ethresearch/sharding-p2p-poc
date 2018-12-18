# -*- coding: utf-8 -*-

import json
import os
import re
import subprocess


least_go_version_str = '1.11'
# ensure go version >= `least_go_version_str`
res_go_version = subprocess.run(["go", "version"], stdout=subprocess.PIPE)
if res_go_version.returncode != 0:
    raise Exception("failed to run `go version`")
res_go_version_str = res_go_version.stdout.decode()
ret = re.search(r"go\sversion\sgo([0-9]+\.[0-9]+)", res_go_version_str)
if ret is None:
    raise Exception("failed to parse the version from `go version`")
version_str = ret.groups(0)[0]
if version_str < least_go_version_str:
    raise Exception("go version should be >= {}".format(least_go_version_str))


GOPATH = os.getenv('GOPATH')


def get_gx_path(gx_hash, repo_name):
    return "{}/src/gx/ipfs/{}/{}".format(GOPATH, gx_hash, repo_name)


def get_github_repo_path(github_repo):
    return "{}/src/{}".format(GOPATH, github_repo)


def download_and_update_deps(root_repo_path):
    # subprocess.run(
    #     "cd GO111MODULE=off go get -d -u {}".format(github_repo),
    #     shell=True,
    # )
    pass


def update_repo_to_go_mod(root_repo_path):
    visited_repos = set()
    queue = list()
    queue.append(('', root_repo_path))
    while len(queue) != 0:
        repo_gx_hash, repo_path = queue.pop()
        if repo_path in visited_repos:
            continue
        visited_repos.add(repo_path)
        json_file_path = "{}/package.json".format(repo_path)
        with open(json_file_path, 'r') as f_read:
            package_info = json.load(f_read)
        # add the deps
        if 'gxDependencies' in package_info:
            deps = package_info['gxDependencies']
            for dep_info in deps:
                dep_name = dep_info['name']
                dep_gx_hash = dep_info['hash']
                dep_gx_path = get_gx_path(dep_gx_hash, dep_name)
                queue.append((dep_gx_hash, dep_gx_path))
        # exclude the root repo
        if repo_path == root_repo_path:
            continue

        # TODO: handle go get here!
        # 1. need `github_repo` + `gx_hash`
        # 2. `git --git-dir=`github_repo`/.git rev-list|xargs git grep `gx_hash`, and find the first commit containing this hash
        # 3. go get `github_repo`@`commit`
        github_repo = package_info['gx']['dvcsimport']
        github_repo_path = get_github_repo_path(github_repo)
        print("handling repo: gx_hash={}, repo_path={}, github_path={}".format(repo_gx_hash, repo_path, github_repo_path))
        res = subprocess.run(
            "git --git-dir={0}/.git rev-list --all | xargs git --git-dir={0}/.git grep {1} | tail -n1".format(
                github_repo_path,
                repo_gx_hash,
            ),
            shell=True,
            stdout=subprocess.PIPE,
        )
        if res.returncode == 0:
            # success
            try:
                result = res.stdout.decode()
                module_commit = result.split(':')[0]
            except IndexError:
                raise Exception("fail to parse the commit record: {}".format(module_commit))
            res = subprocess.run(
                "GO111MODULE=on go get {}@{}".format(github_repo, module_commit),
                shell=True,
            )


if __name__ == "__main__":
    current_path = os.getcwd()
    update_repo_to_go_mod(current_path)
