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

	// true when not generating Commit Blocks
	isGenerateNoOps bool
	// true when busy working on NoOp Block
	isWorkingOnNoOp bool

	// For debugging...
	// done chan int

	abortedMessage string = "This transaction is aborted!!"
)

// Represents a key in the system.
type Key string

// Represent a value in the system.
type Value string

// A block in the blockChain
type Block struct {
	// hash of HashBlock field
	Hash string
	ChildrenHashes []string
	IsOnLongestBranch bool
	HashBlock HashBlock
}

// The part of a Block that gets hashed
type HashBlock struct {
	ParentHash string
	Txn Transaction
	NodeID int
	Nonce uint32
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

type GetChildrenRequest struct {
	ParentHash string
}

type GetChildrenResponse struct {
	Children []string
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
	genesisBlock := Block{Hash: genesisHash}
	blockChain[genesisHash] = genesisBlock
	leafBlockHash = genesisHash
	isGenerateNoOps = true
	isWorkingOnNoOp = false
	printState()
	go generateNoOpBlocks()
	listenClientRPCs()
}

// Generates NoOp Blocks and adds to blockChain when not generating a Commit Block
func generateNoOpBlocks() {
	fmt.Println("In generateNoOpBlocks()")
	for {
		if isGenerateNoOps {
			isWorkingOnNoOp = true
			generateNoOpBlock()
			printState()
			isWorkingOnNoOp = false
		} else {
			time.Sleep(time.Second)
		}
	}
}

// While isGenerateNoOps, works on adding NoOps to the blockChain
// Returns either when isGenerateNoOps = false or successfully generates 1 NoOp
func generateNoOpBlock() {
	fmt.Println("In generateNoOpBlock()")
	noOpBlock := Block { HashBlock: HashBlock{ParentHash: leafBlockHash, Txn: Transaction{}, NodeID: myNodeID, Nonce: 0} }
	for isGenerateNoOps {
		success, _ := generateBlock(&noOpBlock)
		if success {
			return
		}
	}
 	// received a call to commit which set isGenerateNoOps = false
	return
}

// Hashes the given Block's HashBlock once, if has sufficient leading zeroes, adds it 
// to blockChain, returns true and the hash. Otherwise, increments the Nonce and returns false, ""
func generateBlock(block *Block) (bool, string) {
	b := *block
	data := []byte(fmt.Sprintf("%v", b.HashBlock))
	sum := sha256.Sum256(data)
	hash := sum[:] // Converts from [32]byte to []byte
	// TODO: make sure to turn in with call to isLeadingNumZeroCharacters, 
	// not with call to isLeadingNumZeroes (which is used for finer control of block generation)
	if isLeadingNumZeroes(hash) {
	// if isLeadingNumZeroCharacters(hash) {
		hashString := string(hash)
		b.Hash = hashString
		// TODO make sure this is true!!!
		// have to implement fork support and longest branch identification...
		b.IsOnLongestBranch = true
		blockChain[hashString] = b
			
		// set previous leaf Block's new child	
		leafBlock := blockChain[leafBlockHash]
		leafBlockChildren := leafBlock.ChildrenHashes
		leafBlockChildren = append(leafBlockChildren, hashString)
		leafBlock.ChildrenHashes = leafBlockChildren
		blockChain[leafBlockHash] = leafBlock

		leafBlockHash = hashString
		
		// TODO: broadcast Block
		return true, hashString
	} else {
		b.HashBlock.Nonce = b.HashBlock.Nonce + 1
		*block = b
		return false, ""
	}
}

// For visualizing the current state of a kvnode's keyValueStore, transactions and blockChain maps
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
	fmt.Println("-blockChain:")
	printBlockChain()
	fmt.Println("blockChain size:", len(blockChain))
	fmt.Println("Total number of transactions is:", len(transactions), "\n")
}

// Prints the blockChain to console
func printBlockChain() {
	genesisBlock := blockChain[genesisHash]
	fmt.Printf("GenesisBlockHash: %x\n", genesisBlock.Hash)
	fmt.Printf("GenesisBlockChildren: %x\n", genesisBlock.ChildrenHashes)
	for _, childHash := range genesisBlock.ChildrenHashes {
		printBlock(childHash, 1)
	}
}

// Prints one block in the blockChain to console
func printBlock(blockHash string, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += " "
	}
	block := blockChain[blockHash]
	fmt.Printf("%sBlockTransactionID: %v\n", indent, block.HashBlock.Txn.ID)
	fmt.Printf("%sBlock.Hash :%x\n", indent, block.Hash)
	fmt.Printf("%sBlock.ChildrenHashes :%x\n", indent, block.ChildrenHashes)
	fmt.Printf("%sBlock.IsOnLongestBranch :%v\n", indent, block.IsOnLongestBranch)
	hashBlock := block.HashBlock
	fmt.Printf("%sBlock.HashBlock.ParentHash :%x\n", indent, hashBlock.ParentHash)
	fmt.Printf("%sBlock.HashBlock.Txn :%v\n", indent, hashBlock.Txn)
	fmt.Printf("%sBlock.HashBlock.NodeID :%v\n", indent, hashBlock.NodeID)
	fmt.Printf("%sBlock.HashBlock.Nonce :%x\n\n", indent, hashBlock.Nonce)

	for _, childHash := range block.ChildrenHashes {
		printBlock(childHash, depth + 1)
	}
}

// Returns the children hashes of the Block that has the given hash as key in the blockChain
func (p *KVServer) GetChildren(req GetChildrenRequest, resp *GetChildrenResponse) error {
	fmt.Println("Received a call to GetChildren with:", req)
	if req.ParentHash == "" {
		resp.Children = blockChain[genesisHash].ChildrenHashes
	} else {
		resp.Children = blockChain[req.ParentHash].ChildrenHashes
	}
	return nil
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
		isGenerateNoOps = false
		for isWorkingOnNoOp {
			fmt.Println("Commit Waiting for NoOp")
		}
		blockHash := generateCommitBlock(req.TxID)
		// TODO check that it is on longest block...
		// else: regenerate on correct branch??
		// TODO give correct commitID... 
		commitId := commit(req)
		isGenerateNoOps = true
		validateCommit(req, blockHash)
		*resp = CommitResponse{true, commitId, ""}
	}
	printState()
	return nil
}

// Waits until the Block with given blockHash has the correct number of descendant Blocks
func validateCommit(req CommitRequest, blockHash string) {
	for {
		if isBlockValidated(blockChain[blockHash], req.ValidateNum) {
			return
		} else {
			time.Sleep(time.Second)
		}
	}
}

// Recursively traverses the longest branch of the blockChain tree starting at the given block,
// if there are at least validateNum descendents returns true, else returns false  
func isBlockValidated(block Block, validateNum int) bool {
	if !block.IsOnLongestBranch {
		return false
	} else {
		if validateNum == 0 {
			return true
		} else {
			for _, child := range block.ChildrenHashes {
				if isBlockValidated(blockChain[child], validateNum - 1) {
					return true
				}
			}
			return false
		}
	}
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

// Adds a Commit Block with transaction txid to the blockChain, 
// returns its hash
func generateCommitBlock(txid int) string {
	fmt.Println("Generating a Commit Block...")
	block := Block { HashBlock: HashBlock{ParentHash: leafBlockHash, Txn: transactions[txid], NodeID: myNodeID, Nonce: 0} }
	for {
		success, blockHash := generateBlock(&block)
		if success {
			return blockHash
		}
	}
}

// Returns true if hash has numLeadingZeroes number of leading '0' characters (0x30)
// This is the correct implementation provided by the assignment specifications.
func isLeadingNumZeroCharacters(hash []byte) bool {
	if (numLeadingZeroes == 0) {
		return true
	} else {
		for i := 0; i < numLeadingZeroes; i++ {
			if rune(hash[i]) == '0' {
				continue
			} else {
				return false
			}
		}
		return true
	}
}

// Returns true if given hash has the minimum number of leading zeroes.
// This is incorrect given the assignment specs, but is useful for debugging
// as it provides more control over different amounts of proof-of-work required.
// TODO: make sure this is not used in the final code!!  
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
				 " [listen-node-in IP:port] [listen-client-in IP:port]}"
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


