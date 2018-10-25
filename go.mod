module github.com/ethresearch/sharding-p2p-poc

require (
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/btcsuite/btcd v0.0.0-20181013004428-67e573d211ac // indirect
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd // indirect
	github.com/coreos/go-semver v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fd/go-nat v1.0.0 // indirect
	github.com/go-check/check v0.0.0-20180628173108-788fd7840127 // indirect
	github.com/gogo/protobuf v1.1.1 // indirect
	github.com/golang/protobuf v1.2.0
	github.com/google/uuid v1.0.0 // indirect
	github.com/gorilla/websocket v1.4.0 // indirect
	github.com/gxed/GoEndian v0.0.0-20160916112711-0f5c6873267e // indirect
	github.com/gxed/eventfd v0.0.0-20160916113412-80a92cca79a8 // indirect
	github.com/gxed/hashland v0.0.0-20180221191214-d9f6b97f8db2 // indirect
	github.com/hashicorp/golang-lru v0.5.0 // indirect
	github.com/ipfs/go-cid v0.9.0 // indirect
	github.com/ipfs/go-datastore v3.2.0+incompatible
	github.com/ipfs/go-detect-race v1.0.1 // indirect
	github.com/ipfs/go-ipfs-util v1.2.8 // indirect
	github.com/ipfs/go-log v1.5.7
	github.com/ipfs/go-todocounter v1.0.1 // indirect
	github.com/jbenet/go-cienv v0.0.0-20150120210510-1bb1476777ec // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jbenet/go-randbuf v0.0.0-20160322125720-674640a50e6a // indirect
	github.com/jbenet/go-temp-err-catcher v0.0.0-20150120210811-aac704a3f4f2 // indirect
	github.com/jbenet/goprocess v0.0.0-20160826012719-b497e2f366b8 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/libp2p/go-addr-util v2.0.6+incompatible // indirect
	github.com/libp2p/go-buffer-pool v0.1.1 // indirect
	github.com/libp2p/go-conn-security v0.1.14 // indirect
	github.com/libp2p/go-conn-security-multistream v0.1.14 // indirect
	github.com/libp2p/go-floodsub v0.9.36
	github.com/libp2p/go-flow-metrics v0.2.0 // indirect
	github.com/libp2p/go-libp2p v6.0.19+incompatible
	github.com/libp2p/go-libp2p-blankhost v0.3.14 // indirect
	github.com/libp2p/go-libp2p-circuit v2.2.8+incompatible // indirect
	github.com/libp2p/go-libp2p-crypto v2.0.1+incompatible
	github.com/libp2p/go-libp2p-host v3.0.14+incompatible
	github.com/libp2p/go-libp2p-interface-connmgr v0.0.20 // indirect
	github.com/libp2p/go-libp2p-interface-pnet v3.0.0+incompatible // indirect
	github.com/libp2p/go-libp2p-kad-dht v4.4.8+incompatible
	github.com/libp2p/go-libp2p-kbucket v2.2.11+incompatible // indirect
	github.com/libp2p/go-libp2p-loggables v1.1.23 // indirect
	github.com/libp2p/go-libp2p-metrics v2.1.7+incompatible // indirect
	github.com/libp2p/go-libp2p-nat v0.8.7 // indirect
	github.com/libp2p/go-libp2p-net v3.0.14+incompatible
	github.com/libp2p/go-libp2p-netutil v0.4.11 // indirect
	github.com/libp2p/go-libp2p-peer v2.4.0+incompatible
	github.com/libp2p/go-libp2p-peerstore v2.0.4+incompatible
	github.com/libp2p/go-libp2p-protocol v1.0.0
	github.com/libp2p/go-libp2p-record v4.1.7+incompatible // indirect
	github.com/libp2p/go-libp2p-routing v2.6.5+incompatible // indirect
	github.com/libp2p/go-libp2p-secio v2.0.16+incompatible // indirect
	github.com/libp2p/go-libp2p-swarm v3.0.20+incompatible // indirect
	github.com/libp2p/go-libp2p-transport v3.0.14+incompatible // indirect
	github.com/libp2p/go-libp2p-transport-upgrader v0.1.15 // indirect
	github.com/libp2p/go-maddr-filter v1.1.9 // indirect
	github.com/libp2p/go-mplex v0.2.30 // indirect
	github.com/libp2p/go-msgio v0.0.6 // indirect
	github.com/libp2p/go-reuseport v0.1.18 // indirect
	github.com/libp2p/go-reuseport-transport v0.1.10 // indirect
	github.com/libp2p/go-sockaddr v1.0.3 // indirect
	github.com/libp2p/go-stream-muxer v3.0.1+incompatible // indirect
	github.com/libp2p/go-tcp-transport v2.0.15+incompatible // indirect
	github.com/libp2p/go-testutil v1.2.9 // indirect
	github.com/libp2p/go-ws-transport v2.0.14+incompatible // indirect
	github.com/mattn/go-colorable v0.0.9 // indirect
	github.com/mattn/go-isatty v0.0.4 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1 // indirect
	github.com/minio/sha256-simd v0.0.0-20181005183134-51976451ce19 // indirect
	github.com/mr-tron/base58 v1.1.0
	github.com/multiformats/go-multiaddr v1.3.0
	github.com/multiformats/go-multiaddr-dns v0.2.4 // indirect
	github.com/multiformats/go-multiaddr-net v1.6.3 // indirect
	github.com/multiformats/go-multibase v0.3.0 // indirect
	github.com/multiformats/go-multicodec v0.1.6
	github.com/multiformats/go-multihash v1.0.8 // indirect
	github.com/multiformats/go-multistream v0.3.9 // indirect
	github.com/onsi/gomega v1.4.2 // indirect
	github.com/opentracing/opentracing-go v1.0.2
	github.com/pkg/errors v0.8.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/spaolacci/murmur3 v0.0.0-20180118202830-f09979ecbc72 // indirect
	github.com/stretchr/testify v1.2.2 // indirect
	github.com/uber-go/atomic v1.3.2 // indirect
	github.com/uber/jaeger-client-go v2.15.0+incompatible
	github.com/uber/jaeger-lib v1.5.0 // indirect
	github.com/whyrusleeping/base32 v0.0.0-20170828182744-c30ac30633cc // indirect
	github.com/whyrusleeping/go-keyspace v0.0.0-20160322163242-5b898ac5add1 // indirect
	github.com/whyrusleeping/go-logging v0.0.0-20170515211332-0457bb6b88fc // indirect
	github.com/whyrusleeping/go-notifier v0.0.0-20170827234753-097c5d47330f // indirect
	github.com/whyrusleeping/go-smux-multiplex v3.0.16+incompatible // indirect
	github.com/whyrusleeping/go-smux-multistream v2.0.2+incompatible // indirect
	github.com/whyrusleeping/go-smux-yamux v2.0.6+incompatible // indirect
	github.com/whyrusleeping/mafmt v0.0.0-20180627004827-1dc32401ee9f // indirect
	github.com/whyrusleeping/multiaddr-filter v0.0.0-20160516205228-e903e4adabd7 // indirect
	github.com/whyrusleeping/timecache v0.0.0-20160911033111-cfcb2f1abfee // indirect
	github.com/whyrusleeping/yamux v1.1.2 // indirect
	go.uber.org/atomic v1.3.2 // indirect
	golang.org/x/crypto v0.0.0-20181015023909-0c41d7ab0a0e
	golang.org/x/net v0.0.0-20181023162649-9b4f9f5ad519
	google.golang.org/grpc v1.16.0
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)
