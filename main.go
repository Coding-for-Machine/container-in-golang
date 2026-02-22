package main

import (
	"fmt"
	"os"
	"os/exec"
)

// main is the enter point of the program.
// It checks the command-line arguments and decides whether to call run or child.
func main() {
	// If there are fewer than 2 arguments (only the program name),
	// print a message and exit.
	if len(os.Args) < 2 {
		fmt.Println("args >1 ")
		return
	}
	// Decide what to do based on the first argument (os.Args[1]).
	switch os.Args[1] {
	case "run":
		// If the first argument is "run", call the run() function (parent process).
		run()
		// If the first argument is "child", call the child() function (child process).
	case "child":
		child()
	default:
		// If the first argument is anything else, panic with an error message
		panic("Unknown command. Use: run <cmd>")
	}
}

// run is the parent process functin.
// If spawns a new child process that runs the same binary with the "child" argument.
func run() {
	//  Print the PID (process ID) of the current (parent) process.
	fmt.Printf("RUN PROSSES ID: PID=%d\n", os.Getpid())

	// Create a command that runs the current binary again:
	// /proc/self/exe refers to the current executable (this program itself).
	// "child" becomes the first argument in the child process.
	// os.Args[2:] contains all arguments after run (for example: /bin/bash -c "echo hello").
	cmd := exec.Command("/proc/self/exe", append([]string{"child"},
		os.Args[2:]...)...)

	// Connect the child process's stdin/stdout/stderr to the current terminal.
	//  This makes the child process interactive:
	// 	- Whatever you type in the terminal goes to the child.
	// 	- Whatever the child prints goes directly to the terminal.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// cmd.Run() starts the child process and waits for it to finish.
	// If there is an error, must(err) will panic and stop the program.
	must(cmd.Run())
}
func child() {
	fmt.Println("child function.")
}

// must is a helper function to check errors.
// If err is not nil, it panics and stops the program.
func must(err error) {
	if err != nil {
		panic(err)
	}
}
