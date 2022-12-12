package main

import (
	"bufio"
	"fmt"
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
		case "setbucket":
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter filename of bucket: ")
			filename, _ := reader.ReadString('\n')
			fmt.Print("Enter storing id of bucket: ")
			storingId, _ := reader.ReadString('\n')

			n.Bucket[filename[:len(filename)-1]] = Key(storingId[:len(storingId)-1])
			fallthrough
		case "printbucket":
			for file, key := range n.Bucket {
				hash := hashString(file)
				fmt.Println(hash[:5], file, "\t", key[:5])
			}
		case "findsuccessor":
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter id to find successor of: ")

			address, _ := reader.ReadString('\n')
			succ := findSuccessor(n, Key(address[:len(address)-1]))

			fmt.Println(succ.Id)

		case "hash":
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Text to hash: ")

			txt, _ := reader.ReadString('\n')
			hashed := hashString(txt[:len(txt)-1])
			fmt.Println(hashed)

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
		case "notify":
			notify(n, n.Successor[0])
		case "ping":
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter node to ping: ")

			address, _ := reader.ReadString('\n')
			res, err := sendMessage(NodeAddress(address[:len(address)-1]), HandlePing, "")
			if err != nil {
				fmt.Println("error ping", err)
			}
			fmt.Println(string(res))

		// Mandatory functions
		case "lookup", "l":
			lookup(n)
		case "storefile", "file", "f":
			storeFile(n)
		case "printstate", "p":
			printState(n)
		case "exit", "x":
			clear()
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
	path = path[:len(path)-1]

	fileHash := Key(hashString(path))
	succ := findSuccessor(n, fileHash)

	body, err := sendMessage(succ.Addr, HandleLookup, path)
	if err != nil {
		fmt.Println("error lookup", err)
	}

	storingNode := findSuccessor(n, Key(body))

	fmt.Println("Key of resource is:", fileHash)
	fmt.Println("succ:", succ.Id)
	fmt.Println("body: ", string(body))
	fmt.Println("storingNode: ", storingNode)
	//fmt.Println("Address of resource is: ", succ)
}

func storeFile(n *ThisNode) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter name of file: ")

	path, _ := reader.ReadString('\n')
	path = path[:len(path)-1]

	fileHash := Key(hashString(path))
	succ := findSuccessor(n, fileHash)

	_, err := sendMessage(succ.Addr, HandleStoreFile, path+"/"+string(n.Id))
	if err != nil {
		fmt.Println("error storefile", err)
	}
}

func printState(n *ThisNode) {
	fmt.Printf("Pred\t%s\t%s\n", n.Predecessor.Addr, n.Predecessor.Id)
	fmt.Printf("This\t%s\t%s\n", n.Addr, n.Id)

	fmt.Println("Successor list:")
	for i, succ := range n.Successor {
		fmt.Printf("%3d\t%s\t%s\n", i, succ.Addr, succ.Id)
	}

	fmt.Println("Finger table:")
	fmt.Println("2^     _0    _1    _2    _3    _4    _5    _6    _7    _8    _9")
	for i, succ := range n.FingerTable {
		// Don't print if empty
		if succ.Addr == "" {
			continue
		}
		//if i%10 == 0 {
		//	fmt.Printf("%2d_ ", i/10)
		//}
		//if len(succ.Id) > 5 {
		//	fmt.Print(succ.Id[:5] + " ")
		//} else {
		//	fmt.Printf("%5s ", succ.Id)
		//}
		//if i%10 == 9 {
		//	fmt.Printf("\n")
		//}

		fmt.Printf("2^%3d\t%s\t%s\n", i, succ.Addr, succ.Id)
	}
}

func clear() {
	cmd := exec.Command("clear") //Linux example, its tested
	cmd.Stdout = os.Stdout
	cmd.Run()

}
