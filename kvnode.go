/*
A distributed block-chain transactional key-value service

Assignment 7 of UBC CS 416 2016 W2
http://www.cs.ubc.ca/~bestchai/teaching/cs416_2016w2/assign7/index.html

Created by Harlan Sim and Sean Blair, April 2017

This package represents the kvnode component of the system.

The kvnode process command line usage must be:

go run kvnode.go [ghash] [num-zeroes] [nodesFile] [nodeID] [listen-node-in IP:port] [listen-client-in IP:port]

[ghash] : SHA 256 hash in hexadecimal of the genesis block for this instantiation of the system.
[num-zeroes] : required number of leading zeroes in the proof-of-work algorithm, greater or equal to 1.
[nodesFile] : a file containing one line per node in the key-value service. Each line must be terminated by '\n' 
		and indicates the IP:port that should be used to by this node to connect to the other nodes in the service.
[nodeID] : an integer between 1 and number of lines in nodesFile. The IP:port on line i of nodesFile is the external 
		IP:port corresponding to the listen-node-in IP:port, which will be used by other nodes to connect to this node.
[listen-node-in IP:port] : the IP:port that this node should listen on to receive connections from other nodes.
[listen-client-in IP:port] : the IP:port that this node should listen on to receive connections from clients.
*/

package main

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
)

var (
	genesisHash string
	numLeadingZeroes int 
	nodesFilePath string
	myNodeID int 
	listenKVNodeIpPort string
	listenClientIpPort string
	nextTransactionID int
	transactions map[int]Transaction 
	keyValueStore map[Key]Value
)

// Represents a key in the system.
type Key string

// Represent a value in the system.
type Value string

type Transaction struct {
	ID int
	PutSet map[Key]Value
	IsAborted bool
	IsCommitted bool
	CommitID int
}

type KVServer int

type NewTransactionResp struct {
	TxID int
}

type PutRequest struct {
	TxID int
	K Key
	Val Value
}

type PutResponse struct {
	Success bool
	Err error
}

type GetRequest struct {
	TxID int
	K Key
}

type GetResponse struct {
	Success bool
	Val Value
	Err error	
}

func main() {
	err := ParseArguments()
	checkError("Error in main(), ParseArguments():\n", err, true)
	fmt.Println("KVNode's command line arguments are:\ngenesisHash:", genesisHash, 
		"numLeadingZeroes:", numLeadingZeroes, "nodesFilePath:", nodesFilePath, "myNodeID:", myNodeID, 
		"listenKVNodeIpPort:", listenKVNodeIpPort, "listenClientIpPort:", listenClientIpPort)

	nextTransactionID = 10
	transactions = make(map[int]Transaction)
	keyValueStore = make(map[Key]Value)

	printState()
	listenClientRPCs()
}

func printState () {
	fmt.Println("\nKVNODE STATE:")
	fmt.Println("-keyValueStore:")
	for k := range keyValueStore {
		fmt.Println("    Key:", k, "Value:", keyValueStore[k])
	}
	fmt.Println("-transactions:")
	for txId := range transactions {
		tx := transactions[txId]
		fmt.Println("  --Transaction ID:", tx.ID, "IsAborted:", tx.IsAborted, "IsCommitted:", tx.IsCommitted, "CommitId:", tx.CommitID)
		fmt.Println("    PutSet:")
		for k := range tx.PutSet {
			fmt.Println("      Key:", k, "Value:", tx.PutSet[k])
		}
	}
	fmt.Println("Total number of transactions is:", len(transactions), "\n")
}

func (p *KVServer) NewTransaction(req bool, resp *NewTransactionResp) error {
	fmt.Println("Received a call to NewTransaction()")
	txID := nextTransactionID
	nextTransactionID += 10
	transactions[txID] = Transaction{txID, make(map[Key]Value), false, false, 0}
	*resp = NewTransactionResp{txID}
	printState()
	return nil
}

func (p *KVServer) Put(req PutRequest, resp *PutResponse) error {
	fmt.Println("Received a call to Put(", req, ")")
	transactions[req.TxID].PutSet[req.K] = req.Val 
	*resp = PutResponse{true, nil}
	printState()
	return nil
}

func (p *KVServer) Get(req GetRequest, resp *GetResponse) error {
	fmt.Println("Received a call to Get(", req, ")")
	val := getValue(req)
	*resp = GetResponse{true, val, nil}
	return nil
}

func getValue(req GetRequest) (val Value) {
	val, ok := transactions[req.TxID].PutSet[req.K]
	if !ok {
		val = keyValueStore[req.K]
	}
	return
}

func listenClientRPCs() {
	kvServer := rpc.NewServer()
	kv := new(KVServer)
	kvServer.Register(kv)
	l, err := net.Listen("tcp", listenClientIpPort)
	checkError("Error in listenClientRPCs(), net.Listen()", err, true)
	fmt.Println("Listening for client RPC calls on:", listenClientIpPort)
	for {
		conn, err := l.Accept()
		checkError("Error in listenClientRPCs(), l.Accept()", err, true)
		kvServer.ServeConn(conn)
	}
}

// Parses and sets the command line arguments to kvnode.go as global variables
func ParseArguments() (err error) {
	arguments := os.Args[1:]
	if len(arguments) == 6 {
		genesisHash = arguments[0]
		numLeadingZeroes, err = strconv.Atoi(arguments[1])
		checkError("Error in ParseArguments(), strconv.Atoi(arguments[1]):", err, true)
		nodesFilePath = arguments[2]
		myNodeID, err = strconv.Atoi(arguments[3])
		checkError("Error in ParseArguments(), strconv.Atoi(arguments[3]):", err, true)
		listenKVNodeIpPort = arguments[4]
		listenClientIpPort = arguments[5]
	} else {
		usage := "Usage: {go run kvnode.go [ghash] [num-zeroes] [nodesFile] [nodeID]" + 
				 "[listen-node-in IP:port] [listen-client-in IP:port]}"
		err = fmt.Errorf(usage)
	}
	return
}

// Prints msg + err to console and exits program if exit == true
func checkError(msg string, err error, exit bool) {
	if err != nil {
		log.Println(msg, err)
		if exit {
			os.Exit(-1)
		}
	}
}


