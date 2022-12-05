package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

/***** Command line commands *****/

func commandLine(n *ThisNode) {
	for {
		// Read from cmd
		reader := bufio.NewReader(os.Stdin)
		//fmt.Print("Enter command: ")
		input, _ := reader.ReadString('\n')

		// Format string (remove newline and to lower case letters)
		input = strings.ToLower(input[:len(input)-1])
		//fmt.Println(input)

		switch input {
		case "setsuccessor", "succ", "s":
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter address of successor: ")

			address, _ := reader.ReadString('\n')
			n.Successor[0].Addr = NodeAddress(address[:len(address)-1])
			n.Successor[0].Id = hashAddress(n.Successor[0].Addr)

			printState(n)
		case "setpredecessor", "pre":
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter address of predecessor: ")

			address, _ := reader.ReadString('\n')
			n.Predecessor.Addr = NodeAddress(address[:len(address)-1])
			n.Predecessor.Id = hashAddress(n.Predecessor.Addr)

			printState(n)
		case "lookup", "l":
			lookup(n)
		case "storefile", "file", "f":
			storeFile(n)
		case "printstate", "p":
			printState(n)
		case "exit", "x":
			os.Exit(0)
		case "help", "h", "man":
			fallthrough
		default:
			fmt.Print(
				"setsuccessor, succ, s - asks for and sets the successor address\n",
				"setpredecessor, pre - asks for and sets the predecessor address\n",
				"lookup, l - finds the address of a resource\n",
				"storefile, file, f - stores a file in the network\n",
				"printstate, p - prints the state of the node\n",
				"exit, x - terminates the node\n",
				"help, h, man - shows this list of accepted commands\n",
			)
		}
	}
}

func lookup(n *ThisNode) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter name of file: ")

	path, _ := reader.ReadString('\n')

	id := Key(hashString(path))
	succ := findSuccessor(n, id)

	fmt.Println("Address of resource is:", succ)
}

func storeFile(n *ThisNode) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter name of file: ")

	path, _ := reader.ReadString('\n')
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
	}

	id := Key(hashString(path))
	succ := findSuccessor(n, id)

	postFile(succ.Addr, path, data)
}

func printState(n *ThisNode) {
	fmt.Println(n.Predecessor.Addr, "-> (", n.Addr, ") ->", n.Successor[0].Addr)

	p := n.Predecessor.Id
	a := n.Id
	s := n.Successor[0].Id

	if len(p) > 30 {
		p = p[:len(p)-30] + "... "
	}
	if len(a) > 30 {
		a = a[:len(a)-30] + "... "
	}
	if len(s) > 30 {
		s = s[:len(s)-30] + "... "
	}

	// Print first part of hashvalues to see if they are in order
	fmt.Println(p + " -> ( " + a + " ) -> " + s)
}
