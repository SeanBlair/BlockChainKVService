/*
Tests aborting transactions at different times, successive calls do not succeed.

Usage:
go run 4_AbortTests.go
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

	fmt.Println("\nTest1\n")
	t1, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t1, err)

	t1.Abort()
	fmt.Println("\nAborted, successive calls should not succeed\n")

	success, err := t1.Put("A", "T1")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, v, err := t1.Get("A")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, txID, err := t1.Commit(1)
	fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)

	fmt.Println("\nTest2\n")

	t2, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t2, err)

	success, err = t2.Put("B", "T2")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	t2.Abort()
	fmt.Println("\nAborted, successive calls should not succeed\n")

	success, v, err = t2.Get("B")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, txID, err = t2.Commit(1)
	fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)

	fmt.Println("\nTest3\n")

	t3, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t3, err)

	success, err = t3.Put("B", "T2")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, v, err = t3.Get("B")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	t3.Abort()
	fmt.Println("\nAborted, successive calls should not succeed\n")

	success, txID, err = t3.Commit(1)
	fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)

	c.Close()
}
