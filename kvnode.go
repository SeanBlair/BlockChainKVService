/*
A distributed block-chain transactional key-value service

Assignment 7 of UBC CS 416 2016 W2
http://www.cs.ubc.ca/~bestchai/teaching/cs416_2016w2/assign7/index.html

Created by Harlan Sim and Sean Blair, April 2017

This package represents the kvnode component of the system.

The kvnode process command line usage must be:

go run kvnode.go [ghash] [num-zeroes] [nodesFile] [nodeID] [listen-node-in IP:port] [listen-client-in IP:port]

example: go run kvnode.go 5473be60b466a24872fd7a007c41d1455e9044cca57d433eb51271b61bc16987 2 nodeList.txt 1 localhost:2223 localhost:2222

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
	"crypto/sha256"
	"fmt"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"math"
	"time"
)

var (
	genesisBlock Block
	genesisHash string
	leafBlockHash string // TODO: Reconsider naming
	numLeadingZeroes int 
	nodesFilePath string
	myNodeID int 
	listenKVNodeIpPort string
	listenClientIpPort string

	// Transaction ID's are incremented by 1
	nextTransactionID int
	
	// Commit ID's are incremented by 10, to allow reordering due to Block-Chain logic
	nextCommitID int

	// All transactions the system has seen
	transactions map[int]Transaction 

	// Represents the values corresponding to the in-order execution of all the 
	// transactions along the block-chain. Only holds values of commited transactions.
	keyValueStore map[Key]Value

	// Maps BlockHash to Block
	blockChain map[string]Block 

	isGenerateNoOps bool

	// For debugging...
	// done chan int

	abortedMessage string = "This transaction is aborted!!"
)

// Represents a key in the system.
type Key string

// Represent a value in the system.
type Value string

type HashBlock struct {
	ParentHash string
	Txn Transaction
	NodeID int
	Nonce uint32
}

type Block struct {
	Hash string
	ChildrenHashes []string
	HashBlock HashBlock
}

type Transaction struct {
	ID int

	// For storing this transaction's Puts before it commits.
	// On commit, they will be added to the keyValueStore
	PutSet map[Key]Value
	KeySet []Key
	IsAborted bool
	IsCommitted bool
	CommitID int
}

// For registering RPC's
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
	Err string
}

type GetRequest struct {
	TxID int
	K Key
}

type GetResponse struct {
	Success bool
	Val Value
	Err string	
}

type CommitRequest struct {
	TxID int
	ValidateNum int
}

type CommitResponse struct {
	Success bool
	CommitID int
	Err string
}

func main() {
	err := ParseArguments()
	checkError("Error in main(), ParseArguments():\n", err, true)
	fmt.Println("KVNode's command line arguments are:\ngenesisHash:", genesisHash, 
		"numLeadingZeroes:", numLeadingZeroes, "nodesFilePath:", nodesFilePath, "myNodeID:", myNodeID, 
		"listenKVNodeIpPort:", listenKVNodeIpPort, "listenClientIpPort:", listenClientIpPort)

	nextTransactionID = 1
	nextCommitID = 10
	transactions = make(map[int]Transaction)
	keyValueStore = make(map[Key]Value)
	blockChain = make(map[string]Block)
	genesisBlock = Block{Hash: genesisHash}
	blockChain[genesisBlock.Hash] = genesisBlock
	leafBlockHash = genesisHash
	isGenerateNoOps = true
	printState()
	go generateNoOpBlocks()
	listenClientRPCs()
}

func generateNoOpBlocks() {
	fmt.Println("In generateNoOpBlocks()")
	for {
		if isGenerateNoOps {
			generateNoOpBlock()
			printState()
		} else {
			time.Sleep(time.Second)
		}
	}
}

func generateNoOpBlock() {
	fmt.Println("In generateNoOpBlock()")
	noOpBlock := Block { HashBlock: HashBlock{ParentHash: leafBlockHash, Txn: Transaction{}, NodeID: myNodeID, Nonce: 0} }
	tempHashBlock := noOpBlock.HashBlock
	data := []byte(fmt.Sprintf("%v", tempHashBlock))
	sum := sha256.Sum256(data)
	hash := sum[:] // Converts from [32]byte to []byte

	for isGenerateNoOps {
		if isLeadingNumZeroes(hash) {
			hashString := string(hash)
			noOpBlock.Hash = hashString
			noOpBlock.HashBlock = tempHashBlock
			
			leafBlock := blockChain[leafBlockHash]
			leafBlockChildren := leafBlock.ChildrenHashes
			leafBlockChildren = append(leafBlockChildren, hashString)
			leafBlock.ChildrenHashes = leafBlockChildren
			blockChain[leafBlockHash] = leafBlock

			leafBlockHash = hashString
			// TODO: broadcast Block
			return
		} else {
			tempHashBlock.Nonce = tempHashBlock.Nonce + 1
			data = []byte(fmt.Sprintf("%v", tempHashBlock))
			sum = sha256.Sum256(data)
			hash = sum[:] // Converts from [32]byte to []byte
		}
	}
 	// received a call to commit which set isGenerateNoOps = false
	return
}

// For visualizing the current state of a kvnode's keyValueStore and transactions maps
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
	fmt.Println("blockChain size:", len(blockChain))
	fmt.Println("Total number of transactions is:", len(transactions), "\n")
}

// Adds a Transaction struct to the transactions map, returns a unique transaction ID
func (p *KVServer) NewTransaction(req bool, resp *NewTransactionResp) error {
	fmt.Println("Received a call to NewTransaction()")
	txID := nextTransactionID
	nextTransactionID++
	transactions[txID] = Transaction{txID, make(map[Key]Value), []Key{}, false, false, 0}
	*resp = NewTransactionResp{txID}
	printState()
	return nil
}

// Returns false if the given transaction is aborted, otherwise adds a Put record 
// to the given transaction's PutSet, 
func (p *KVServer) Put(req PutRequest, resp *PutResponse) error {
	fmt.Println("Received a call to Put(", req, ")")
	if transactions[req.TxID].IsAborted {
		*resp = PutResponse{false, abortedMessage}
	} else {
		transactions[req.TxID].PutSet[req.K] = req.Val 
		appendKeyIfMissing(req.TxID, req.K)
		*resp = PutResponse{true, ""}	
	}
	printState()
	return nil
}

// Returns the Value corresponding to the given Key and to the given transaction's
// previous Put calls. Returns "" if key does not exist, and false
// if transaction is aborted.
func (p *KVServer) Get(req GetRequest, resp *GetResponse) error {
	fmt.Println("Received a call to Get(", req, ")")
	if transactions[req.TxID].IsAborted {
		*resp = GetResponse{false, "", abortedMessage}
	} else {
		appendKeyIfMissing(req.TxID, req.K)
		val := getValue(req)
		*resp = GetResponse{true, val, ""}
	}
	return nil
}

// Returns the given Key's value by first checking in the given transaction's PutSet,
// otherwise retrieves it from the keyValueStore. Returns "" if key does not exist
func getValue(req GetRequest) (val Value) {
	val, ok := transactions[req.TxID].PutSet[req.K]
	if !ok {
		val = keyValueStore[req.K]
	}
	return
}

// Sets the IsAborted field, of transaction with id == txid, to true
func (p *KVServer) Abort(txid int, resp *bool) error {
	fmt.Println("Received a call to Abort(", txid, ")")
	tx := transactions[txid]
	tx.IsAborted = true
	transactions[txid] = tx
	*resp = true
	printState()
	return nil
}

// If the given transaction is aborted returns false, otherwise commits the transaction,
// and returns its CommitID value, 
func (p *KVServer) Commit(req CommitRequest, resp *CommitResponse) error {
	fmt.Println("Received a call to Commit(", req, ")")
	if transactions[req.TxID].IsAborted {
		*resp = CommitResponse{false, 0, abortedMessage}
	} else {
		// TODO change this call??
		commitId := commit(req)
		newBlock := Block { HashBlock: HashBlock{ParentHash: leafBlockHash, Txn: transactions[req.TxID], NodeID: myNodeID, Nonce: 0} }
		newBlock = computeHash(newBlock)
		// Add to blockChain
		blockChain[newBlock.Hash] = newBlock
		// TODO: Broadcast newBlock
		*resp = CommitResponse{true, commitId, ""}
	}
	printState()
	return nil
}

// Adds all values in the given transaction's PutSet into the keyValueStore.
// Sets the given transaction's IsCommited field to true and its CommitID to the 
// current value of nextCommitID, returns this value and increments nextCommitID
func commit(req CommitRequest) (commitId int) {
	tx := transactions[req.TxID]
	putSet := tx.PutSet
	for k := range putSet {
		keyValueStore[k] = putSet[k]
	}
	commitId = nextCommitID
	nextCommitID += 10
	tx.IsCommitted = true
	tx.CommitID = commitId
	transactions[req.TxID] = tx
	return
}

func computeHash(block Block) Block {
	fmt.Println("Computing Hash...")
	tempHashBlock := block.HashBlock
	for {
		data := []byte(fmt.Sprintf("%v", tempHashBlock))
		fmt.Println("The tempHashBlock:", data)
		sum := sha256.Sum256(data)
		hash := sum[:] // Converts from [32]byte to []byte
		if isLeadingNumZeroes(hash) {
			block.Hash = string(hash)
			fmt.Println("The correct hash of tempHashBlock:", hash)
			block.HashBlock = tempHashBlock
			return block
		} else {
			tempHashBlock.Nonce = tempHashBlock.Nonce + 1
		}
	}
}

// Returns true if given hash has the minimum number of leading zeroes.  
func isLeadingNumZeroes(hash []byte) bool {
	if (numLeadingZeroes == 0) {
		return true
	} else {
		i := 0;
		numZeroes := numLeadingZeroes
		for {
			// numZeroes <= 8, byte at hash[i] will determine validity
			if numZeroes - 8 <= 0 {
				break
			} else {
				// numZeroes is greater than 8, byte at hash[i] must be zero
				if hash[i] != 0 {
					return false
				} else {
					i++
					numZeroes -= 8
				}
			}
		}
		// returns true if byte at hash[i] has the the minimum number of leading zeroes
		// if numZeroes is 8: hash[i] < 2^(8-8) == hash[1] < 1 == hash[i] must be (0000 0000)b.
		// if numZeroes is 1: hash[i] < 2^(8-1) == hash[1] < (1000 0000)b == hash[i] <= (0111 1111)b
		return float64(hash[i]) < math.Pow(2, float64(8 - numZeroes))
	}
}

// Looks up the transaction, if it already has key - return. Else, append it to
// the transaction's KeySet and replace the transaction in the map with it. 
// (Must replace transaction because it cannot be directly appended) ...Stupid pointer issues
func appendKeyIfMissing(txID int, k Key) {
	txn := transactions[txID]
	for _, key := range txn.KeySet {
		if key == k {
			return
		}
	}
	txn.KeySet = append(txn.KeySet, k)
	transactions[txID] = txn
	return
}

// Infinitely listens and serves KVServer RPC calls
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


