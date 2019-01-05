# LOG_ADD_PEER = 'rpcserver:AddPeer: ip=192.168.11.69, port=10001, seed=1'
LOG_ADD_PEER_FMT = 'rpcserver:AddPeer: ip={}, port={}, seed={}'
LOG_ADD_PEER_FINISHED = 'rpcserver:AddPeer: finished'
# LOG_BROADCAST_COLLATION = 'rpcserver:BroadcastCollation: broadcasting: shardID=0, numCollations=1, sizeInBytes=900, timeInMs=123'  # noqa: E501
LOG_BROADCAST_COLLATION_FMT = 'rpcserver:BroadcastCollation: broadcasting: shardID={}, numCollations={}, sizeInBytes={}, timeInMs={}'  # noqa: E501
LOG_BROADCAST_COLLATION_FINISHED = 'rpcserver:BroadcastCollation: finished'
LOG_RECEIVE_MSG = 'Validating the received message'
# LOG_SUBSCRIBE_SHARD = 'rpcserver:SubscribeShard: shardIDs=[0 1 2]'
LOG_SUBSCRIBE_SHARD_FMT = 'rpcserver:SubscribeShard: shardIDs={}'
LOG_SUBSCRIBE_SHARD_FINISHED = 'rpcserver:SubscribeShard: finished'
# LOG_UNSUBSCRIBE_SHARD = 'rpcserver:UnsubscribeShard: shardIDs=[0]'
LOG_UNSUBSCRIBE_SHARD_FMT = 'rpcserver:UnsubscribeShard: shardIDs={}'
LOG_UNSUBSCRIBE_SHARD_FINISHED = 'rpcserver:UnsubscribeShard: finished'
