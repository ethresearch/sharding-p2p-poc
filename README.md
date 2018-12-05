# sharding-p2p-poc

[![Build Status](https://travis-ci.org/ethresearch/sharding-p2p-poc.svg?branch=master)](https://travis-ci.org/ethresearch/sharding-p2p-poc) [![](https://img.shields.io/badge/gitter-sharding--p2p--poc-ed1965.svg)](https://gitter.im/sharding-p2p-testing/Lobby)

> This is a proof of concept of Ethereum sharding peer to peer layer with PubSub in libp2p, based on the idea from the [slide](
https://docs.google.com/presentation/d/11a0jibNz0fyUnsWt9fa2MmghHANdHAAABa0TV7EieHs/).

For more information, please check out the [document](https://github.com/ethresearch/sharding-p2p-poc/blob/master/docs/README.md).

## Getting Started

### Prerequisites

- `go` with `1.11.x` or above is installed and properly configured on your machine.
    - `$GOPATH` variable has been [specified](https://github.com/golang/go/wiki/GOPATH).
    - `$GOPATH/bin/` is part of your `$PATH`.
- If you modify `*.proto` files, you will also need `protoc` to compile them to `*.pb.go`
    - e.g., [Install protoc on mac](https://medium.com/@erika_dike/installing-the-protobuf-compiler-on-a-mac-a0d397af46b8)

### Build

```bash
$ git clone https://github.com/ethresearch/sharding-p2p-poc.git
$ cd sharding-p2p-poc
$ make build
```

### Testing

```bash
$ go test -v
```

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
  -verbose
        set the log level to DEBUG, i.e., print all messages
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

Since they are still subject to changes, please check out the [documentations](https://notes.ethereum.org/s/HJdvvyTmm) for further reference to the RPC methods. You can also check out the examples [here](https://github.com/ethresearch/sharding-p2p-poc/tree/master/cli-example).

## Using Docker

```
make docker-build
```

### Setting up tracer

See [tracer](./docs/tracer.md) in the document.

## Contributing

Issues, PRs, or any kind of help are definitely welcomed.
