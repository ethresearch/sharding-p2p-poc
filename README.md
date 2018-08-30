# sharding-p2p-poc

[![Build Status](https://travis-ci.org/ethresearch/sharding-p2p-poc.svg?branch=master)](https://travis-ci.org/ethresearch/sharding-p2p-poc) [![](https://img.shields.io/badge/gitter-sharding--p2p--poc-ed1965.svg)](https://gitter.im/ethresearch/sharding-p2p-poc)

> This is the poc of ethereum sharding p2p layer with pubsub in libp2p, based on the idea from the [slide](
https://docs.google.com/presentation/d/11a0jibNz0fyUnsWt9fa2MmghHANdHAAABa0TV7EieHs/edit?usp=sharing).

For more information, please check out the [document](https://github.com/ethresearch/sharding-p2p-poc/blob/master/docs/README.md).

## Install

### Prequisites
- `go` is installed and properly configured on your machine.  
    - `$GOPATH` variable has been [specified](https://github.com/golang/go/wiki/GOPATH).
    - `$GOPATH/bin/` is part of your `$PATH`.
- `gx` and `gx-go` are [installed](https://github.com/whyrusleeping/gx) on your machine. 
- If you modify `*.proto` files, you will also need `protoc` to compile them to `*.pb.go`
    - e.g., [Install protoc on mac](https://medium.com/@erika_dike/installing-the-protobuf-compiler-on-a-mac-a0d397af46b8)

### Build
```bash
$ go get github.com/ethresearch/sharding-p2p-poc/...
$ cd $GOPATH/src/github.com/ethresearch/sharding-p2p-poc
$ make deps
$ go build
```

### Testing
```bash
$ go test -v
```

### Sidenote: version control of packages
We use [gx](https://github.com/whyrusleeping/gx) as the package management tools for libp2p packages. Running the command `make gx-rw` will rewrite the import statements `github.com/aaa/bbb/ccc` of those packages to `gx/ipfs/xxx/ddd/ccc` in our repository, which makes us use of the specific version of the packages stored on ipfs.
Command `make gx-uw` will undo the effect of `make gx-rw`.

**NOTE**: (This is a contingency plan) If the packages are `gx-rw`'ed, please make sure it's followed by one more command `make partial-gx-uw` to unwrite certain packages. We apologize for the inconvenience.

## Usage
```bash
$ ./sharding-p2p-poc --help
Usage of ./sharding-p2p-poc:
  -bootnodes string
    	multiaddresses of the bootnodes
  -bootstrap
    	whether to do bootstrapping or not
  -client
    	is RPC client or server
  -port int
    	port listened by the node for incoming connections (default 10000)
  -rpcport int
    	rpc port listened by the rpc server (default 13000)
  -seed int
    	set random seed for id generation
```
**Note**: `-bootstrap` controls whether to spin up a bootstrap routine, which periodically queries its peers for new peers. There will be no effect if you feed `-bootnodes` without specifying the flag `-bootstrap`.
### Example

#### Spin up a node with a running RPC server

```bash
$ ./sharding-p2p-poc -seed=1 -port=10001 -rpcport=13001 -bootstrap -bootnodes=/ip4/127.0.0.1/tcp/5566/ipfs/QmS5QmciTXXnCUCyxud5eWFenUMAmvAWSDa1c7dvdXRMZ7,/ip4/127.0.0.1/tcp/10001/ipfs/QmexAnfpHrhMmAC5UNQVS8iBuUUgDrMbMY17Cck2gKrqeX
```
This command spins up a node with seed `0`, listening to new connections at port `10001`, listening to RPC requests at port `13001`, turning on bootstrapping mode with the flag`-bootstrap`, with the bootstrapping nodes `/ip4/127.0.0.1/tcp/5566/ipfs/QmS5QmciTXXnCUCyxud5eWFenUMAmvAWSDa1c7dvdXRMZ7` and `/ip4/127.0.0.1/tcp/10001/ipfs/QmexAnfpHrhMmAC5UNQVS8iBuUUgDrMbMY17Cck2gKrqeX`.

**Note**: `/ip4/127.0.0.1/tcp/10001/ipfs/QmexAnfpHrhMmAC5UNQVS8iBuUUgDrMbMY17Cck2gKrqeX` is the format of ipfs address, which is in the form of `/ip4/{ip}/tcp/{port}/{peerID}`.

#### Command Line Interface
To give commands to the node you have spun up, you can run the command in this format:

```bash
$ ./sharding-p2p-poc -client -rpcport={rpcport} {method_name} {params}
```
The flag `-rpcport` is important since it is used to specify which node you are going to communicate with. If you don't specify it, the current default `rpcport` is `13000`

Since they are still subject to changes, please check out the [documentations](https://notes.ethereum.org/-0j1baijSJWwm5qn2GKBIA?both) for further reference to the RPC methods. You can also check out the examples [here](https://github.com/ethresearch/sharding-p2p-poc/tree/master/cli-example).

## Docker dev environment playground

This require `docker` and `docker-compose`

### Building the image

```
# This builds a image that with all depending go packages.
make build-dev
```

```
# When done developing, this builds a binary file `main`.
make run-dev
```

Run many instances

```
# This runs a private net with 1 bootstrap node and 5 other nodes.
make run-many-dev
# Stop and remove unused container.
make down-dev
```

### Testing

```
make test-dev
```

## Contributing

Issues, PRs, or any kind of help are definitely welcomed.

**NOTE**: Please make sure you have run `gx-go uw` before making a PR. Very much appreciated! :)
