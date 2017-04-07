/*
Tests basic Puts.
First tx puts 4 different key/values and commits.

Second tx puts 4 different values on same key.

Usage:
go run 2_PutTests.go
*/

package main

import "../kvservice"

import (
	"fmt"
)
func main() {
	var nodes []string
	nodes = []string{"localhost:2224", "localhost:2226"}

	c := kvservice.NewConnection(nodes)
	fmt.Printf("NewConnection returned: %v\n", c)

	
	// t1 Puts 4 different keys/values and commits
	// Expect to see Keys A, B, C, D with values in keyValueStore
	fmt.Println("\nTest1\n")
	fmt.Println("t1 Puts 4 different keys/values and commits\n")	
	
	t1, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t1, err)

	success, err := t1.Put("A", "T1A")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, err = t1.Put("B", "T1B")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, err = t1.Put("C", "T1C")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, err = t1.Put("D", "T1D")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, txID, err := t1.Commit(1)
	fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)

	// t2 Puts 4 different values on same key and commits
	// Expect to see only Key "E" with value "T2_4" in keyValueStore
	fmt.Println("\nTest2")
	fmt.Println("t2 Puts 4 different values on same key and commits\n")

	t2, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t2, err)

	success, err = t2.Put("E", "T2_1")
	fmt.Printf("Put returned: %v, %v\n", success, err)


	success, err = t2.Put("E", "T2_2")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, err = t2.Put("E", "T2_3")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, err = t2.Put("E", "T2_4")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, txID, err = t2.Commit(1)
	fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)

	c.Close()
}
