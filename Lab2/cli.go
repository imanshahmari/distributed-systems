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

		//manually invoke timed functions
		case "stabilize":
			i := 0
			stabilize(n, &i)
		case "notify":
			notify(n, n.Successor[0])
		case "checkpredecessor":
			i := 0
			checkPredecessor(n, &i)
		case "ping":
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter node to ping: ")

			address, _ := reader.ReadString('\n')
			res, err := sendMessage(NodeAddress(address[:len(address)-1]), HandlePing, "")
			if err != nil {
				fmt.Println(err)
			}

			fmt.Println(string(res))

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

	fmt.Println("Key of resource is:", id)
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
	fmt.Printf("Pred\t%s\t%s\n", n.Predecessor.Addr, n.Predecessor.Id)
	fmt.Printf("This\t%s\t%s\n", n.Addr, n.Id)

	fmt.Println("Successor list:")
	for i, succ := range n.Successor {
		fmt.Printf("%3d\t%s\t%s\n", i, succ.Addr, succ.Id)
	}

	fmt.Println("Finger table:")
	for i, succ := range n.FingerTable {
		// Don't print if empty
		if succ.Addr == "" {
			continue
		}
		fmt.Printf("2^%3d\t%s\t%s\n", i, succ.Addr, succ.Id)
	}
}
