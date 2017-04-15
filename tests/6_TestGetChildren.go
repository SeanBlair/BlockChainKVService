/*
Tests getting first child of each Block starting from genesis Block

Usage:
go run 6_TestGetChildren.go
*/

package main

import "../kvservice"

import (
	"fmt"
)
var count [2]int

func main() {
	var nodes []string
	nodes = []string{"52.187.214.143:2222", "52.233.32.107:2222"}
	for index, _ := range count{
		count[index] = 0
	}
	for i := range nodes {
		iterateThroughChildren(nodes, "", i)
		fmt.Println("Node", i, "Tree length=", count[i])
	}

}

func iterateThroughChildren(nodes []string, parentHash string, i int) {
	c := kvservice.NewConnection(nodes)
	// fmt.Printf("NewConnection returned: %v\n", c)

	kids := c.GetChildren(nodes[i], parentHash)
	fmt.Printf("GetChildren returned: %s\n", kids)
	c.Close()
	if len(kids) > 0 {
		for j := range kids {
			iterateThroughChildren(nodes, kids[j], i)
		}
		count[i] = count[i] + 1
	} else {
		fmt.Println("leaf")
		count[i] = count[i] + 1
		return
	}
}

