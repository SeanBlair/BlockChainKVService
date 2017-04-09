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
    "io/ioutil"
	"log"
	"net"
	"net/rpc"
	"os"
	"strconv"
	"strings"
	"math"
	"time"
)

var (
	genesisHash string

	leafBlocks map[string]Block

	numLeadingZeroes int 
	nodeIPs []string
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
	// true when not receiving a Block from other kvnode
	isGenerateCommits bool
	// true when busy working on NoOp Block
	isWorkingOnNoOp bool
	// true when busy working on commit Block
	isWorkingOnCommit bool

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
	Depth int
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
	KeySet map[Key]bool
	IsAborted bool
	IsCommitted bool
	CommitID int
}

// For registering RPC's
type KVNode int
type KVServer int

// KVNode Request and Response structs
type AddBlockRequest struct {
	Block Block
}

// KVClient Request and Response structs
type NewTransactionResp struct {
	TxID int
	// The keyValueStore state on call to NewTX
	KeyValueStore map[Key]Value
}

type CommitRequest struct {
	Transaction Transaction
	// The original values in keyValueStore of keys that the transaction touched
	RequiredKeyValues map[Key]Value
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
		"numLeadingZeroes:", numLeadingZeroes, "nodesFilePath:", nodeIPs, "myNodeID:", myNodeID, 
		"listenKVNodeIpPort:", listenKVNodeIpPort, "listenClientIpPort:", listenClientIpPort)

	nextTransactionID = 1
	nextCommitID = 10
	transactions = make(map[int]Transaction)
	keyValueStore = make(map[Key]Value)
	blockChain = make(map[string]Block)
	leafBlocks = make(map[string]Block)

	// Add genesis block to blockChain map
	genesisBlock := Block{Hash: genesisHash, IsOnLongestBranch: true, Depth: 0}
	blockChain[genesisHash] = genesisBlock
	// Add genesis block to leafBlocks map
	leafBlocks[genesisHash] = genesisBlock

	isGenerateNoOps = true
	isWorkingOnNoOp = false
	isGenerateCommits = true
	isWorkingOnCommit = false
	printState()
	go listenNodeRPCs()
	go listenClientRPCs()
	time.Sleep(4 * time.Second)
	generateNoOpBlocks()
}

// Generates NoOp Blocks and adds to blockChain when not generating a Commit Block
func generateNoOpBlocks() {
	fmt.Println("In generateNoOpBlocks()")
	for {
		if isGenerateNoOps && !isWorkingOnCommit {
			isWorkingOnNoOp = true
			generateNoOpBlock()
			isWorkingOnNoOp = false
			printState()
			time.Sleep(time.Millisecond * 100)
		} else {
			time.Sleep(time.Second)
		}
	}
}

// While isGenerateNoOps, works on adding NoOps to the blockChain
// Returns either when isGenerateNoOps = false or successfully generates 1 NoOp
func generateNoOpBlock() {
	fmt.Println("Generating a NoOp Block...")
	if len(leafBlocks) > 1 {
		fmt.Println("We have a fork!!!!!!!!!!!!!!")
	}
	noOpBlock := Block { HashBlock: HashBlock{Txn: Transaction{}, NodeID: myNodeID, Nonce: 0}}
	noOpBlock = setCorrectParentHashAndDepth(noOpBlock)
	for isGenerateNoOps {
		success, _ := generateBlock(&noOpBlock)
		if success {
			return
		}
	}
 	// received a call to commit or AddBlock which set isGenerateNoOps = false
	return
}

func setCorrectParentHashAndDepth(block Block) Block {
	fmt.Println("in setCorrectParentHashAndDepth()")
	commitBlocks := getCommitLeafBlocks()
	var parentBlock Block
	// only one block (no fork), or all noOp blocks
	if len(leafBlocks) == 1 || len(commitBlocks) == 0 {
		for leafHash := range leafBlocks {
			// this randomly picks a block because the order returned from range on maps is undefined
			parentBlock = leafBlocks[leafHash]
			break
		}
	} else {
		// need to choose a Commit Block
		// if only 1, choose it, or if block is a NoOp choose random
		// if block is Commit Block, there shouldn't be conflicting transactions commited
		// bacause Commit checks that before this call, therefore random parent should be ok.
		for leafHash := range commitBlocks {
			parentBlock = commitBlocks[leafHash]
			break
		}	
	}
	block.HashBlock.ParentHash = parentBlock.Hash
	block.Depth = parentBlock.Depth + 1
	return block 
}

// returns leaf blocks that are Commit blocks (not NoOp blocks)
func getCommitLeafBlocks() (commitBlocks map[string]Block) {
	fmt.Println("in getCommitLeafBlocks()")
	commitBlocks = make(map[string]Block)
	for leafBlockHash := range leafBlocks {
		if leafBlocks[leafBlockHash].HashBlock.Txn.ID != 0 {
			commitBlocks[leafBlockHash] = leafBlocks[leafBlockHash]
		}
	}
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
		b.IsOnLongestBranch = true
		addToBlockChain(b)
		printState()
		broadcastBlock(b)
		fmt.Println("Done generating new block")
		return true, hashString
	} else {
		b.HashBlock.Nonce = b.HashBlock.Nonce + 1
		*block = b
		return false, ""
	}
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
		fmt.Println("    KeySet:")
		for k := range tx.KeySet {
			fmt.Println("      Key:", k)
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
	fmt.Printf("GenesisBlockChildren: %x\n\n", genesisBlock.ChildrenHashes)
	for _, childHash := range genesisBlock.ChildrenHashes {
		printBlock(childHash)
	}
}

// Prints one block in the blockChain to console
func printBlock(blockHash string) {
	block := blockChain[blockHash]
	indent := ""
	for i := 0; i < block.Depth; i++ {
		indent += " "
	}
	fmt.Printf("%sBlockTransactionID: %v\n", indent, block.HashBlock.Txn.ID)
	fmt.Printf("%sBlock.Hash :%x\n", indent, block.Hash)
	fmt.Printf("%sBlock.Depth :%v\n", indent, block.Depth)
	fmt.Printf("%sBlock.ChildrenHashes :%x\n", indent, block.ChildrenHashes)
	fmt.Printf("%sBlock.IsOnLongestBranch :%v\n", indent, block.IsOnLongestBranch)
	hashBlock := block.HashBlock
	fmt.Printf("%sBlock.HashBlock.ParentHash :%x\n", indent, hashBlock.ParentHash)
	fmt.Printf("%sBlock.HashBlock.Txn :%v\n", indent, hashBlock.Txn)
	fmt.Printf("%sBlock.HashBlock.NodeID :%v\n", indent, hashBlock.NodeID)
	fmt.Printf("%sBlock.HashBlock.Nonce :%x\n\n", indent, hashBlock.Nonce)

	for _, childHash := range block.ChildrenHashes {
		printBlock(childHash)
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
	*resp = NewTransactionResp{txID, keyValueStore}
	printState()
	return nil
}

// If the given transaction is aborted returns false, otherwise commits the transaction,
// and returns its CommitID value, 
func (p *KVServer) Commit(req CommitRequest, resp *CommitResponse) error {
	fmt.Println("Received a call to Commit(", req, ")")
	tx := req.Transaction
	transactions[tx.ID] = tx
	isGenerateNoOps = false
	fmt.Println("Commit Waiting for NoOp...")
	for isWorkingOnNoOp {
		// This stopped it from hanging... !
		time.Sleep(time.Millisecond)
	}
	fmt.Println("Commit done waiting for NoOp.")
	if !isCommitPossible(req.RequiredKeyValues) {
		t := transactions[tx.ID]
		t.IsAborted = true
		transactions[tx.ID] = t
		*resp = CommitResponse{false, 0, abortedMessage}
		isGenerateNoOps = true
	} else {
		txn := transactions[tx.ID]
		txn.IsCommitted = true
		transactions[tx.ID] = txn
		blockHash := generateCommitBlock(tx.ID)
		// TODO check that it is on longest block...
		// else: regenerate on correct branch??
		// Idea: check if Block.IsOnLongestBranch == true. Maybe don't set it until sure???
		// TODO give correct commitID... (Block Depth?)
		commitId := commit(tx.ID)
		isGenerateNoOps = true
		validateCommit(req, blockHash)
		*resp = CommitResponse{true, commitId, ""}
	}
	// printState()
	return nil
}

// Returns true if keyValueStore has the same values for the keys of requiredKeyValues
// This means the keyValueStore has the same values it had when the transaction started.
func isCommitPossible(requiredKeyValues map[Key]Value) bool {
	for k := range requiredKeyValues {
		val, ok := keyValueStore[k]
		if ok && val != requiredKeyValues[k] {
			return false
		} else if !ok && val != "" {
			return false	
		}
	}
	return true
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
	// TODO is this bool necessary?? Is this the only place using it??
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
func commit(txid int) (commitId int) {
	tx := transactions[txid]
	putSet := tx.PutSet
	for k := range putSet {
		keyValueStore[k] = putSet[k]
	}
	commitId = nextCommitID
	nextCommitID += 10
	tx.IsCommitted = true
	tx.CommitID = commitId
	transactions[txid] = tx
	return
}

// Adds a Commit Block with transaction txid to the blockChain, 
// returns its hash
func generateCommitBlock(txid int) string {
	fmt.Println("Generating a Commit Block...")
	block := Block { HashBlock: HashBlock{Txn: transactions[txid], NodeID: myNodeID, Nonce: 0} }	
	for {
		isInChain, hash := isBlockInChain(txid)
		if isInChain {
			return hash 
		} else {
			block = setCorrectParentHashAndDepth(block)
			for isGenerateCommits {
				// fmt.Println("trying new commit")
				// does not allow AddBlock to interupt
				isWorkingOnCommit = true
				success, blockHash := generateBlock(&block)
				isWorkingOnCommit = false
				if success {
					// Allows AddBlock to add to the block chain
					return blockHash
				}
			}
		}
	}
}

func isBlockInChain(txid int) (bool, string) {
	bChain := blockChain 
	for hash := range bChain {
		if bChain[hash].HashBlock.Txn.ID == txid {
			return true, hash
		}
	}
	return false, ""
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

func broadcastBlock(block Block) {
	fmt.Println("In broadcastBlock()")
	req := AddBlockRequest{block}

	for i, ip := range nodeIPs {
		id := i + 1
		if(id == myNodeID) {
			continue
		} else {
			fmt.Println(id, ip)
			var resp bool
			client, err := rpc.Dial("tcp", ip)
			err = client.Call("KVNode.AddBlock", req, &resp)
			checkError("Failed KVNode.AddBlock in broadcastBlock()", err, false)
			if(resp == false) {
				// TODO: Decide what to do when node fails to accept new block
			}
		}
	}
}

func (p *KVNode) AddBlock(req AddBlockRequest, resp *bool) error {
	b := req.Block
	hb := b.HashBlock
	data := []byte(fmt.Sprintf("%v", hb))
	sum := sha256.Sum256(data)
	hash := sum[:] // Converts from [32]byte to []byte
	*resp = isLeadingNumZeroes(hash)
	if(*resp == true) {
		fmt.Println("Received HashBlock: VERIFIED")
		// to allow return to caller
		go func() {

			
			// stop generating noOps when we have a new Block in the block chain...
			isGenerateNoOps = false
			// stop generating Commits when we have a new Block in the chain
			isGenerateCommits = false
			fmt.Println("AddBlock is Waiting for NoOp...")
			for isWorkingOnNoOp {
				// This stopped it from hanging... !!!
				time.Sleep(time.Millisecond)
			}
			fmt.Println("AddBlock is Done Waiting for NoOp")
			
			fmt.Println("AddBlock is Waiting for Commit")
			for isWorkingOnCommit {
				// This stopped it from hanging... !!!
				time.Sleep(time.Millisecond)
			}
			fmt.Println("AddBlock is Done Waiting for Commit")	

			addToBlockChain(b)
			isGenerateCommits = true
			isGenerateNoOps = true
			printState()
			} ()
	} else {
		fmt.Println("Received HashBlock: FAILED VERIFICATION")
	}
	return nil
}

// 
func addToBlockChain(block Block) {
	fmt.Println("In addToBlockChain()")
	blockChain[block.Hash] = block
	setParentsNewChild(block)
	updateLeafBlocks(block)
	// TODO if a commit block, ensure that it is on the longest chain. ??
}

// Adds block to leafBlocks and remove blocks with lesser depth than block
func updateLeafBlocks(block Block) {
	leafBlocks[block.Hash] = block
	for leafBlockHash := range leafBlocks {
		leafBlock := leafBlocks[leafBlockHash]
		// Remove blocks with lesser depth
		if leafBlock.Depth < block.Depth {
			delete(leafBlocks, leafBlockHash)	
		}
	}
}

// Adds block.Hash to its parent's ChildrenHashes
func setParentsNewChild(block Block) {
	parentBlock := blockChain[block.HashBlock.ParentHash]
	children := parentBlock.ChildrenHashes
	children = append(children, block.Hash)
	parentBlock.ChildrenHashes = children
	blockChain[parentBlock.Hash] = parentBlock
}

// Infinitely listens and serves KVNode RPC calls
func listenNodeRPCs() {
	kvNode := rpc.NewServer()
	kv := new(KVNode)
	kvNode.Register(kv)
	l, err := net.Listen("tcp", listenKVNodeIpPort)
	checkError("Error in listenNodeRPCs(), net.Listen()", err, true)
	fmt.Println("Listening for node RPC calls on:", listenKVNodeIpPort)
	for {
		conn, err := l.Accept()
		checkError("Error in listenNodeRPCs(), l.Accept()", err, true)
		go kvNode.ServeConn(conn)
	}
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
		nodeIPs = parseNodeFile(arguments[2])
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

func parseNodeFile(nodeFile string) (nodeIPs []string) {
	var err error
	nodeContent, err := ioutil.ReadFile(nodeFile)
	checkError("Failed to parse Nodefile: ", err, true)
	nodeIPs = strings.Split(string(nodeContent), "\n")
	nodeIPs = nodeIPs[:len(nodeIPs)-1] // Remove empty string
	fmt.Printf(" Nodes = %v, length = %v\n", nodeIPs, len(nodeIPs))
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


