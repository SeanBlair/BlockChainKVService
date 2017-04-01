/*
Tests basic communication between client and kvnode using kvservice.
Run repeatedly to verify that each transaction ID returned from NewTX() is unique.
Calls all required functionality once.

Usage:
go run 1_kvserviceStubs_NewTransactionRPC.go
*/

package main

import "../kvservice"

import (
	"fmt"
)
func main() {
	var nodes []string
	nodes = []string{"localhost:2222"}

	c := kvservice.NewConnection(nodes)
	fmt.Printf("NewConnection returned: %v\n", c)

	t, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t, err)

	children := c.GetChildren("nodeIpPort", "parentHash")
	fmt.Printf("GetChildren returned: %v\n", children) 

	success, err := t.Put("A", "Aclient")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, v, err := t.Get("B")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, txID, err := t.Commit(1)
	fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)

	t.Abort()

	c.Close()
}
