/*
Tests aborting transactions at different times, successive calls do not succeed.

Usage:
go run 5_TestSequential.go
*/

package main

import "../kvservice"

import (
	"fmt"
)
func main() {
	done := make(chan int)
	var nodes []string
	nodes = []string{"52.233.45.243:2222", "52.175.29.87:2222", "40.69.195.111:2222", "13.65.91.243:2222"}

	c := kvservice.NewConnection(nodes)
	fmt.Printf("NewConnection returned: %v\n", c)

	// 
	fmt.Println("\nTest1\n")
	t1, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t1, err)

	t2, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t2, err)
	go func() {
		success, err := t1.Put("A", "T1")
		fmt.Printf("Put returned: %v, %v\n", success, err)
	}()
	go func() {
		success, err := t2.Put("B", "T2")
		fmt.Printf("Put returned: %v, %v\n", success, err)
	}()
	<-done
}

// Test status: PASS -> kvnode sequentially deals with put requests. 

/*
Change the put function in kvnode.go
// Returns false if the given transaction is aborted, otherwise adds a Put record 
// to the given transaction's PutSet, 
func (p *KVServer) Put(req PutRequest, resp *PutResponse) error {
	fmt.Println("Received a call to Put(", req, ")")
	if transactions[req.TxID].IsAborted {
		*resp = PutResponse{false, abortedMessage}
	} else {
		transactions[req.TxID].PutSet[req.K] = req.Val 
		fmt.Println("Sleeping -", req.Val)
		time.Sleep(10 * time.Second)
		fmt.Println("Awaking -", req.Val)
		*resp = PutResponse{true, ""}	
	}
	printState()
	return nil
}
*/