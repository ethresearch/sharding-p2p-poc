package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	mrand "math/rand"
	"strconv"
	"strings"
	"time"

	ds "github.com/ipfs/go-datastore"
	dsync "github.com/ipfs/go-datastore/sync"
	golog "github.com/ipfs/go-log"
	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p-crypto"
	ic "github.com/libp2p/go-libp2p-crypto"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	rhost "github.com/libp2p/go-libp2p/p2p/host/routed"
	ma "github.com/multiformats/go-multiaddr"
	gologging "github.com/whyrusleeping/go-logging"
)

// import "C"

type ShardIDType = int64

const numShards ShardIDType = 100

func makeKey(seed int64) (ic.PrivKey, peer.ID, error) {
	// If the seed is zero, use real cryptographic randomness. Otherwise, use a
	// deterministic randomness source to make generated keys stay the same
	// across multiple runs
	r := mrand.New(mrand.NewSource(seed))
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

// makeNode creates a LibP2P host with a random peer ID listening on the
// given multiaddress. It will use secio if secio is true.
func makeNode(
	ctx context.Context,
	listenPort int,
	randseed int64,
	bootstrapPeers []pstore.PeerInfo) (*Node, error) {

	listenAddrString := fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)

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
	dht := dht.NewDHT(ctx, basicHost, dstore)

	// Make the routed host
	routedHost := rhost.Wrap(basicHost, dht)

	// try to connect to the chosen nodes
	bootstrapConnect(ctx, routedHost, bootstrapPeers)

	// Bootstrap the host
	err = dht.Bootstrap(ctx)
	if err != nil {
		return nil, err
	}

	// Make a host that listens on the given multiaddress
	node := NewNode(ctx, routedHost, int(randseed))

	log.Printf("I am %s\n", node.GetFullAddr())

	return node, nil
}

const (
	portBase    = 10000
	rpcPortBase = 13000
)

func main() {
	// LibP2P code uses golog to log messages. They log with different
	// string IDs (i.e. "swarm"). We can control the verbosity level for
	// all loggers with:
	golog.SetAllLoggers(gologging.INFO) // Change to DEBUG for extra info

	// Parse options from the command line

	seed := flag.Int64("seed", 0, "set random seed for id generation")
	sendCollationOption := flag.String("send", "", "send collations")
	peerSeed := flag.Int64("find", -1, "use dht to find a certain peer with the given peerSeed")
	isClient := flag.Bool("client", false, "is RPC client or server")
	flag.Parse()
	// log.Print(*isClient)
	// log.Print(flag.Args())
	// log.Print(*seed)
	// log.Print(reflection.TypeOf())
	// return

	listenPort := portBase + int32(*seed)
	rpcPort := rpcPortBase + int32(*seed)
	rpcAddr := fmt.Sprintf("127.0.0.1:%v", rpcPort)

	ctx := context.Background()
	node, err := makeNode(ctx, int(listenPort), *seed, []pstore.PeerInfo{})

	if err != nil {
		log.Fatal(err)
	}

	if *isClient {
		if len(flag.Args()) <= 0 {
			log.Fatalf("Client: wrong args")
			return
		}
		rpcCmd := flag.Args()[0]
		rpcArgs := flag.Args()[1:]
		if rpcCmd == "addpeer" {
			if len(rpcArgs) != 2 {
				log.Fatalf("Client: addpeer: wrong args")
			}
			targetIP := rpcArgs[0]
			targetSeed, err := strconv.ParseInt(rpcArgs[1], 10, 64)
			if err != nil {
				panic(err)
			}
			targetPort := portBase + targetSeed
			callRPCAddPeer(rpcAddr, targetIP, int(targetPort), targetSeed)
			return
		} else if rpcCmd == "subshard" {
			if len(rpcArgs) == 0 {
				log.Fatalf("Client: subshard: wrong args")
			}
			shardIDs := []ShardIDType{}
			for _, shardIDString := range rpcArgs {
				shardID, err := strconv.ParseInt(shardIDString, 10, 64)
				if err != nil {
					panic(err)
				}
				shardIDs = append(shardIDs, shardID)
			}
			callRPCSubscribeShard(rpcAddr, shardIDs)
			return
		} else if rpcCmd == "unsubshard" {
			if len(rpcArgs) == 0 {
				log.Fatalf("Client: unsubshard: wrong args")
			}
			shardIDs := []ShardIDType{}
			for _, shardIDString := range rpcArgs {
				shardID, err := strconv.ParseInt(shardIDString, 10, 64)
				if err != nil {
					panic(err)
				}
				shardIDs = append(shardIDs, shardID)
			}
			callRPCUnsubscribeShard(rpcAddr, shardIDs)
			return
		} else if rpcCmd == "getsubshard" {
			callRPCGetSubscribedShard(rpcAddr)
			return
		} else if rpcCmd == "broadcastcollation" {
			if len(rpcArgs) != 4 {
				log.Fatalf("Client: broadcastcollation: wrong args")
			}
			shardID, err := strconv.ParseInt(rpcArgs[0], 10, 64)
			if err != nil {
				log.Fatalf("wrong send shards %v", rpcArgs)
			}
			numCollations, err := strconv.Atoi(rpcArgs[1])
			if err != nil {
				log.Fatalf("wrong send shards %v", rpcArgs)
			}
			if err != nil {
				log.Fatalf("wrong send shards %v", rpcArgs)
			}
			collationSize, err := strconv.Atoi(rpcArgs[2])
			timeInMs, err := strconv.Atoi(rpcArgs[3])
			if err != nil {
				log.Fatalf("wrong send shards %v", rpcArgs)
			}
			callRPCBroadcastCollation(
				rpcAddr,
				shardID,
				numCollations,
				collationSize,
				timeInMs,
			)
			return
		}
	}

	if *peerSeed != -1 {
		_, peerID, err := makeKey(*peerSeed)
		if err != nil {
			log.Printf("[ERROR] error parsing peerID in multihash %v", peerID)
		}
		pi := pstore.PeerInfo{
			ID:    peerID,
			Addrs: []ma.Multiaddr{},
		}
		t := time.Now()
		err = node.Connect(context.Background(), pi)
		if err != nil {
			log.Printf("Failed to connect %v", pi.ID)
		} else {
			log.Printf("node.Connect takes %v", time.Since(t))
		}
	}

	time.Sleep(time.Millisecond * 500)

	if *sendCollationOption != "" {
		options := strings.Split(*sendCollationOption, ",")
		if len(options) != 3 {
			log.Fatalf("wrong collation option %v", *sendCollationOption)
		}
		numSendShards, err := strconv.Atoi(options[0])
		if err != nil {
			log.Fatalf("wrong send shards %v", options)
		}
		numCollations, err := strconv.Atoi(options[1])
		if err != nil {
			log.Fatalf("wrong send shards %v", options)
		}
		timeInMs, err := strconv.Atoi(options[2])
		if err != nil {
			log.Fatalf("wrong send shards %v", options)
		}

		blobSize := int(math.Pow(2, 20) - 100) // roughly 1 MB

		for i := 0; i < numSendShards; i++ {
			time.Sleep(time.Millisecond * time.Duration(timeInMs))
			go func(shardID int) {
				for j := 0; j < numCollations; j++ {
					time.Sleep(time.Millisecond * time.Duration(timeInMs*numSendShards))
					node.SendCollation(
						ShardIDType(shardID),
						int64(j),
						string(make([]byte, blobSize)),
					)
					// TODO: control the speed of sending collations
				}
			}(i)
		}
	}
	log.Printf("%v: listening for connections", node.Name())
	// go func() {
	// 	time.Sleep(time.Second * 1)
	// }()
	// TODO: add "for: n.PublishListeningShards()" back
	runRPCServer(node, rpcAddr)

	// for {
	// 	log.Println(node.Name())
	// 	time.Sleep(time.Millisecond * 1000)
	// }
}
