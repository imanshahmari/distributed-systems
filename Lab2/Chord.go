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
	- Change n.Successor to a list??
	- Initialize n.Successors and n.FingerTable correctly
	- Implement closest_preceeding_node and update find_successor


TODOs:
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
	Successor   []NodeData

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
	n.Successor = make([]NodeData, *r)
	n.FingerTable = make([]NodeData, 256)

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
	go stabilize(&n, ts)
	go fix_fingers(&n, tff)
	go check_predecessor(&n, tcp)

	// Handle command line commands
	commandLine(&n)
}

/***** Chord functions *****/

func find_successor(n *Node, searchId Key, returnAddress NodeAddress) {
	curr := n.Id
	succ := n.Successor[0].Id
	succAddr := n.Successor[0].Addr

	// If r is between c and s, or if successor wraps around
	if succ != "" &&
		((searchId > curr && searchId <= succ) ||
			(succ <= curr && (curr < searchId || searchId < succ))) {
		// The return- is between current- and successor- address' -> found successor
		sendMessage(returnAddress, "recieve_successor", succ, succAddr)

	} else {
		// Iteratively send find_successor to next node to continue searching
		closestPrecNode := closest_preceding_node(n, searchId)
		sendMessage(closestPrecNode.Addr, "find_successor", searchId, returnAddress)
	}

}

// search the local table for the highest predecessor of id
/*
n.closest preceding node(id)
for i = m downto 1
	if (finger[i] ∈ (n,id))
		return finger[i];
return n;
*/
func closest_preceding_node(n *Node, id Key) NodeData {
	for i := 255; i >= 0; i-- {
		finger := n.FingerTable[i]

		if n.Id < finger.Id && finger.Id > id {
			return finger
		}
	}
	// Will probably crash if it comes here, but will not come here (???) as this case is the first part of the if-statement in find_successor
	return n.NodeData
}

// Recieves the final successor directly from final node
// - Not secure at all since bad actors could send anything and we just accept
func recieve_successor(n *Node, SuccessorId Key, successorAddress NodeAddress) {
	n.Successor[0].Id = SuccessorId
	n.Successor[0].Addr = successorAddress
	fmt.Println("Recieved successor at: ", successorAddress)
}

// create a new Chord ring
func create(n *Node) {
	n.Successor[0] = n.NodeData
}

// join a Chord ring containing node n' (ja, jp)
func join(n *Node, ja *string, jp *int) {
	n.Successor[0].Addr = NodeAddress(*ja + ":" + fmt.Sprint(*jp))
	find_successor(n, n.Id, n.Addr)
	//find_successor(n.Addr, *ja+":"+fmt.Sprint(*jp), n.Addr)
}

/***** Fix ring *****/

// called periodically. verifies n’s immediate
// successor, and tells the successor about n.
/*
x = successor.predecessor;
if (x ∈ (n,successor))
successor = x;
successor.notify(n);
*/
func stabilize(n *Node, tc *int) {
	// wait tc milliseconds

}

// n' thinks it might be our predecessor.
/*
if (predecessor is nil or n' ∈ (predecessor, n))
predecessor = n';
*/
func notify(n *Node, n_prime NodeData) {
	if n.Predecessor.Addr == "" || n.Predecessor.Id == "" ||
		n.Predecessor.Id < n_prime.Id {

	}
}

// called periodically. refreshes finger table entries.
// next stores the index of the next finger to fix.
/*
next = next + 1 ;
if (next > m)
next = 1 ;
finger[next] = find successor(n + 2^(next−1) );
*/
func fix_fingers(n *Node, tff *int) {
	//for i := 0; i < 256; i++ {

	//	n.FingerTable[i] = find_successor(n, n.Id+2**(i), n.Addr)
	//}

	// wait tff milliseconds

}

// called periodically. checks whether predecessor has failed
func check_predecessor(n *Node, tcp *int) {
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
			n.Successor[0].Addr = NodeAddress(address[:len(address)-1])
			n.Successor[0].Id = hash_addr(n.Successor[0].Addr)

			print_state(n)
		case "setpredecessor", "pre":
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter address of predecessor: ")

			address, _ := reader.ReadString('\n')
			n.Predecessor.Addr = NodeAddress(address[:len(address)-1])
			n.Predecessor.Id = hash_addr(n.Predecessor.Addr)

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
	fmt.Println("$> Recieved message: ", message)

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

	fmt.Println("$> Sent message: ", fmt.Sprint(msg))
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
