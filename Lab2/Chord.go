package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"net"
	"os"
	"strings"
)

type Key big.Int

type NodeAddress string

type Node struct {
	Id          string
	Address     string
	FingerTable map[string]string
	Predecessor string
	Successor   string

	Bucket map[string]string
}

// struct to unmarshal json into
type Communication struct {
	Function string `json:"function"`
	Var1     string `json:"var1"`
	Var2     string `json:"var2"`
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

	n := Node{
		Address: *a + ":" + fmt.Sprint(*p),
	}

	if *i == "" {
		// If the id is not defined by comand line argument, generate it from hashing ip and port
		n.Id = hash_string(n.Address)
	} else {
		n.Id = *i
	}

	// Create a new Chord ring if there is no --ja defined
	if *ja == "" {
		// Create a new ring
		n.Successor = n.Address
	} else {
		// Join network
		find_successor(n.Address, *ja+":"+fmt.Sprint(*jp), n.Address)
	}

	go listen(&n, p)
	fmt.Println("Chord server started on adress: ", n.Address, " with id: ", n.Id)

	// Start a go routine for each of the steps to make the network consistent
	go stabilize(ts)
	go fix_fingers(tff)
	go check_predecessor(tcp)

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
			n.Successor = address[:len(address)-1]

			print_state(&n)
		case "setpredecessor", "pre":
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Enter address of predecessor: ")

			address, _ := reader.ReadString('\n')
			n.Predecessor = address[:len(address)-1]

			print_state(&n)
		case "lookup", "l":
			lookup()
		case "storefile", "file", "f":
			store_file()
		case "printstate", "p":
			print_state(&n)
		case "exit", "x":
			return
		}
	}

}

/***** Find successor *****/

func find_successor(currentAddress string, successorAddress string,
	returnAddress string) {

	c := hash_string(currentAddress)
	s := hash_string(successorAddress)
	r := hash_string(returnAddress)

	// TODO: Fix case where new node is between largest and smallest node (wrap around the circle)

	// If r is between c and s, or if successor wraps around
	if (r > c && r <= s) || (s <= c && r > c) {
		// The return- is between current- and successor- address' -> found successor
		sendMessage(returnAddress, "recieve_successor", successorAddress, "")
	} else {
		// Iteratively send find_successor to next node to continue searching
		sendMessage(successorAddress, "find_successor", returnAddress, "")
	}

}

// Recieves the final successor directly from final node
// - Not secure at all since bad actors could send anything and we just accept
func recieve_successor(n *Node, successorAddress string) {
	n.Successor = successorAddress
	fmt.Println("Recieved successor at: ", successorAddress)
}

/***** Command line commands *****/

func lookup() {
	return
}

func store_file() {
	return
}

func print_state(n *Node) {
	fmt.Println(n.Predecessor, "-> (", n.Address, ") ->", n.Successor)
	fmt.Println(hash_string(n.Predecessor)[:10], "... -> (", hash_string(n.Address)[:10], "... ) ->", hash_string(n.Successor)[:10], "...")
}

/***** Fix ring *****/

func stabilize(tc *int) {
	// wait tc milliseconds
	return
}

func fix_fingers(tff *int) {
	// wait tff milliseconds
	return
}

func check_predecessor(tcp *int) {
	// wait tcp milliseconds
	return
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
		// Var1 is the successorAddress
		recieve_successor(n, message.Var1)
	case "find_successor":
		// Var1 is the returnAddress
		find_successor(n.Address, n.Successor, message.Var1)

	}
}

func sendMessage(address string, function string, var1 string, var2 string) {
	msg := Communication{
		Function: function,
		Var1:     var1,
		Var2:     var2,
	}

	// Dial up the node at address
	conn, err := net.Dial("tcp", address)
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

func hash(str string) *big.Int {
	hasher := sha1.New()
	hasher.Write([]byte(str))
	return new(big.Int).SetBytes(hasher.Sum(nil))
}

func hash_string(str string) string {
	h := sha1.New()
	h.Write([]byte(str))
	sha1_hash := hex.EncodeToString(h.Sum(nil))
	return sha1_hash
}
