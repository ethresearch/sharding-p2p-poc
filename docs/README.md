# sharding-p2p-poc document
This is the document for [ethresearch/sharding-p2p-poc](https://github.com/ethresearch/sharding-p2p-poc)

## Ethereum sharding P2P requirement
What does a node in the sharding p2p network need?
- A node should be able to subscribe to multiple shards simultaneously
- A node should be able to jump(i.e., unsubscribe A and then subscribe B) between shards with low latency

## Design
In the current stage, we are building a gossip layer on top of a [PubSub](https://en.wikipedia.org/wiki/Publish%E2%80%93subscribe_pattern) system. The basic concept is that every shard is one-to-one mapped to a topic in the PubSub and every node will subscribe to the topics they are interested in.
**NOTE**: To avoid adding too many details in this documentation, please refer to PubSub documents for basic understanding about what a topic is and how publishing/subscribing works.
- If a node wants to publish shard-specific messages, it **publishes** them to the topic corresponding to that shard.
    - E.g. We can agree on using the topic "Shard_9_collation" as the topic for the collation messages in shard 9. In this manner, collations in shard 9 are published to that topic, and nodes subscribing that topic will get the published collations.
- A node interested in a shard **subscribes** to the topic corresponding to that shard, in order to receive messages regarding the shard.

### Topic allocation
Some messages are required by all of the nodes, e.g., beacon chain block headers. We broadcast those messages in the corresponding topics, for convenience we call them "global topics". While some messages are shard-specific and only required by some nodes, we call the corresponding topics "local topics".

- Global topics
    - Shard subscription preferences
        - `shardIDs`: `SHARD_COUNT` bits
    - Beacon chain messages
- Local topics
    - Local shard transactions
    - Local shard headers
    - Local shard collations

### Peer routing
The "peer routing" here refers to how we translate a `peerID` to its corresponding IP address and port. We use Kademlia DHT to do it currently.

### Peer discovery
#### General peer discovery
A list of hardcoded nodes serve as the initial contact nodes, and each node performs bootstrapping process with DHT through the contact nodes.
#### Shard peer discovery
A global topic is used for every node to broadcast its "Shard Preference". A "Shard Preference" specifies which shards a node is currently listening to. With this topic, a node is able to discover peers in specific shard through looking up the table of "Shard Preference".

## Implementation
### Goal
- To see if our current design as stated [above](#design) actually meets the need of [ethereum sharding](https://ethresear.ch/c/sharding) p2p layer.
- To see if the [P2P stack](#p2p-stack) is sufficient for our usage.

### P2P Stack
Currently, we use [go-libp2p](https://github.com/libp2p/go-libp2p) as the p2p stack.
- [PubSub](https://github.com/libp2p/specs/tree/master/pubsub/gossipsub): we use [gossipsub](https://github.com/libp2p/go-libp2p-pubsub/blob/master/gossipsub.go) right now. 
- DHT: [go-libp2p-kad-dht](https://github.com/libp2p/go-libp2p-kad-dht)

### Functionalities
- Join the sharding network
- Broadcast/receive messages in the global topics
- Subscribe to multiple shards
- Broadcast/receive messages in the shard which the node has subscribed to
- Request for collations from the node's peers
- Request for peer infos from the node's peers
    - E.g., request for info of peers who are also interested in shard 3 from my peer A

## Reminder
Because the idea might keep changing, please check out https://github.com/ethresearch/sharding-p2p-poc/issues for the latest issues and idea.
