package main

import "fmt"

// commandHelp represents a help command.
type commandHelp struct{}

// run prints the help of the GMQ Client commands.
func (cmd *commandHelp) run() error {
	printHelp()

	return nil
}

// newCommandHelp creates and returns a commandHelp.
func newCommandHelp() *commandHelp {
	return &commandHelp{}
}

// printHelp prints the help of the GMQ Client commands to standard output.
func printHelp() {
	printVersion()
	fmt.Println("Usage:")
	fmt.Printf("%-8s %s\n", cmdNameHelp, "print this help message")
}
