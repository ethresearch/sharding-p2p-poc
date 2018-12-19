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
TMP_GIT_REPO_PATH = "/tmp/gx-git-repos"


class DownloadFailure(Exception):
    pass


class FetchCommitFailure(Exception):
    pass


def make_gx_path(gx_hash, repo_name):
    return "{}/src/gx/ipfs/{}/{}".format(GOPATH, gx_hash, repo_name)


def make_github_repo_path(github_repo):
    return "{}/src/{}".format(GOPATH, github_repo)


def make_git_cmd(path, command):
    """Helper function to do git operations avoiding changing directories
    """
    return "git --git-dir={}/.git {}".format(path, command)


def download_git_repo(git_repo):
    repo_path = "{}/{}".format(TMP_GIT_REPO_PATH, git_repo)
    if not os.path.exists(TMP_GIT_REPO_PATH):
        try:
            os.mkdir(TMP_GIT_REPO_PATH)
        except FileExistsError:
            pass

    if not os.path.exists(repo_path):
        git_url = "https://{}".format(git_repo)
        res = subprocess.run("git clone {} {}".format(git_url, repo_path), shell=True)
    else:
        res = subprocess.run(
            make_git_cmd(repo_path, "pull origin master"),
            shell=True,
        )
    if res.returncode != 0:
        raise DownloadFailure("failed to download/update the git repo: repo={}, res={}".format(
            git_repo,
            res,
        ))


def get_commit_from_git_repo(git_repo, gx_hash):
    git_repo_path = make_github_repo_path(git_repo)
    res = subprocess.run(
        "{} | xargs {} | tail -n1".format(
            make_git_cmd(git_repo_path, "rev-list --all"),
            make_git_cmd(git_repo_path, "grep {}".format(gx_hash)),
        ),
        shell=True,
        stdout=subprocess.PIPE,
    )
    if res.returncode != 0:
        raise FetchCommitFailure("failed to fetch the commit: gx_hash={}, repo={}".format(
            gx_hash,
            git_repo,
        ))
    # success
    try:
        result = res.stdout.decode()
        module_commit = result.split(':')[0]
    except IndexError:
        raise FetchCommitFailure(
            "fail to parse the commit: gx_hash={}, repo={}, result={}".format(
                gx_hash,
                git_repo,
                result,
            )
        )
    return module_commit


def update_repo_to_go_mod(root_repo_path):
    """Go through the dependencies
    """
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
                dep_gx_path = make_gx_path(dep_gx_hash, dep_name)
                queue.append((dep_gx_hash, dep_gx_path))
        # exclude the root repo
        if repo_path == root_repo_path:
            continue

        # TODO: handle go get here!
        # 1. need `github_repo` + `gx_hash`
        # 2. `git --git-dir=`github_repo`/.git rev-list|xargs git grep `gx_hash`, and find the first commit containing this hash
        # 3. go get `github_repo`@`commit`
        git_repo = package_info['gx']['dvcsimport']
        git_repo_path = make_github_repo_path(git_repo)
        print("handling repo: gx_hash={}, repo_path={}, github_path={}".format(repo_gx_hash, repo_path, git_repo_path))
        download_git_repo(git_repo)
        commit = get_commit_from_git_repo(git_repo, repo_gx_hash)
        print("!@# commit={}".format(commit))


if __name__ == "__main__":
    current_path = os.getcwd()
    update_repo_to_go_mod(current_path)
