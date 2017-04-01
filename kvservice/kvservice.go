/*

This package specifies the application's interface to the key-value
service library to be used in assignment 7 of UBC CS 416 2016 W2.

*/

package kvservice

import (
	"fmt"
	"log"
	"net/rpc"
	"os"
)


var (
	kvnodeIpPorts []string
)

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

type myconn int

type mytx struct {
	ID int
}

type NewTransactionResp struct {
	TxID int
}

type PutRequest struct {
	TxID int
	K Key
	V Value
}

type PutResponse struct {
	Success bool
	Err error
}

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

func (c *myconn) NewTX() (tx, error) {
	fmt.Println("kvservice received a call to NewTX()")
	newTx := new(mytx)
	newTx.ID = getNewTransactionID()
	return newTx, nil
}

func getNewTransactionID() int {
	var resp NewTransactionResp
	client, err := rpc.Dial("tcp", kvnodeIpPorts[0])
	checkError("Error in getNewTransactionID(), rpc.Dial():", err, true)
	err = client.Call("KVServer.NewTransaction", true, &resp)
	checkError("Error in getNewTransactionID(), client.Call():", err, true)
	err = client.Close()
	checkError("Error in getNewTransactionID(), client.Close():", err, true)
	return resp.TxID
}

func (c *myconn) GetChildren(node string, parentHash string) (children []string) {
	fmt.Println("kvservice received a call to GetChildren(", node, parentHash, ")")
	return []string{"TODO: implement GetChildren", "First Child", "Second Child"}
}

func (c *myconn) Close() {
	fmt.Println("kvservice received a call to Close()")
}

func (t *mytx) Get(k Key) (success bool, v Value, err error) {
	fmt.Println("kvservice received a call to Get(", k, ")")
	return true, *new(Value), nil
}

func (t *mytx) Put(k Key, v Value) (success bool, err error) {
	fmt.Println("kvservice received a call to Put(", k, v, ")")
	success, err = put(t.ID, k, v)
	return
}

func put(txid int, k Key, v Value) (success bool, err error) {
	req := PutRequest{txid, k, v}
	var resp PutResponse
	client, err := rpc.Dial("tcp", kvnodeIpPorts[0])
	checkError("Error in put(), rpc.Dial():", err, true)
	err = client.Call("KVServer.Put", req, &resp)
	checkError("Error in put(), client.Call():", err, true)
	err = client.Close()
	checkError("Error in put(), client.Close():", err, true)
	return resp.Success, resp.Err
}

func (t *mytx) Commit(validateNum int) (success bool, txID int, err error) {
	fmt.Println("kvservice received a call to Commit(", validateNum, ")")
	return true, 0, nil
}

func (t *mytx) Abort() {
	fmt.Println("kvservice received a call to Abort()")
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
