/*

A distributed block-chain transactional key-value service

Assignment 7 of UBC CS 416 2016 W2
http://www.cs.ubc.ca/~bestchai/teaching/cs416_2016w2/assign7/index.html

Created by Harlan Sim and Sean Blair, April 2017

This package specifies the application's interface to the key-value
service library.

*/

package kvservice

import (
	"errors"
	"fmt"
	"log"
	"net/rpc"
	"os"
	"sort"
	"strings"
	"math"
	"strconv"
	"sync"
)


var (
	sortedKvnodeIpPortStatuses []NodeIpPortStatus
	currentTransaction Transaction
	originalKeyValueStore map[Key]Value
	abortedMessage string = "This transaction is aborted!!"
	mutex      *sync.Mutex
)

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

type NodeIpPortStatus struct {
	IpPort string
	IsAlive bool
}

// Represents a key in the system.
type Key string

// Represent a value in the system.
type Value string

// An interface representing a connection to the key-value store. To
// create a new connection use the NewConnection() method.
type connection interface {
	// The 'constructor' for a new logical transaction object. This is the
	// only way to create a new transaction. The returned transaction must
	// correspond to a specific, reachable, node in the k-v service. If
	// none of the nodes are reachable then tx must be nil and error must
	// be set (non-nil).
	NewTX() (newTX tx, err error)

	// Used by a client to ask a node for information about the
	// block-chain. Node is an IP:port string of one of the nodes that
	// was used to create the connection.  parentHash is either an
	// empty string to indicate that the client wants to retrieve the
	// SHA 256 hash of the genesis block. Or, parentHash is a string
	// identifying the hexadecimal SHA 256 hash of one of the blocks
	// in the block-chain. In this case the return value should be the
	// string representations of SHA 256 hash values of all of the
	// children blocks that have the block identified by parentHash as
	// their prev-hash value.
	GetChildren(node string, parentHash string) (children []string)

	// Close the connection.
	Close()
}

// An interface representing a client's transaction. To create a new
// transaction use the connection.NewTX() method.
type tx interface {
	// Retrieves a value v associated with a key k as part of this
	// transaction. If success is true then v contains the value
	// associated with k and err is nil. If success is false then the
	// tx has aborted, v is an empty string, and err is non-nil. If
	// success is false, then all future calls on this transaction
	// must immediately return success = false (indicating an earlier
	// abort).
	Get(k Key) (success bool, v Value, err error)

	// Associates a value v with a key k as part of this
	// transaction. If success is true, then put was recoded
	// successfully, otherwise the transaction has aborted (see
	// above).
	Put(k Key, v Value) (success bool, err error)

	// Commits this transaction. If success is true then commit
	// succeeded, otherwise the transaction has aborted (see above).
	// The validateNum argument indicates the number of blocks that
	// must follow this transaction's block in the block-chain along
	// the longest path before the commit returns with a success.
	// txID represents the transactions's global sequence number
	// (which determines this transaction's position in the serialized
	// sequence of all the other transactions executed by the
	// service).
	Commit(validateNum int) (success bool, txID int, err error)

	// Aborts this transaction. This call always succeeds.
	Abort()
}

// Concrete implementation of a connection interface
type myconn int

// Concrete implementation of a tx interface
type mytx struct {
	ID int
}

// RPC structs /////////////////////////////
type NewTransactionResp struct {
	TxID int
	KeyValueStore map[Key]Value
}

type CommitRequest struct {
	Transaction Transaction
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
/////////////////////////////////////////////


// The 'constructor' for a new logical connection object. This is the
// only way to create a new connection. Takes a set of k-v service
// node ip:port strings.
func NewConnection(nodes []string) connection {
	// fmt.Println("kvservice received a call to NewConnection() with nodes:", nodes)
	setSortedIpPorts(nodes)
	// fmt.Println("sortedKvnodeIpPortStatuses:", sortedKvnodeIpPortStatuses)
	c := new(myconn)
	return c
}

// sorts the unique ip addresses and sets sortedKvnodesIpPorts
func setSortedIpPorts(nodes []string) {
	nodeTotalKey := make(map[int]string)
	var totalList []int
	for _, node := range nodes {
		sum := getWeightedSum(node)
		totalList = append(totalList, sum)
		// check if sum already in nodeTotalKey map
		_, ok := nodeTotalKey[sum]
		var err error
		if ok {
			err = errors.New("NewConnection was called with non-unique ip addresses")
		}
		checkError("Error in setSortedIpPorts():", err, true)
		nodeTotalKey[sum] = node
	}
	// they are all unique
	sort.Ints(totalList)
	for _, sum := range totalList {
		nodeIpPortStatus := NodeIpPortStatus{nodeTotalKey[sum], true}
		sortedKvnodeIpPortStatuses = append(sortedKvnodeIpPortStatuses, nodeIpPortStatus)
	}
}

// returns the weighted sum of the ipv4 address which represents a unique number for each possible Ipv4 address
func getWeightedSum(ipPort string) (sum int) {
	ip := strings.Split(ipPort, ":")
	nums := strings.Split(ip[0], ".")
	for i, n := range nums {
		intN, err := strconv.Atoi(n)
		checkError("Error in getSum(), strconv.Atoi()", err, true)
		// a.b.c.d ==  a * 256^(4-0) + b * 256^(4-1) + c * 256^(4-2) + d * 256^(4-3)
		// returns a unique number representing each ip address 
		sum += int(math.Pow(256, float64(4 - i)) * float64(intN))
	}
	return
}

// Initializes a Transaction
func (c *myconn) NewTX() (tx, error) {
	// fmt.Println("kvservice received a call to NewTX()")
	mutex = &sync.Mutex{}
	newTx := new(mytx)
	newTx.ID = getNewTransactionIDFromAll()
	return newTx, nil
}

// Returns a transaction Id by querying all kvnodes.
// If none alive (impossible according to specs), returns -1
func getNewTransactionIDFromAll() (txid int) {
	newTxResponses := make(map[string]NewTransactionResp)
	txChannel := make(chan(NewTransactionResp))
	nodeChannel := make(chan(string))
	for i, nodeIpPortStatus := range sortedKvnodeIpPortStatuses {
		go func(nodeIP string, index int) {
			txChannel <- getNewTransactionID(nodeIP, index)
			nodeChannel <- nodeIP
		}(nodeIpPortStatus.IpPort, i)
	}
	// waits for all to respond
	for i := 0; i < len(sortedKvnodeIpPortStatuses); i++ {
		newTx := <-txChannel
		node := <-nodeChannel
		mutex.Lock()
		newTxResponses[node] = newTx 
		mutex.Unlock()
	}
	fmt.Println("Received all responses and they are:", newTxResponses)
	// TODO check and resolve different answers
	// currently just randomly returns one from a non-dead node...
	for nodeIpP := range newTxResponses {
		resp := newTxResponses[nodeIpP]
		if resp.TxID != -1 {
			fmt.Println("Returning newTx response:", resp, " from node:", nodeIpP)
			currentTransaction = Transaction{resp.TxID, make(map[Key]Value), make(map[Key]bool), false, false, 0}
			originalKeyValueStore = resp.KeyValueStore
			txid = resp.TxID
			return
		}
	}
	return -1
}

// Calls KVServer.NewTransaction RPC, returns a unique transaction ID
// if the called node is dead, sets its IsAlive = false and returns TxID -1
func getNewTransactionID(ipPort string, index int) (resp NewTransactionResp) {
	client, err := rpc.Dial("tcp", ipPort)
	// checkError("Error in getNewTransactionID(), rpc.Dial():", err, false)
	if err != nil {
		sortedKvnodeIpPortStatuses[index].IsAlive = false
		return NewTransactionResp{-1, nil}
	}
	err = client.Call("KVServer.NewTransaction", true, &resp)
	// checkError("Error in getNewTransactionID(), client.Call():", err, false)
	if err != nil {
		sortedKvnodeIpPortStatuses[index].IsAlive = false
		return NewTransactionResp{-1, nil}
	}
	err = client.Close()
	// checkError("Error in getNewTransactionID(), client.Close():", err, false)
	if err != nil {
		sortedKvnodeIpPortStatuses[index].IsAlive = false
		return NewTransactionResp{-1, nil}
	} 
	return
}

// 
func (c *myconn) GetChildren(node string, parentHash string) (children []string) {
	// fmt.Printf("kvservice received a call to GetChildren  %s  %x\n", node, parentHash)	
	req := GetChildrenRequest{parentHash}
	var resp GetChildrenResponse
	client, err := rpc.Dial("tcp", node)
	// checkError("Error in GetChildren(), rpc.Dial():", err, false)
	if err != nil {
		return []string{"The provided node is dead!!!"}
	}
	err = client.Call("KVServer.GetChildren", req, &resp)
	// checkError("Error in GetChildren(), client.Call():", err, false)
	if err != nil {
		return []string{"The provided node is dead!!!"}
	}
	err = client.Close()
	// checkError("Error in GetChildren(), client.Close():", err, false)
	if err != nil {
		return []string{"The provided node is dead!!!"}
	}
	return resp.Children
}

// Stub
func (c *myconn) Close() {
	// fmt.Println("kvservice received a call to Close()")
}

// Returns the Value associated with the given Key
func (t *mytx) Get(k Key) (success bool, v Value, err error) {
	// fmt.Println("kvservice received a call to Get(", k, ")")
	if currentTransaction.IsAborted {
		return false, "", errors.New(abortedMessage)
	} else {
		val, ok := currentTransaction.PutSet[k]
		if !ok {
			val = originalKeyValueStore[k]
		}
		currentTransaction.KeySet[k] = true
		printState()
		return true, val, nil	
	}
}

// Associates Value v with Key k in the system
func (t *mytx) Put(k Key, v Value) (bool, error) {
	// fmt.Println("kvservice received a call to Put(", k, v, ")")
	if currentTransaction.IsAborted {
		return false, errors.New(abortedMessage)
	} else {
		currentTransaction.PutSet[k] = v 
		currentTransaction.KeySet[k] = true
 		printState()
		return true, nil
	}
}

// Commits a transaction
func (t *mytx) Commit(validateNum int) (success bool, commitID int, err error) {
	// fmt.Println("kvservice received a call to Commit(", validateNum, ")")
	mutex = &sync.Mutex{}
	if currentTransaction.IsAborted {
		return false, 0, errors.New(abortedMessage)
	} else {
		success, commitID, err = commitAll(validateNum)
		if success {
			currentTransaction.IsCommitted = true
			currentTransaction.CommitID = commitID
		}	
	}
	printState()
	return
}

// 
func commitAll(validateNum int) (success bool, commitID int, err error) {
	commitResponses := make(map[string]CommitResponse)
	commitChan := make(chan(CommitResponse))
	nodeChan := make(chan(string))
	for i, nodeIpPortStatus := range sortedKvnodeIpPortStatuses {
		go func(node string, index int) {
			commitChan <- commit(node, validateNum, index)
			nodeChan <- node
		}(nodeIpPortStatus.IpPort, i)
	}
	// waits for all to respond
	for i := 0; i < len(sortedKvnodeIpPortStatuses); i++ {
		commitResp := <-commitChan
		node := <- nodeChan
		mutex.Lock()
		commitResponses[node] = commitResp
		mutex.Unlock()  
	}
	fmt.Println("Received all responses and they are:", commitResponses)

	// TODO check and resolve different answers
	// currently returns the first successful answer...
	for nodeIpP := range commitResponses {
		resp := commitResponses[nodeIpP]
		// TODO don't be so optimistic?
		// optimistic, if one was successful, nailed it!!
		if resp.Success {
			fmt.Println("Returning commit response:", resp, " from node:", nodeIpP)
			return resp.Success, resp.CommitID, errors.New(resp.Err)
		}
	}
	return false, -1, errors.New("This transaction is aborted!!")
}

// Calls KVServer.Commit RPC to start the process of committing transaction txid
func commit(nodeIpPort string, validateNum int, index int) (resp CommitResponse) {
	deadNodeMessage := "The queried kvnode is dead!!!"
	requiredKeyValues := getRequiredKeyValues()
	req := CommitRequest{currentTransaction, requiredKeyValues, validateNum}
	client, err := rpc.Dial("tcp", nodeIpPort)
	// checkError("Error in commit(), rpc.Dial():", err, false)
	if err != nil {
		sortedKvnodeIpPortStatuses[index].IsAlive = false
		return CommitResponse{false, -1, deadNodeMessage}
	}
	err = client.Call("KVServer.Commit", req, &resp)
	// checkError("Error in commit(), client.Call():", err, false)
	if err != nil {
		sortedKvnodeIpPortStatuses[index].IsAlive = false
		return CommitResponse{false, -1, deadNodeMessage}
	}
	err = client.Close()
	// checkError("Error in commit(), client.Close():", err, false)
	if err != nil {
		sortedKvnodeIpPortStatuses[index].IsAlive = false
		return CommitResponse{false, -1, deadNodeMessage}
	}
	return 
}

// Creates map of the original values that keys in KeySet had in originalKeyValueStore
// If they did not exist adds ""
func getRequiredKeyValues() map[Key]Value {
	kvMap := make(map[Key]Value)
	for k := range currentTransaction.KeySet {
		kvMap[k] = originalKeyValueStore[k]	
	}
	return kvMap
}

// Calls KVServer.Abort to abort the given transaction
func (t *mytx) Abort() {
	currentTransaction.IsAborted = true
	printState()
	return	
}

// For visualizing the current state of kvservice's originalKeyValueStore map and currentTransaction
func printState () {
	// fmt.Println("\nKVSERVICE STATE:")
	// fmt.Println("-originalKeyValueStore:")
	// for k := range originalKeyValueStore {
	// 	fmt.Println("    Key:", k, "Value:", originalKeyValueStore[k])
	// }
	// fmt.Println("-currentTransaction:")
	// tx := currentTransaction
	// fmt.Println("  --Transaction ID:", tx.ID, "IsAborted:", tx.IsAborted, "IsCommitted:", tx.IsCommitted, "CommitId:", tx.CommitID)
	// fmt.Println("    PutSet:")
	// for k := range tx.PutSet {
	// 	fmt.Println("      Key:", k, "Value:", tx.PutSet[k])
	// }
	// fmt.Println("    KeySet:")
	// for k := range tx.KeySet {
	// 	fmt.Println("      Key:", k)
	// }
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
