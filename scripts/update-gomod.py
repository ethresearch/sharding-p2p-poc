# -*- coding: utf-8 -*-

import json
import logging
import os
import re
import subprocess


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


logger = logging.getLogger("update-gomod")


class DownloadFailure(Exception):
    pass


class GetCommitFailure(Exception):
    pass


class GetVersionFailure(Exception):
    pass


class GxPath:
    gx_hash = None
    repo_name = None

    def __init__(self, gx_hash, repo_name):
        self.gx_hash = gx_hash
        self.repo_name = repo_name

    @property
    def gx_path(self):
        return "{}/src/gx/ipfs/{}/{}".format(GOPATH, self.gx_hash, self.repo_name)

    @classmethod
    def from_gx_path(cls, gx_path):
        path_list = gx_path.split('/')
        if len(path_list) < 6:
            raise ValueError("malform gx_path={}".format(gx_path))
        return cls(path_list[4], path_list[5])


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
        res = subprocess.run("git clone {} {}".format(git_url, repo_path), shell=True, encoding='utf-8')
    else:
        res = subprocess.run(
            make_git_cmd(repo_path, "pull origin master"),
            shell=True,
            encoding='utf-8',
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
        shell=True,
        encoding='utf-8',
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
        shell=True,
        stdout=subprocess.PIPE,
        encoding="utf-8",
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
    visited_repos = {}  # repo_gx_path -> git_repo
    queue = list()

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
            deps = package_info['gxDependencies']
            for dep_info in deps:
                dep_name = dep_info['name']
                dep_gx_hash = dep_info['hash']
                dep_gx_path = GxPath(dep_gx_hash, dep_name).gx_path
                queue.append(dep_gx_path)
        try:
            git_repo = _dvcsimport_to_git_repo(_remove_url_prefix(package_info["bugs"]["url"]))
        except KeyError:
            git_repo = _dvcsimport_to_git_repo(package_info['gx']['dvcsimport'])
        version = None
        if "version" in package_info:
            version = package_info["version"]
        visited_repos[repo_path] = (git_repo, version)
    # exclude the root repo itself
    del visited_repos[root_repo_path]
    return visited_repos


def parse_version_from_repo_gx_hash(git_repo, raw_version, repo_gx_hash):
    """dep to version or commit?
    """
    # TODO: add checks to ensure gx repos are downloaded(with `gx install`?)
    # try to find the version in the downloaded git repo
    commit = version = None
    sem_ver = "v{}".format(raw_version)
    if is_version_in_repo(git_repo, sem_ver):
        version = sem_ver
    else:
        # try to find the commit in the downloaded git repo
        # XXX: will fail if the git repo does not contain information of gx_hash
        #      the usual case is, the repo is not maintained by IPFS teams
        commit = get_commit_from_repo(git_repo, repo_gx_hash)
    return version, commit


def update_repo_to_go_mod(git_repo, version=None, commit=None):
    if version is not None:
        print("updating git_repo={} with version={} ...".format(git_repo, version), end="")
        version_indicator = version
    elif commit is not None:
        print("updating git_repo={} with commit={} ...".format(git_repo, commit), end="")
        version_indicator = commit
    else:
        raise ValueError("failed to update {}: version and commit are both None".format(git_repo))
    subprocess.run(
        "GO111MODULE=on go get {}@{}".format(git_repo, version_indicator),
        shell=True,
        stdout=subprocess.PIPE,
    )
    print("finished")


def update_repos(map_repos):
    for repo_gx_path, repo_version in map_repos.items():
        git_repo, raw_version = repo_version
        gx_obj = GxPath.from_gx_path(repo_gx_path)
        gx_hash = gx_obj.gx_hash
        version, commit = parse_version_from_repo_gx_hash(git_repo, raw_version, gx_hash)
        try:
            update_repo_to_go_mod(git_repo, version, commit)
        except ValueError:
            # FIXME: ignore the failed updates for now
            logger.debug("failed to update the repo %s", git_repo)


if __name__ == "__main__":
    current_path = os.getcwd()
    import sys
    deps_map = get_repo_deps(current_path)
    # filter out non-github deps
    github_deps = {
        repo_gx_path: repo_version
        for repo_gx_path, repo_version in deps_map.items()
        if "github.com" in repo_version[0]
    }
    update_repos(github_deps)
