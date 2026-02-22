package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("args >1 ")
		return
	}

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		child()
	default:
		panic("Unknown command. Use: run <cmd>")
	}
}

func run() {
	fmt.Println("run function.")
}
func child() {
	fmt.Println("child function.")
}

// Add helper function `must` to panic on non-nil errors
func must(err error) {
	if err != nil {
		panic(err)
	}
}
