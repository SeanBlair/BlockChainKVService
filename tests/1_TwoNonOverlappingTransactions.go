/*
Tests basic communication between client and kvnode using kvservice.
Creates 2 non-overlapping transactions and commits them.

Usage:
go run 1_TwoNonOverlappingTransactions.go
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

	children := c.GetChildren(nodes[0], "parentHash")
	fmt.Printf("GetChildren returned: %v\n", children) 

	t1, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t1, err)

	success, err := t1.Put("A", "T1")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	// success, v, err := t1.Get("A")
	// fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	// success, txID, err := t1.Commit(0)
	// fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)

	// t2, err := c.NewTX()
	// fmt.Printf("NewTX returned: %v, %v\n", t2, err)

	// success, err = t2.Put("B", "T2")
	// fmt.Printf("Put returned: %v, %v\n", success, err)

	// success, v, err = t2.Get("B")
	// fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	// success, txID, err = t2.Commit(0)
	// fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)

	c.Close()
}
