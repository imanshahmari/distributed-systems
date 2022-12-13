package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
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

		inputs := strings.Split(input, " ")

		switch inputs[0] {
		// Mandatory functions
		case "lookup", "l":
			lookup(n, &inputs)
		case "storefile", "file", "f":
			storeFile(n, &inputs)
		case "printstate":
			printState(n,
				// How to print the successors and fingers
				!(len(inputs) > 1 && inputs[1] == "false"),
				!(len(inputs) > 1 && inputs[1] == "list"))
		case "p":
			printState(n, false, false)

		// Manual modifications
		case "setsuccessor", "succ", "s":
			if len(inputs) < 2 {
				addInput(&inputs, "Enter address of successor: ")
			}
			n.Successor[0].Addr = NodeAddress(inputs[1])
			n.Successor[0].Id = hashAddress(n.Successor[0].Addr)
			printState(n, false, true)
		case "setpredecessor", "pre":
			if len(inputs) < 2 {
				addInput(&inputs, "Enter address of predecessor: ")
			}
			n.Predecessor.Addr = NodeAddress(inputs[1])
			n.Predecessor.Id = hashAddress(n.Predecessor.Addr)
			printState(n, false, true)
		case "setbucket":
			if len(inputs) < 3 {
				addInput(&inputs, "Enter filename of bucket: ")
				addInput(&inputs, "Enter storing id of bucket: ")
			}
			n.Bucket[inputs[1]] = Key(inputs[2])
			fallthrough
		case "printbucket":
			for file, key := range n.Bucket {
				hash := hashString(file)
				fmt.Println(hash[:5], file, "\t", key[:5])
			}
		case "findsuccessor":
			if len(inputs) < 2 {
				addInput(&inputs, "Enter id to find successor of: ")
			}
			succ := findSuccessor(n, Key(inputs[1]))
			fmt.Println(succ.Id)
		case "hash":
			if len(inputs) < 2 {
				addInput(&inputs, "Text to hash: ")
			}
			hashed := hashString(inputs[1])
			fmt.Println(hashed)
		case "notify":
			notify(n, n.Successor[0])
		case "ping":
			if len(inputs) < 2 {
				addInput(&inputs, "Enter node to ping: ")
			}
			res, err := sendMessage(NodeAddress(inputs[1]), HandlePing, "")
			if err != nil {
				log.Println("error ping", err)
			}
			fmt.Println(string(res))

		//manually invoke timed functions
		case "stabilize":
			i := 1000000000
			stabilize(n, &i)
		case "checkpredecessor":
			i := 1000000000
			checkPredecessor(n, &i)
		case "fixfingers":
			i := 1000000000
			fixFingers(n, &i)

		case "clear":
			clear()
		case "exit", "x":
			clear()
			fallthrough
		case "quit", "q":
			os.Exit(0)
		case "help", "h", "man":
			fallthrough
		default:
			fmt.Print(
				"lookup, l - finds the address of a resource\n",
				"storefile, file, f - stores a file in the network\n",
				"printstate, p - prints the state of the node (args: 'false' to skip printing the lists, 'list' to show fingers as list\n",
				"printbucket - prints content of bucket and hash vals",
				"setsuccessor, succ, s - asks for and sets the successor address\n",
				"setpredecessor, pre - asks for and sets the predecessor address\n",
				"setbucket - add file manually to bucket",
				"findsuccessor - finds successor to an id",
				"hash - hashes the input",
				"notify - run notify",
				"ping - pings an address",
				"stabilize - run stabilize",
				"checkpredecessor - run checkPredecessor",
				"fixfingers - run fixFingers",
				"exit, x - terminates the node and clears terminal\n",
				"quit, q - terminates the node without clearing the terminal\n",
				"help, h, man - shows this list of accepted commands\n",
			)
		}
	}
}

func lookup(n *ThisNode, inputs *[]string) {
	if len(*inputs) < 2 {
		addInput(inputs, "Enter name of file: ")
	}

	fileHash := Key(hashString((*inputs)[1]))
	succ := findSuccessor(n, fileHash)

	body, err := sendMessage(succ.Addr, HandleLookup, (*inputs)[1])
	if err != nil {
		log.Println("error lookup", err)
	}

	storingNode := findSuccessor(n, Key(body))

	fmt.Println("Key of resource is:", fileHash)
	fmt.Println("bucket node:", succ.Id)
	//fmt.Println("body: ", string(body))
	fmt.Println("storingNode: ", storingNode)
	//fmt.Println("Address of resource is: ", succ)
}

func storeFile(n *ThisNode, inputs *[]string) {
	if len(*inputs) < 2 {
		addInput(inputs, "Enter name of file: ")
	}
	fileHash := Key(hashString((*inputs)[1]))
	succ := findSuccessor(n, fileHash)

	_, err := sendMessage(succ.Addr, HandleStoreFile, (*inputs)[1]+"/"+string(n.Id))
	if err != nil {
		log.Println("error storefile", err)
	}
}

func printState(n *ThisNode, printAll bool, printTable bool) {

	var predId, thisId Key
	if printAll {
		predId = n.Predecessor.Id
		thisId = n.Id
	} else {
		predId = n.Predecessor.Id[:5]
		thisId = n.Id[:5]
	}

	fmt.Printf("Pred\t%s\t%s\n", n.Predecessor.Addr, predId)
	fmt.Printf("This\t%s\t%s\n", n.Addr, thisId)

	fmt.Println("Successor list:")
	for i, succ := range n.Successor {
		// Only show first
		if !printAll {
			fmt.Printf("%3d\t%s\t%s\n", i, succ.Addr, succ.Id[:5])
			break
		}

		fmt.Printf("%3d\t%s\t%s\n", i, succ.Addr, succ.Id)
	}

	if printAll {
		fmt.Println("Finger table:")
		if printTable {
			fmt.Println("2^     _0    _1    _2    _3    _4    _5    _6    _7    _8    _9")
		}
		for i, succ := range n.FingerTable {
			// Don't print if empty
			if succ.Addr == "" {
				continue
			}
			// Choose to print as table or list
			if printTable {
				if i%10 == 0 {
					fmt.Printf("%2d_ ", i/10)
				}
				if len(succ.Id) > 5 {
					fmt.Print(succ.Id[:5] + " ")
				} else {
					fmt.Printf("%5s ", succ.Id)
				}
				if i%10 == 9 {
					fmt.Printf("\n")
				}
			} else {
				fmt.Printf("2^%3d\t%s\t%s\n", i, succ.Addr, succ.Id)
			}
		}
	}
}

func addInput(inputs *[]string, txt string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(txt)

	input, _ := reader.ReadString('\n')
	*inputs = append(*inputs, input[:len(input)-1])
}

func clear() {
	cmd := exec.Command("clear") //Linux example, its tested
	cmd.Stdout = os.Stdout
	cmd.Run()

}
