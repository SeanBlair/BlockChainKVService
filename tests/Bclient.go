/*
For concurrent client testing scenarios

Usage:
go run Bclient.go
*/

package main

import "../kvservice"

import (
	"fmt"
)
func main() {
	var nodes []string
	nodes = []string{"52.233.40.133:2222"}

	c := kvservice.NewConnection(nodes)
	fmt.Printf("NewConnection returned: %v\n", c)

	children := c.GetChildren(nodes[0], "parentHash")
	fmt.Printf("GetChildren returned: %v\n", children) 

	t1, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t1, err)

	success, err := t1.Put("B", "T2")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, v, err := t1.Get("B")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, txID, err := t1.Commit(1)
	fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)

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
