/*
For conflicting transactions abort

Scenario:
DB=[]																			DB = [A:b]
		A) newTx, Put(A, a), Get(A), wait..................................................Commit (A aborts because DB does not match original state)
				 							B) newTx, Put(A, b), Get(A), Commit (success DB matches original state)  

Usage:
Start kvnode system, so it has empty DB
go run AConflictingTxsAbort.go and then run BConflictingTxsAbort.go
If B commits before A, (expected), then A should return abort on call to commit

To test the oposite scenario, change BConflictingTxsAbort.go to access different keys than AConflictingTxsAbort.go
for example "B" instead of "A". Both txs should commit.
*/

package main

import "../kvservice"

import (
	"fmt"
)
func main() {
	var nodes []string
	nodes = []string{"52.233.45.243:2222", "52.175.29.87:2222", "40.69.195.111:2222", "13.65.91.243:2222", "51.140.126.235:2222", "52.233.190.164:2222"}

	c := kvservice.NewConnection(nodes)
	fmt.Printf("NewConnection returned: %v\n", c)

	t1, err := c.NewTX()
	fmt.Printf("NewTX returned: %v, %v\n", t1, err)

	success, err := t1.Put("A", "T2")
	fmt.Printf("Put returned: %v, %v\n", success, err)

	success, v, err := t1.Get("A")
	fmt.Printf("Get returned: %v, %v, %v\n", success, v, err)

	success, txID, err := t1.Commit(0)
	fmt.Printf("Commit returned: %v, %v, %v\n", success, txID, err)

	c.Close()
}
