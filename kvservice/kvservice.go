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
)


var (
	kvnodeIpPorts []string
	currentTransaction Transaction
	originalKeyValueStore map[Key]Value
	abortedMessage string = "This transaction is aborted!!"
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
/////////////////////////////////////////////


// The 'constructor' for a new logical connection object. This is the
// only way to create a new connection. Takes a set of k-v service
// node ip:port strings.
func NewConnection(nodes []string) connection {
	fmt.Println("kvservice received a call to NewConnection() with nodes:", nodes)
	kvnodeIpPorts = nodes
	fmt.Println("kvnodeIpPorts:", kvnodeIpPorts)
	c := new(myconn)
	return c
}

// Initializes a Transaction
func (c *myconn) NewTX() (tx, error) {
	fmt.Println("kvservice received a call to NewTX()")
	newTx := new(mytx)
	newTx.ID = getNewTransactionID()
	return newTx, nil
}

// Calls KVServer.NewTransaction RPC, returns a unique transaction ID
func getNewTransactionID() int {
	var resp NewTransactionResp
	client, err := rpc.Dial("tcp", kvnodeIpPorts[0])
	checkError("Error in getNewTransactionID(), rpc.Dial():", err, true)
	err = client.Call("KVServer.NewTransaction", true, &resp)
	checkError("Error in getNewTransactionID(), client.Call():", err, true)
	err = client.Close()
	checkError("Error in getNewTransactionID(), client.Close():", err, true)

	currentTransaction = Transaction{resp.TxID, make(map[Key]Value), make(map[Key]bool), false, false, 0}
	originalKeyValueStore = resp.KeyValueStore
	printState()

	return resp.TxID
}

// 
func (c *myconn) GetChildren(node string, parentHash string) (children []string) {
	fmt.Printf("kvservice received a call to GetChildren  %s  %x\n", node, parentHash)	
	req := GetChildrenRequest{parentHash}
	var resp GetChildrenResponse
	client, err := rpc.Dial("tcp", node)
	checkError("Error in GetChildren(), rpc.Dial():", err, true)
	err = client.Call("KVServer.GetChildren", req, &resp)
	checkError("Error in GetChildren(), client.Call():", err, true)
	err = client.Close()
	checkError("Error in GetChildren(), client.Close():", err, true)
	return resp.Children
}

// Stub
func (c *myconn) Close() {
	fmt.Println("kvservice received a call to Close()")
}

// Returns the Value associated with the given Key
func (t *mytx) Get(k Key) (success bool, v Value, err error) {
	fmt.Println("kvservice received a call to Get(", k, ")")
	if currentTransaction.IsAborted {
		return false, "", errors.New(abortedMessage)
	} else {
		val, ok := currentTransaction.PutSet[k]
		if !ok {
			val = originalKeyValueStore[k]
		}
		return true, val, nil	
	}
}

// Associates Value v with Key k in the system
func (t *mytx) Put(k Key, v Value) (bool, error) {
	fmt.Println("kvservice received a call to Put(", k, v, ")")
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
	fmt.Println("kvservice received a call to Commit(", validateNum, ")")
	success, commitID, err = commit(t.ID, validateNum)
	return
}

// Calls KVServer.Commit RPC to start the process of committing transaction txid
func commit(txid int, validateNum int) (success bool, commitID int, err error) {
	req := CommitRequest{txid, validateNum}
	var resp CommitResponse
	client, err := rpc.Dial("tcp", kvnodeIpPorts[0])
	checkError("Error in commit(), rpc.Dial():", err, true)
	err = client.Call("KVServer.Commit", req, &resp)
	checkError("Error in commit(), client.Call():", err, true)
	err = client.Close()
	checkError("Error in commit(), client.Close():", err, true)
	return resp.Success, resp.CommitID, errors.New(resp.Err)
}

// Calls KVServer.Abort to abort the given transaction
func (t *mytx) Abort() {
	var resp bool
	fmt.Println("kvservice received a call to Abort()")
	client, err := rpc.Dial("tcp", kvnodeIpPorts[0])
	checkError("Error in Abort(), rpc.Dial():", err, true)
	err = client.Call("KVServer.Abort", t.ID, &resp)
	checkError("Error in Abort(), client.Call():", err, true)
	err = client.Close()
	checkError("Error in Abort(), client.Close():", err, true)
	return	
}

// For visualizing the current state of kvservice's originalKeyValueStore map and currentTransaction
func printState () {
	fmt.Println("\nKVSERVICE STATE:")
	fmt.Println("-originalKeyValueStore:")
	for k := range originalKeyValueStore {
		fmt.Println("    Key:", k, "Value:", originalKeyValueStore[k])
	}
	fmt.Println("-currentTransaction:")
	tx := currentTransaction
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

// Prints msg + err to console and exits program if exit == true
func checkError(msg string, err error, exit bool) {
	if err != nil {
		log.Println(msg, err)
		if exit {
			os.Exit(-1)
		}
	}
}
