package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

/*
Done:
	- The find_successor always uses the hashed address, change to use id defined in terminal flag. This means changeing how the networking is done and also the comparison in find_successor. Do we need to store the successor id in our Node??


TODOs:
   - Change n.Successor to a list??
   - Initialize n.Successors and n.FingerTable correctly
   - Change how the networking works to be call then response, ie. where it returns a value or a forwarding address (might be better for security later)
   - Implement stabilize (add networking method get_predecessor)
   - Implement notify (add networking method recieved_notify)
   - Implement fix_fingers (does find_successor need changes??)
   - Implement check_predecessor  (add networking method recieved_ping (ie responds back with OK message if alive))
   - Pass in all flags to Chord via docker (can you do all in one env variable)
   - Update readme

   *Remember that if we need our node object n in these functions, it has to be passed in as a pointer otherwise we copy the values and it will not be changed for the rest of the functions (&n creates a pointer reference (from main()), *n uses the pointer as a value, and (n *Node) is the type to use in the function definitions)
*/

type Key string

type NodeAddress string

type NodeData struct {
	Id   Key
	Addr NodeAddress
}

type Node struct {
	NodeData
	FingerTable []NodeData
	Predecessor NodeData
	Successor   NodeData

	Bucket map[string]string
}

// struct to decode json into
type Communication struct {
	Function string      `json:"function"`
	Id       Key         `json:"id"`
	Addr     NodeAddress `json:"addr"`
}

func main() {

	// Get the command line flags
	a := flag.String("a", "", "ip of client")
	p := flag.Int("p", 0, "port of client")
	ja := flag.String("ja", "", "ip of existing node")
	jp := flag.Int("jp", 0, "port of existing node")
	ts := flag.Int("ts", 500, "time in ms between stabilize")
	tff := flag.Int("tff", 500, "time in ms between fix fingers")
	tcp := flag.Int("tcp", 500, "time in ms between check predecessor")
	r := flag.Int("r", 1, "number of successors")
	i := flag.String("i", "", "id of client (optional)")

	flag.Parse()

	// Check for missing/wrong flags
	if *a == "" || *p == 0 || (*ja != "" && *jp == 0) || /* *ts == 0 || *tff == 0 || *tcp == 0 ||*/ *r == 0 {
		fmt.Print("Flag missing, usage:\n",
			"-a    <String> ip of client\n",
			"-p    <Number> port of client\n",
			"--ja  <String> ip of existing node (leave out to create new network)\n",
			"--jp  <Number> port of existing node\n",
			"--ts  <Number> time in ms between stabilize (default=500)\n",
			"--tff <Number> time in ms between fix fingers (default=500)\n",
			"--tcp <Number> time in ms between check predecessor (default=500)\n",
			"-r    <Number> number of successors (default=1)\n",
			"-i    <String> id of client (default is random)\n")
		return
	}

	n := Node{}
	n.Addr = NodeAddress(*a + ":" + fmt.Sprint(*p))
	n.Bucket = make(map[string]string)

	if *i == "" {
		// If the id is not defined by comand line argument, generate it from hashing ip and port
		n.Id = hash_addr(n.Addr)
	} else {
		n.Id = Key(*i)
	}

	// Create a new Chord ring if there is no --ja defined otherwise join
	if *ja == "" {
		create(&n)
	} else {
		join(&n, ja, jp)
	}

	// Start listening to incoming connections from other nodes
	go listen(&n, p)
	fmt.Println("Chord server started on adress: ", n.Addr, " with id: ", n.Id)

	// Start a go routine for each of the steps to make the network consistent
	go stabilize(ts)
	go fix_fingers(tff)
	go check_predecessor(tcp)

	// Handle command line commands
	commandLine(&n)
}

/***** Chord functions *****/

func find_successor(n *Node, searchId Key, returnAddress NodeAddress) {
	curr := n.Id
	succ := n.Successor.Id
	succAddr := n.Successor.Addr

	// If r is between c and s, or if successor wraps around
	if (searchId > curr && searchId <= succ) || (succ <= curr && (searchId > curr || searchId < succ)) {
		// The return- is between current- and successor- address' -> found successor
		sendMessage(returnAddress, "recieve_successor", succ, succAddr)

	} else if searchId == curr {
		// Not needed but more efficient as it does not need to go around the whole ring
		sendMessage(returnAddress, "recieve_successor", curr, succAddr)
	} else {
		// Iteratively send find_successor to next node to continue searching
		sendMessage(succAddr, "find_successor", searchId, returnAddress)
	}

}

// Recieves the final successor directly from final node
// - Not secure at all since bad actors could send anything and we just accept
func recieve_successor(n *Node, SuccessorId Key, successorAddress NodeAddress) {
	n.Successor.Id = SuccessorId
	n.Successor.Addr = successorAddress
	fmt.Println("Recieved successor at: ", successorAddress)
}
func create(n *Node) {
	n.Successor.Addr = n.Addr
}

func join(n *Node, ja *string, jp *int) {

	find_successor(n, n.Id, n.Addr)

	//find_successor(n.Addr, *ja+":"+fmt.Sprint(*jp), n.Addr)
}

/***** Fix ring *****/

func stabilize(tc *int) {
	// wait tc milliseconds

}

func fix_fingers(tff *int) {
	// wait tff milliseconds

}

func check_predecessor(tcp *int) {
	// wait tcp milliseconds

}

/***** Command line commands *****/

func commandLine(n *Node) {
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
			n.Successor.Addr = NodeAddress(address[:len(address)-1])

			print_state(n)
		case "setpredecessor", "pre":
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter address of predecessor: ")

			address, _ := reader.ReadString('\n')
			n.Predecessor.Addr = NodeAddress(address[:len(address)-1])

			print_state(n)
		case "lookup", "l":
			lookup()
		case "storefile", "file", "f":
			store_file()
		case "printstate", "p":
			print_state(n)
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

func lookup() {

}

func store_file() {

}

func print_state(n *Node) {
	fmt.Println(n.Predecessor, "-> (", n.Addr, ") ->", n.Successor)

	p := hash_addr(n.Predecessor.Addr)
	a := hash_addr(n.Addr)
	s := hash_addr(n.Successor.Addr)

	if p != "" {
		p = p[:len(p)-30]
	} else {
		p = "          "
	}

	// Print first part of hashvalues to see if they are in order
	fmt.Println(p, "... -> (", a[:len(a)-30], "... ) ->", s[:len(s)-30], "...")
}

/***** Networking *****/

func listen(n *Node, p *int) {
	// Listen on port p
	listner, err := net.Listen("tcp", ":"+fmt.Sprint(*p))
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		conn, err := listner.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handler(n, conn)
	}
}

func handler(n *Node, conn net.Conn) {
	defer conn.Close()

	var message Communication
	decoder := json.NewDecoder(conn)

	// Decode the message into a Communication struct
	err := decoder.Decode(&message)
	if err != nil {
		fmt.Println(err)
	}

	// Process message
	fmt.Println("Recieved message: ", message)

	switch message.Function {
	case "recieve_successor":
		// Id is the successor Id and Addr is the successor address
		recieve_successor(n, message.Id, message.Addr)
	case "find_successor":
		// Id is the searchId and Addr is the returnAddress
		find_successor(n, message.Id, message.Addr)

	}
}

func sendMessage(address NodeAddress, function string, Id Key, Addr NodeAddress) {
	msg := Communication{
		Function: function,
		Id:       Id,
		Addr:     Addr,
	}

	// Dial up the node at address
	conn, err := net.Dial("tcp", string(address))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer conn.Close()

	// Encode message as json bytes
	data, err := json.Marshal(msg)
	if err != nil {
		fmt.Println(err)
	}

	// Send message
	conn.Write(data)

	fmt.Println("Sent message: ", fmt.Sprint(msg))
}

/***** Hashing *****/

/*func hash(str string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(str))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}*/

func hash_addr(a NodeAddress) Key {
	return Key(hash_string(string(a)))
}

func hash_string(str string) string {
	// No hashing of the empty string
	if str == "" {
		return ""
	}

	h := sha1.New()
	h.Write([]byte(str))
	sha1_hash := hex.EncodeToString(h.Sum(nil))
	return sha1_hash
}
