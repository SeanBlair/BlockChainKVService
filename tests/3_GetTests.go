/*
Tests basic Gets.
Tx1 gets 4 non existent keys, expect successful (but empty)
Then tx1 puts the 4 keys, does not commit and gets them, expect successful with values
Then tx1 commits and gets the 4 keys, expect successfull with values.

Tx2 gets 4 keys it hasn't Put, (but Tx1 did put) expect success with previous values
Then Tx2 Puts new values in the same 4 keys, does not commit and gets those keys, expect new values.
Then Tx2 commits, gets the same updated values, Expect keyValueStore to have Tx2's new values instead
of Tx1's.

Usage:
go run 3_GetTests.go
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

	t1, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t1, err)

	// Test 1

	fmt.Println("\nTest1\n")
	fmt.Println("\n Tx1 tries to get 4 nonexistent keys (if kvnode freshly run)\n")
	
	success, v, err := t1.Get("A")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t1.Get("B")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t1.Get("C")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t1.Get("D")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)


	// Puts the 4 Keys
	fmt.Println("\nPuts the 4 keys\n")
	success, err = t1.Put("A", "AAA")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, err = t1.Put("B", "BBB")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, err = t1.Put("C", "CCC")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, err = t1.Put("D", "DDD")
	fmt.Printf("Put returned: %v, %v\n", success, err)


	fmt.Println("\n Tx1 tries to get 4 keys it just put but hasn't yet commited\n")

	success, v, err = t1.Get("A")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t1.Get("B")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t1.Get("C")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t1.Get("D")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	
	// commit
	success, txID, err := t1.Commit(1)
	fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)


	// Gets 4 keys from commited tx. 
	fmt.Println("\n Tx1 gets 4 existent keys it just commited\n")

	success, v, err = t1.Get("A")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t1.Get("B")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t1.Get("C")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t1.Get("D")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)


	// Test 2
	fmt.Println("\nTest2\n")
	t2, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t2, err)


	// Tx2 tries to get 4 keys it hasn't yet put.
	fmt.Println("\nTx2 Gets 4 keys it hasn't yet put \n")

	success, v, err = t2.Get("A")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t2.Get("B")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t2.Get("C")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t2.Get("D")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)


	// Puts the 4 Keys
	fmt.Println("\nPuts the 4 keys\n")
	success, err = t2.Put("A", "AAA_T2")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, err = t2.Put("B", "BBB_T2")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, err = t2.Put("C", "CCC_T2")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, err = t2.Put("D", "DDD_T2")
	fmt.Printf("Put returned: %v, %v\n", success, err)


	// Gets 4 keys from uncommited tx.
	fmt.Println("\nTx2 Gets the 4 keys it just Put, but hasn't yet commited\n")

	success, v, err = t2.Get("A")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t2.Get("B")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t2.Get("C")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t2.Get("D")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	
	// commit
	success, txID, err = t2.Commit(1)
	fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)


	// Gets 4 keys from commited tx.
	fmt.Println("\nTx2 Gets the 4 keys it just Put and commited\n")
	// Expect to see new values in keyValueStore

	success, v, err = t2.Get("A")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t2.Get("B")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t2.Get("C")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, v, err = t2.Get("D")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	c.Close()
}
