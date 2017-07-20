Distributed Blockchain Transactional Key-Value Service
=
Description
-
Provides a key - value store that uses optimistic concurrency control and validates transactions via a hashing proof of work
blockchain protocol to mitigate byzantine nodes. Detailed assignment description: http://www.cs.ubc.ca/~bestchai/teaching/cs416_2016w2/assign7/index.html

**Summary** The system is composed of N nodes and supports N-1 node failures. It supports concurrent client transactions and
client failures. It supports transactional semantics (except Durability), put(key, value), get(key), abort() and commit(). 
When a client commits a transaction, all nodes race to complete a proof-of-work in order to integrate the transaction into the
blockchain.

Dependencies
-
Developed and tested with Go version 1.7.4 linux/amd64 on a Linux ubuntu 14.04 LTS machine

Running instructions:
=

- Open a terminal and clone the repo: git clone https://github.com/SeanBlair/BlockChainKVService.git
- Enter the folder: cd BlockChainKVService

Simple single machine test
-

Open four terminals, navigate to the BlockChainKVService folder and input one of the following commands in each terminal (without running them)
- go run kvnode.go asdf 22 nodeList.txt 1 localhost:1111 localhost:1999
- go run kvnode.go asdf 22 nodeList.txt 2 localhost:2222 localhost:2999
- go run kvnode.go asdf 22 nodeList.txt 3 localhost:3333 localhost:3999
- go run kvnode.go asdf 22 nodeList.txt 4 localhost:4444 localhost:4999

In rapid sucession, start the four processes by pressing enter in each terminal.

The four processes will start to build a distributed blockchain tree, each time a node finds a nonce that results in a hash with 22 zeroes, it will share it with the other nodes by using the ip:ports provided in the nodeList.txt file. These are called no-op blocks and the nodes are constantly working on these while there are no transactions to commit. 

Open a new terminal and navigate to the BlockChainKVService/tests folder.

Input and run the following command:

**go run 1_TwoNonOverlappingTransactions.go**

This client is configured to connect to the first kvnode and perform 2 simple consecutive transactions. It will take
some time (max a couple minutes, min a few seconds) to successfully commit the 2 transactions, at which time all four
nodes will have a record of them.

Next input and run:

**go run 2_PutTests.go**

This client is configured to connect to the second kvnode and test the put functionality.

Next input and run:

**go run 3_GetTests.go**

This client is configured to connect to the third kvnode and test the get functionality.

Next input and run:

**go run 4_AbortTests.go**

This client is configured to connect to the fourth kvnode and test the abort functionality.

Additional behaviour supported is the arbitrary death (Ctrl+c) of up to three of the four kvnodes. To continue to test the 
system, run the tests that are configured to connect to the kvnodes that are still alive. Also supported is client death
before transactions are finalized. Additional supported functionality is concurrent clients which can be tested be opening
additional terminals and running various combinations of the four tests concurrently.

Multiple Machine Testing
-
This system was designed to run on separate machines and was extensively tested using MS Azure to set up multiple virtual
machines situated in different geographical regions worldwide. This requires adjustment in the nodesList.txt and test client files, as well as the command line arguments of the kvnode.go programs. This allows additional functionality such as a single
client knowing about all the N initial kvnodes as well as increases the complexity due to varying network latency between all
parties. If you are interested in testing the system on multiple machines, please shoot me an email at seandibango@gmail.com
to help set it up correctly.
