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
func main() {
	var nodes []string
	nodes = []string{"localhost:2222"}

	c := kvservice.NewConnection(nodes)
	fmt.Printf("NewConnection returned: %v\n", c)

	isChild := true
	parent := ""
	count := 1
	for isChild {
		kids := c.GetChildren("localhost:2222", parent)
		fmt.Printf("GetChildren returned: %x\n", kids)		
		if len(kids) > 0 {
			parent = kids[0]
			count++		
		} else {
			isChild = false
		}	
	}
	fmt.Println("Test found", count, "children")
	fmt.Println("Note: this test sucks, it only returns the first child of each parent....")

	c.Close()
}
