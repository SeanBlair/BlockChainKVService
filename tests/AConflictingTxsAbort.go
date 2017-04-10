/*
For conflicting transactions abort

Scenario:
DB=[]																			DB = [A:b]
		A) newTx, Put(A, a), Get(A), wait..................................................Commit (A aborts because DB does not match original state)
				 							B) newTx, Put(A, b), Get(A), Commit (success DB matches original state)  

Usage:
go run AConflictingTxsAbort.go and immediately run BConflictingTxsAbort.go

To test the oposite scenario, change BConflictingTxsAbort.go to access different keys than AConflictingTxsAbort.go
for example "B" instead of "A". Both txs should commit.
*/

package main

import "../kvservice"

import (
	"fmt"
	"time"
)
func main() {
	var nodes []string
	nodes = []string{"198.162.33.28:2222", "198.162.33.46:2222", "198.162.33.51:2222", "198.162.33.14:2222"}

	c := kvservice.NewConnection(nodes)
	fmt.Printf("NewConnection returned: %v\n", c)

	t1, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t1, err)

	success, err := t1.Put("A", "T1")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, v, err := t1.Get("A")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	time.Sleep(time.Second * 10)
	success, txID, err := t1.Commit(0)
	fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)

	c.Close()
}
