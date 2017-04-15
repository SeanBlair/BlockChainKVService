package main
import "../kvservice"

import (
	"fmt"
)
func main() {
	var nodes []string
	nodes = []string{"52.187.214.143:2222", "52.233.32.107:2222"}

	c := kvservice.NewConnection(nodes)
	fmt.Printf("NewConnection returned: %v\n", c)

	for {
		t1, err := c.NewTX()
		fmt.Printf("NewTX returned: %v, %v\n", t1, err)
	}
}
