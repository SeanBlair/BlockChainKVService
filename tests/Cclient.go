/*
For concurrent client testing scenarios

Usage:
go run Aclient.go
*/

package main

import "../kvservice"

import (
	"fmt"
)
func main() {
	var nodes []string
	nodes = []string{"52.233.45.243:2222", "52.175.29.87:2222", "40.69.195.111:2222", "13.65.91.243:2222"}

	c := kvservice.NewConnection(nodes)
	fmt.Printf("NewConnection returned: %v\n", c)

	t1, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t1, err)

	success, err := t1.Put("A", "3")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, v, err := t1.Get("A")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, txID, err := t1.Commit(0)
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
