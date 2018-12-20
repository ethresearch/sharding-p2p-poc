# -*- coding: utf-8 -*-

from collections import (
    namedtuple,
)
import json
import logging
import os
import re
import subprocess
import sys


least_go_version_str = '1.11'
# ensure go version >= `least_go_version_str`
res_go_version = subprocess.run(["go", "version"], stdout=subprocess.PIPE, encoding='utf-8')
if res_go_version.returncode != 0:
    raise Exception("failed to run `go version`")
res_go_version_str = res_go_version.stdout
ret = re.search(r"go\sversion\sgo([0-9]+\.[0-9]+)", res_go_version_str)
if ret is None:
    raise Exception("failed to parse the version from `go version`")
version_str = ret.groups(0)[0]
if version_str < least_go_version_str:
    raise Exception("go version should be >= {}".format(least_go_version_str))


GOPATH = os.getenv('GOPATH')
TMP_GIT_REPO_PATH = "/tmp/gx-git-repos"
GX_PREFIX = "{}/src/gx/ipfs".format(GOPATH)

RepoVersion = namedtuple("RepoVersion", ["gx_path", "git_repo", "version"])

logger = logging.getLogger("update-gomod")


class DownloadFailure(Exception):
    pass


class GetCommitFailure(Exception):
    pass


class GetVersionFailure(Exception):
    pass


def make_gx_path(gx_hash, repo_name):
    return "{}/{}/{}".format(GX_PREFIX, gx_hash, repo_name)


def extract_gx_hash(gx_path):
    if not gx_path.startswith(GX_PREFIX):
        raise ValueError("gx_path={} should have the preifx {}".format(gx_path, GX_PREFIX))
    # get rid of the prefix f"{GX_PREFIX}/", and split with '/'
    path_list = gx_path[len(GX_PREFIX) + 1:].split('/')
    if len(path_list) < 2:
        raise ValueError("malform gx_path={}".format(gx_path))
    return path_list[0]


def make_git_repo_path(git_repo):
    return "{}/{}".format(TMP_GIT_REPO_PATH, git_repo)


def make_git_path(git_repo_path):
    return "{}/.git".format(git_repo_path)


def make_git_cmd(path, command):
    """Helper function to do git operations avoiding changing directories
    """
    return "git --git-dir={} {}".format(make_git_path(path), command)


def download_git_repo(git_repo):
    repo_path = make_git_repo_path(git_repo)
    if not os.path.exists(TMP_GIT_REPO_PATH):
        try:
            os.mkdir(TMP_GIT_REPO_PATH)
        except FileExistsError:
            pass
    if not os.path.exists(make_git_path(repo_path)):
        git_url = "https://{}".format(git_repo)
        res = subprocess.run(
            "git clone {} {}".format(git_url, repo_path),
            shell=True,
        )
    else:
        res = subprocess.run(
            make_git_cmd(repo_path, "fetch"),
            shell=True,
        )
    if res.returncode != 0:
        raise DownloadFailure("failed to download/update the git repo: repo={}, res={}".format(
            git_repo,
            res,
        ))


def download_repos(git_repos):
    for git_repo in git_repos:
        download_git_repo(git_repo)


def is_version_in_repo(git_repo, sem_ver):
    git_repo_path = make_git_repo_path(git_repo)
    res = subprocess.run(
        "{}".format(
            make_git_cmd(git_repo_path, "tag")
        ),
        encoding='utf-8',
        shell=True,
        stdout=subprocess.PIPE,
    )
    if res.returncode != 0:
        raise GetVersionFailure("failed to access the repo: git_repo={}".format(git_repo))
    version_list = res.stdout.split('\n')
    return sem_ver in version_list


def get_commit_from_repo(git_repo, gx_hash):
    git_repo_path = make_git_repo_path(git_repo)
    res = subprocess.run(
        "{} | xargs {} | tail -n1".format(
            make_git_cmd(git_repo_path, "rev-list --all"),
            make_git_cmd(git_repo_path, "grep {}".format(gx_hash)),
        ),
        encoding="utf-8",
        shell=True,
        stdout=subprocess.PIPE,
    )
    if res.returncode != 0:
        raise GetCommitFailure("failed to fetch the commit: gx_hash={}, repo={}".format(
            gx_hash,
            git_repo,
        ))
    # success
    try:
        result = res.stdout
        module_commit = result.split(':')[0]
    except IndexError:
        raise GetCommitFailure(
            "fail to parse the commit: gx_hash={}, repo={}, result={}".format(
                gx_hash,
                git_repo,
                result,
            )
        )
    # check if `module_commit` is a commit hash, e.g. 5a13bddfa3a06705681ade8e1d4ea85374b6b12e
    try:
        int(module_commit, 16)
    except ValueError:
        return None
    if len(module_commit) != 40:
        return None
    return module_commit


def _remove_url_prefix(url):
    return re.sub(r"\w+://", "", url)


def _dvcsimport_to_git_repo(dvcsimport_path):
    # Assume `devcsimport_path` is in the format of `site/user/repo/package`.
    # Therefore, we only need to get rid of the `package`
    layered_paths = dvcsimport_path.split('/')
    return "/".join(layered_paths[:3])


def get_repo_deps(root_repo_path):
    """Go through the dependencies
    """
    visited_repos = set()
    queue = list()
    deps = []

    queue.append(root_repo_path)
    while len(queue) != 0:
        repo_path = queue.pop()
        if repo_path in visited_repos:
            continue
        package_file_path = "{}/package.json".format(repo_path)
        with open(package_file_path, 'r') as f_read:
            package_info = json.load(f_read)
        # add the deps
        if 'gxDependencies' in package_info:
            for dep_info in package_info['gxDependencies']:
                dep_name = dep_info['name']
                dep_gx_hash = dep_info['hash']
                dep_gx_path = make_gx_path(dep_gx_hash, dep_name)
                queue.append(dep_gx_path)
        # try:
        #     git_repo = _dvcsimport_to_git_repo(_remove_url_prefix(package_info["bugs"]["url"]))
        # except KeyError:
        git_repo = _dvcsimport_to_git_repo(package_info['gx']['dvcsimport'])
        version = None
        if "version" in package_info:
            version = package_info["version"]
        visited_repos.add(repo_path)
        if repo_path != root_repo_path:
            rv = RepoVersion(gx_path=repo_path, git_repo=git_repo, version=version)
            deps.append(rv)
    # filter out non-github deps
    github_deps = [
        repo_version
        for repo_version in deps
        if "github.com" in repo_version.git_repo
    ]
    return github_deps


def parse_version_from_repo_gx_hash(git_repo, raw_version, repo_gx_hash):
    """dep to version or commit?
    """
    # TODO: add checks to ensure gx repos are downloaded(with `gx install`?)
    # try to find the version in the downloaded git repo
    commit = version = None

    if raw_version is None:
        version = None
    else:
        sem_ver = "v{}".format(raw_version)
        if is_version_in_repo(git_repo, sem_ver):
            version = sem_ver
    # try to find the commit in the downloaded git repo
    # XXX: will fail if the git repo does not contain information of gx_hash
    #      the usual case is, the repo is not maintained by IPFS teams
    commit = get_commit_from_repo(git_repo, repo_gx_hash)
    if version is not None and commit is None:
        logger.debug("can only find version of git_repo={} through versions, raw_version={}, repo_gx_hash={}".format(git_repo, raw_version, repo_gx_hash))  # noqa: E501
    return version, commit


def update_repo_to_go_mod(git_repo, version=None, commit=None):
    """Update the repo with either version or commit(priority: version > commit)
    """
    if version is not None:
        print("updating git_repo={} with version={} ...".format(git_repo, version))
        version_indicator = version
    elif commit is not None:
        print("updating git_repo={} with commit={} ...".format(git_repo, commit))
        version_indicator = commit
    else:
        raise ValueError("failed to update {}: version and commit are both None".format(git_repo))
    subprocess.run(
        "GO111MODULE=on go get {}@{}".format(git_repo, version_indicator),
        shell=True,
        stdout=subprocess.PIPE,
    )


def update_repos(repos):
    for repo_version in repos:
        gx_path, git_repo, raw_version = repo_version
        gx_hash = extract_gx_hash(gx_path)
        version, commit = parse_version_from_repo_gx_hash(git_repo, raw_version, gx_hash)
        try:
            update_repo_to_go_mod(git_repo, version, commit)
        except ValueError:
            # FIXME: ignore the failed updates for now
            print("failed to update repo_version={}".format(repo_version))
            logger.debug("failed to update the repo %s", git_repo)


def do_update():
    current_path = os.getcwd()
    deps = get_repo_deps(current_path)
    update_repos(deps)


def do_download():
    current_path = os.getcwd()
    deps = get_repo_deps(current_path)
    git_repo_list = [i.git_repo for i in deps]
    download_repos(git_repo_list)


supported_modes = {
    "update": do_update,
    "download": do_download,
}


if __name__ == "__main__":
    if len(sys.argv) <= 1:
        raise ValueError("Need the argument `mode`")
    mode = sys.argv[1]
    if mode not in supported_modes:
        raise ValueError("Wrong mode={}. Supported: {}".format(
            mode,
            ", ".join([i for i in supported_modes.keys()])
        ))
    supported_modes[mode]()
