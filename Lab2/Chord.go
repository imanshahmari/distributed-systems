package main

import (
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
)

/*
Done:
	- The findSuccessor always uses the hashed address, change to use id defined in terminal flag. This means changeing how the networking is done and also the comparison in findSuccessor. Do we need to store the successor id in our Node??
	- Change n.Successor to a list??
	- Initialize n.Successors and n.FingerTable correctly
	- Implement closestPreceedingNode and update findSuccessor
	- Change how the networking works to be call then response, ie. where it returns a value or a forwarding address (might be better for security later)
	- Convert all underscore_names to camelCasing


TODOs:
   - Implement stabilize (add networking method getPredecessor)
   - Implement notify (add networking method handleNotify)
   - Implement fixFingers (does findSuccessor need changes??)
   - Implement checkPredecessor  (add networking method handlePing (ie responds back with OK message if alive))
   - Pass in all flags to Chord via docker (can you do all in one env variable)
   - Update readme

   *Remember that if we need our node object n in these functions, it has to be passed in as a pointer otherwise we copy the values and it will not be changed for the rest of the functions (&n creates a pointer reference (from main()), *n uses the pointer as a value, and (n *Node) is the type to use in the function definitions)
*/

type Key string

type NodeAddress string

type Node struct {
	Id   Key
	Addr NodeAddress
}

type ThisNode struct {
	Node
	FingerTable []Node
	Predecessor Node
	Successor   []Node

	Bucket map[Key]string
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

	n := ThisNode{}
	n.Addr = NodeAddress(*a + ":" + fmt.Sprint(*p))
	n.Bucket = make(map[Key]string)
	n.Successor = make([]Node, *r)
	n.FingerTable = make([]Node, 256)

	if *i == "" {
		// If the id is not defined by comand line argument, generate it from hashing ip and port
		n.Id = hashAddress(n.Addr)
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
	go listen(&n, *p)
	fmt.Println("Chord server started on adress: ", n.Addr, " with id: ", n.Id)

	// Start a go routine for each of the steps to make the network consistent
	go stabilize(&n, ts)
	go fixFingers(&n, tff)
	go checkPredecessor(&n, tcp)

	// Handle command line commands
	commandLine(&n)
}

/***** Chord functions *****/

// create a new Chord ring
func create(n *ThisNode) {
	n.Successor[0] = n.Node
}

// join a Chord ring containing node n' (ja, jp)
func join(n *ThisNode, ja *string, jp *int) {
	n.Successor[0].Addr = NodeAddress(*ja + ":" + fmt.Sprint(*jp))
	findSuccessor(n, n.Id)
}

func findSuccessor(n *ThisNode, searchId Key) Node {
	// Try to find in this node
	succ, isRelayAddress := findSuccessorIteration(n, searchId)

	// If relay then repeat while we still get relay adresses to the next node
	for isRelayAddress {
		c, err := getFindSuccessor(succ.Addr, string(searchId))
		if err != nil {
			fmt.Println(err)
		}

		succ, isRelayAddress = c.Node, c.IsRelayAddr
	}

	// When it no longer relays we have the successor to the id
	return succ
}

/* This function only finds the immediate successor and does not handle
 * recursion/iteration as it has to be able to be called from both
 * findSuccessor and from networking calls
 */
func findSuccessorIteration(n *ThisNode, searchId Key) (Node, bool) {
	curr := n.Id
	succ := n.Successor[0].Id
	//succAddr := n.Successor[0].Addr

	// If r is between c and s, or if successor wraps around
	if succ != "" &&
		((searchId > curr && searchId <= succ) ||
			(succ <= curr && (curr < searchId || searchId < succ))) {
		// The return- is between current- and successor- address' -> found successor
		return n.Successor[0], false

	} else {
		// Iteratively send findSuccessor to next node to continue searching
		closestPrecNode := closestPrecedingNode(n, searchId)

		return closestPrecNode, true
	}
}

// search the local table for the highest predecessor of id
func closestPrecedingNode(n *ThisNode, id Key) Node {
	for i := 255; i >= 0; i-- {
		finger := n.FingerTable[i]

		if n.Id < finger.Id && finger.Id > id {
			return finger
		}
	}
	// Will probably crash if it comes here, but will not come here (???) as this case is the first part of the if-statement in findSuccessor
	return n.Node
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
func stabilize(n *ThisNode, tc *int) {
	// wait tc milliseconds

}

// n' thinks it might be our predecessor.
/*
if (predecessor is nil or n' ∈ (predecessor, n))
predecessor = n';
*/
func notify(n *ThisNode, nPrime Node) {
	/*
		if n.Predecessor.Addr == "" || n.Predecessor.Id == "" || n.Predecessor.Id < nPrime.Id {

		}
	*/
	if n.Predecessor.Addr == "" || n.Predecessor.Id == "" || (n.Predecessor.Id < nPrime.Id && nPrime.Id < n.Id) {
		n.Predecessor = nPrime

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
func fixFingers(n *ThisNode, tff *int) {
	//for i := 0; i < 256; i++ {

	//	n.FingerTable[i] = findSuccessor(n, n.Id+2**(i), n.Addr)
	//}

	// wait tff milliseconds

}

// called periodically. checks whether predecessor has failed
func checkPredecessor(n *ThisNode, tcp *int) {
	// TODO: wait tcp milliseconds
	// TODO: what to do when predecessor has failed???

	msg, err := sendMessage(n.Predecessor.Addr, HandlePing, "")
	if err != nil {
		fmt.Println("Predecessor has failed", err)
	}

	if string(msg) != "200 OK" {
		fmt.Println("Predecessor has failed", msg)
	}
}

/***** Hashing *****/

func hashAddress(a NodeAddress) Key {
	return Key(hashString(string(a)))
}

func hashString(str string) string {
	// No hashing of the empty string
	if str == "" {
		return ""
	}

	h := sha1.New()
	h.Write([]byte(str))
	sha1_hash := hex.EncodeToString(h.Sum(nil))
	return sha1_hash
}
