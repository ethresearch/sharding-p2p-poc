# Testing Methodology

## Overview
The network is segmented into X number of shards. Every ~10 minutes, validators are randomly assigned to a shard, so the stress point is observing and testing the ability of validators to subscribe to new topics and send/receive messages pertaining to this new topic in an adequate amount of time.

Please reference [this document](https://notes.ethereum.org/s/ByYhlJBs7) for further details pertaining to the test plan.

## Test Utility
We will perform tests using the [Whiteblock](www.whiteblock.io) testing platform. The following functionalities would likely be most relevant to this particular test series:

* Number of nodes: <100 (more if necessary)
* Automated provisioning of nodes
* Behaviors, parameters, commands, and actions can be automated or assigned to individual nodes and also the network as a whole.
* Bandwidth: 1G (standard, up to 10G if necessary) can be configured and assigned to each individual node.
* VLAN: Each node can be configured within its own VLAN and assigned a unique IP address, allowing for the emulation of realistic network conditions which accurately mimic real-world conditions.
* Latency: Up to 1 second of network latency can be applied to each node’s individual link.
* Data aggregation and visualization

## Test Scenarios

* Observe and measure performance under the presence of various network conditions.
  * Latency between nodes:
    * What is the maximum amount of network latency each individual node can tolerate before performance begins to degrade?
    * What are the security implications of high degrees of latency?
    * Are there any other unforeseen issues which may arise from network conditions for which we can’t accommodate via traditional code?
  * Latency between shards.
  * Intermittent blackout conditions
  * High degrees of packet loss
  * Bandwidth constraints (various bandwidth sizes)
* Introduce new nodes to network:
  * Add/remove nodes at random.
  * Add/remove nodes at set intervals.
  * Introduce a high volume of nodes simultaneously.
* Partition tolerance
  * Prevent segments of nodes from communicating with one another.
* Measure the performance of sending/receiving messages within set time periods and repeat for N epoches.
* Observe process of nodes joining and leaving shards
  * Subscribing to shards
  * Connecting to other nodes within shard
  * Synchronize collations and ShardPreferenceTable
  * Unsubscribing from shards

## Need to Define

* Configuration specifications for relevant test cases to create templates which allow for quicker test automation.
* Code which should be tested.
* Preliminary testing methodology should be established based on everyone's input.
  * We can make adjustments to this methodology based on the results of each test case.
  * It's generally best (in my experience) to create a high-level overview which provides a more granular definition of the first three test cases and then make adjustments to each subsequent test series based on the results of those three.

## Other Notes

* This document acts as a high-level overview to communicate potential test scenarios based on our initial assumptions of the existing codebase. It is meant to act as a starting point to guide the development of a preliminary test series.
* Although tests will be run locally within our lab, access to the test network can be granted to appropriate third-parties for the sake of due diligence, validation, or other purposes as deemed necessary.
* Network statistics and performance data dashboard can be assigned a public IP to allow for public access.
* Raw and formatted data will be shared within appropriate repos.
* Please voice any suggestions, comments, or concerns in this thread and feel free to contact me on Gitter.
