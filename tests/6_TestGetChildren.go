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
	nodes = []string{"52.233.45.243:2222", "52.175.29.87:2222", "40.69.195.111:2222", "13.65.91.243:2222", "51.140.126.235:2222", "52.233.190.164:2222"}

	c := kvservice.NewConnection(nodes)
	fmt.Printf("NewConnection returned: %v\n", c)

	isChild := true
	parent := ""
	count := 1
	for isChild {
		kids := c.GetChildren(nodes[0], parent)
		fmt.Printf("GetChildren returned: %s\n", kids)		
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
