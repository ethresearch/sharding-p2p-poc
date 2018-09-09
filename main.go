package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	mrand "math/rand"
	"strconv"
	"strings"

	// gologging "gx/ipfs/QmQvJiADDe7JR4m968MwXobTCCzUqQkP87aRHe29MEBGHV/go-logging"
	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	ic "github.com/libp2p/go-libp2p-crypto"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"

	opentracing "github.com/opentracing/opentracing-go"
	jaeger "github.com/uber/jaeger-client-go"
	jaegerconfig "github.com/uber/jaeger-client-go/config"

	kaddht "github.com/libp2p/go-libp2p-kad-dht"
)

type ShardIDType = int64
type PBInt = int64

const numShards ShardIDType = 100

func makeKey(seed int) (ic.PrivKey, peer.ID, error) {
	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	r := mrand.New(mrand.NewSource(int64(seed)))
	// r := rand.Reader

	// Generate a key pair for this host. We will use it at least
	// to obtain a valid host ID.
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, "", err
	}

	// Get the peer id
	pid, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return nil, "", err
	}
	return priv, pid, nil
}

func makeNode(
	ctx context.Context,
	listenIP string,
	listenPort int,
	randseed int,
	eventNotifier EventNotifier,
	doBootstrapping bool,
	bootstrapPeers []pstore.PeerInfo) (*Node, error) {
	// FIXME: should be set to localhost if we don't want to expose it to outside
	listenAddrString := fmt.Sprintf("/ip4/%v/tcp/%v", listenIP, listenPort)

	priv, _, err := makeKey(randseed)
	if err != nil {
		return nil, err
	}

	basicHost, err := libp2p.New(
		ctx,
		libp2p.Identity(priv),
		libp2p.ListenAddrStrings(listenAddrString),
	)
	if err != nil {
		panic(err)
	}

	// Construct a datastore (needed by the DHT). This is just a simple, in-memory thread-safe datastore.
	dstore := dsync.MutexWrap(ds.NewMapDatastore())

	// Make the DHT
	dht := kaddht.NewDHT(ctx, basicHost, dstore)

	// Make the routed host
	routedHost := rhost.Wrap(basicHost, dht)

	if doBootstrapping {
		// try to connect to the chosen nodes
		bootstrapConnect(ctx, routedHost, bootstrapPeers)

		err = dht.Bootstrap(ctx)
		if err != nil {
			return nil, err
		}
	}

	// Make a host that listens on the given multiaddress
	node := NewNode(ctx, routedHost, int(randseed), eventNotifier)

	return node, nil
}

const (
	defaultListenPort = 10000
	defaultRPCPort    = 13000
	defaultIP         = "127.0.0.1"
)

func main() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	// golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// Parse options from the command line

	seed := flag.Int("seed", 0, "set random seed for id generation")
	listenIP := flag.String(
		"ip",
		defaultIP,
		"ip listened by the process for incoming connections",
	)
	listenPort := flag.Int(
		"port",
		defaultListenPort,
		"port listened by the node for incoming connections",
	)
	rpcIP := flag.String(
		"rpcip",
		defaultIP,
		"ip listened by the RPC server",
	)
	rpcPort := flag.Int("rpcport", defaultRPCPort, "RPC port listened by the RPC server")
	notifierPort := flag.Int(
		"notifierport",
		defulatEventRPCPort,
		"notifier port listened by the event rpc server",
	)
	doBootstrapping := flag.Bool("bootstrap", false, "whether to do bootstrapping or not")
	bootnodesStr := flag.String("bootnodes", "", "multiaddresses of the bootnodes")
	isClient := flag.Bool("client", false, "is RPC client or server")
	flag.Parse()

	rpcAddr := fmt.Sprintf("%v:%v", *rpcIP, *rpcPort)
	notifierAddr := fmt.Sprintf("%v:%v", *rpcIP, *notifierPort)

	if *isClient {
		runClient(rpcAddr, flag.Args())
	} else {
		var bootnodes []pstore.PeerInfo
		if *bootnodesStr == "" {
			bootnodes = []pstore.PeerInfo{}
		} else {
			bootnodes = convertPeers(strings.Split(*bootnodesStr, ","))
		}
		runServer(*listenIP, *listenPort, *seed, *doBootstrapping, bootnodes, rpcAddr, notifierAddr)
	}
}

func runServer(
	listenIP string,
	listenPort int,
	seed int,
	doBootstrapping bool,
	bootnodes []pstore.PeerInfo,
	rpcAddr string,
	notifierAddr string) {
	ctx := context.Background()
	eventNotifier, err := NewRpcEventNotifier(ctx, notifierAddr)
	if err != nil {
		// TODO: don't use eventNotifier if it is not available
		eventNotifier = nil
	}
	node, err := makeNode(
		ctx,
		listenIP,
		listenPort,
		seed,
		eventNotifier,
		doBootstrapping,
		bootnodes,
	)
	if err != nil {
		log.Fatal(err)
	}

	// Set up Opentracing and Jaeger tracer
	cfg := &jaegerconfig.Configuration{
		Sampler: &jaegerconfig.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &jaegerconfig.ReporterConfig{
			LogSpans: true,
		},
	}
	tracerName := fmt.Sprintf("RPC Server@%v", rpcAddr)
	tracer, closer, err := cfg.New(tracerName, jaegerconfig.Logger(jaeger.StdLogger))
	defer closer.Close()
	if err != nil {
		panic(fmt.Sprintf("ERROR: cannot init Jaeger: %v\n", err))
	}
	opentracing.SetGlobalTracer(tracer)
	// End of tracer setup

	runRPCServer(node, rpcAddr)
}

func parseArgsShardIDs(rpcArgs []string) ([]ShardIDType, error) {
	shardIDs := []ShardIDType{}
	for _, shardIDString := range rpcArgs {
		shardID, err := strconv.ParseInt(shardIDString, 10, 64)
		if err != nil {
			return nil, err
		}
		shardIDs = append(shardIDs, shardID)
	}
	return shardIDs, nil
}

func runClient(rpcAddr string, cliArgs []string) {
	if len(cliArgs) <= 0 {
		log.Fatalf("Client: wrong args")
		return
	}
	rpcCmd := cliArgs[0]
	rpcArgs := cliArgs[1:]
	if rpcCmd == "addpeer" {
		if len(rpcArgs) != 3 {
			log.Fatalf("Client: usage: addpeer ip port seed")
		}
		targetIP := rpcArgs[0]
		targetPort, err := strconv.Atoi(rpcArgs[1])
		targetSeed, err := strconv.Atoi(rpcArgs[2])
		if err != nil {
			panic(err)
		}
		callRPCAddPeer(rpcAddr, targetIP, targetPort, targetSeed)
	} else if rpcCmd == "subshard" {
		if len(rpcArgs) == 0 {
			log.Fatalf("Client: usage: subshard shard0 shard1 ...")
		}
		shardIDs, err := parseArgsShardIDs(rpcArgs)
		if err != nil {
			panic(err)
		}
		callRPCSubscribeShard(rpcAddr, shardIDs)
	} else if rpcCmd == "unsubshard" {
		if len(rpcArgs) == 0 {
			log.Fatalf("Client: usage: unsubshard shard0 shard1 ...")
		}
		shardIDs, err := parseArgsShardIDs(rpcArgs)
		if err != nil {
			panic(err)
		}
		callRPCUnsubscribeShard(rpcAddr, shardIDs)
	} else if rpcCmd == "getsubshard" {
		callRPCGetSubscribedShard(rpcAddr)
	} else if rpcCmd == "broadcastcollation" {
		if len(rpcArgs) != 4 {
			log.Fatalf(
				"Client: usage: broadcastcollation shardID, numCollations, collationSize, timeInMs",
			)
		}
		shardID, err := strconv.ParseInt(rpcArgs[0], 10, 64)
		if err != nil {
			log.Fatalf("wrong shardID: %v", rpcArgs)
		}
		numCollations, err := strconv.Atoi(rpcArgs[1])
		if err != nil {
			log.Fatalf("wrong numCollations: %v", rpcArgs)
		}
		collationSize, err := strconv.Atoi(rpcArgs[2])
		if err != nil {
			log.Fatalf("wrong collationSize: %v", rpcArgs)
		}
		timeInMs, err := strconv.Atoi(rpcArgs[3])
		if err != nil {
			log.Fatalf("wrong timeInMs: %v", rpcArgs)
		}
		callRPCBroadcastCollation(
			rpcAddr,
			shardID,
			numCollations,
			collationSize,
			timeInMs,
		)
	} else if rpcCmd == "stop" {
		callRPCStopServer(rpcAddr)
	} else if rpcCmd == "getconn" {
		callRPCGetConnection(rpcAddr)
	} else if rpcCmd == "getpeer" {
		var topic string
		if len(rpcArgs) > 1 {
			log.Fatalf(
				"usage: getpeer (topic)",
			)
		} else if len(rpcArgs) == 1 {
			topic = rpcArgs[0]
		} else {
			topic = ""
		}
		callRPCGetPeer(rpcAddr, topic)
	} else if rpcCmd == "syncshardpeer" {
		if len(rpcArgs) < 2 {
			log.Fatalf(
				"usage: syncshardpeer peerID shardID0 shardID1 ...",
			)
		}
		peerID, err := stringToPeerID(rpcArgs[0])
		if err != nil {
			log.Fatalf("wrong peerID format: %v", err)
		}
		shardIDs, err := parseArgsShardIDs(rpcArgs[1:])
		if err != nil {
			log.Fatalf("wrong shardIDs format: %v", err)
		}
		callRPCSyncShardPeer(rpcAddr, peerID, shardIDs)
	} else if rpcCmd == "synccollation" {
		if len(rpcArgs) < 3 {
			log.Fatalf(
				"usage: synccollation peerID shardID period (collationHash)",
			)
		}
		peerID, err := stringToPeerID(rpcArgs[0])
		if err != nil {
			log.Fatalf("wrong peerID: %v, reason: %v", rpcArgs, err)
		}
		shardID, err := strconv.ParseInt(rpcArgs[1], 10, 64)
		if err != nil {
			log.Fatalf("wrong shardID: %v, reason: %v", rpcArgs, err)
		}
		period, err := strconv.Atoi(rpcArgs[2])
		if err != nil {
			log.Fatalf("wrong period: %v, reason: %v", rpcArgs, err)
		}
		collationHash := ""
		if len(rpcArgs) == 4 {
			collationHash = rpcArgs[3]
		}
		callRPCSyncCollation(rpcAddr, peerID, shardID, period, collationHash)
	} else {
		log.Fatalf("Client: wrong cmd '%v'", rpcCmd)
	}
}
