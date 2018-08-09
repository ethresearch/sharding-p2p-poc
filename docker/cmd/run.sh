#!/bin/bash

sleep 1

export BOOTSTRAP_IP=$(getent hosts bootstrap | awk '{ print $1 }')

/main \
    -seed=$RANDOM \
    -port=5566 \
    -rpcport=7788 \
    -bootstrap \
    -bootnodes=/ip4/$BOOTSTRAP_IP/tcp/5566/ipfs/QmS5QmciTXXnCUCyxud5eWFenUMAmvAWSDa1c7dvdXRMZ7