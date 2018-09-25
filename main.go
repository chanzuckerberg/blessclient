package main

import (
	"fmt"
)

func main() {

	a := 3
	var b string
	if true {
		a, b = test()
		fmt.Printf("%d\n", a)
		fmt.Printf("%s\n", b)
	}
	fmt.Printf("%d\n", a)
	// cmd.Execute()
}

func test() (int, string) {
	return 4, "test"
}
