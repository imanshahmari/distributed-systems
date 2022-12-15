package main

import (
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/big"
	"time"
)

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

	Bucket          map[string]Key
	FilesOnThisNode map[string]bool
}

var (
	logFunctionCalls  = false
	logFunctionCalls2 = false
)

func main() {

	// Get the command line flags
	a := flag.String("a", "localhost", "ip of client")
	p := flag.Int("p", 0, "port of client")
	ja := flag.String("ja", "localhost", "ip of existing node")
	jp := flag.Int("jp", 0, "port of existing node")
	ts := flag.Int("ts", 1000, "time in ms between stabilize")
	tff := flag.Int("tff", 5000, "time in ms between fix fingers")
	tcp := flag.Int("tcp", 1000, "time in ms between check predecessor")
	tfs := flag.Int("tfs", 1000, "time in ms between fix successor list")
	r := flag.Int("r", 4, "number of successors")
	i := flag.String("i", "", "id of client (optional)")

	testSuccessor := flag.String("testSuccessor", "", "set successor address manually")
	testPre := flag.String("testPre", "", "set predecessor address manually")

	flag.Parse()

	// Check for missing/wrong flags
	if /*a == "" ||*/ *p == 0 || (*ja != "localhost" && *jp == 0) || *ts == 0 || *tff == 0 || *tcp == 0 || *tfs == 0 || *r == 0 {
		fmt.Print("Flag missing, usage:\n",
			"-a    <String> ip of client\n",
			"-p    <Number> port of client\n",
			"--ja  <String> ip of existing node (leave out to create new network)\n",
			"--jp  <Number> port of existing node\n",
			"--ts  <Number> time in ms between stabilize (default=1000)\n",
			"--tff <Number> time in ms between fix fingers (default=5000)\n",
			"--tcp <Number> time in ms between check predecessor (default=1000)\n",
			"--tfs <Number> time in ms between fix successor list (default=10000)\n",
			"-r    <Number> number of successors (default=1)\n",
			"-i    <String> id of client (default is random)\n")
		return
	}

	n := ThisNode{}
	n.Addr = NodeAddress(*a + ":" + fmt.Sprint(*p))
	n.Bucket = make(map[string]Key)
	n.Successor = make([]Node, *r)
	n.FingerTable = make([]Node, 160)
	n.FilesOnThisNode = make(map[string]bool)

	if *i == "" {
		// If the id is not defined by comand line argument, generate it from hashing ip and port
		n.Id = Key(fmt.Sprintf("%040s", hashAddress(n.Addr)))
	} else {
		n.Id = Key(fmt.Sprintf("%040s", *i))
	}

	// Create a new Chord ring if there is no --jp defined otherwise join
	if *jp == 0 {
		create(&n)
	} else {
		join(&n, ja, jp)
	}

	// Manually set successor and id from terminal
	if *testSuccessor != "" {
		n.Successor[0].Addr = NodeAddress(*testSuccessor)
		n.Successor[0].Id = hashAddress(n.Successor[0].Addr)
	}
	if *testPre != "" {
		n.Predecessor.Addr = NodeAddress(*testPre)
		n.Predecessor.Id = hashAddress(n.Predecessor.Addr)
	}

	// Start listening to incoming connections from other nodes
	go listen(&n, *p)
	log.Println("Chord server started on adress: ", n.Addr, " with id: ", n.Id)

	// Start a go routine for each of the steps to make the network consistent
	go stabilize(&n, ts)
	go fixFingers(&n, tff)
	go checkPredecessor(&n, tcp)
	go fixSuccessorList(&n, tfs)

	// Handle command line commands
	commandLine(&n)
}

/***** Chord functions *****/

// create a new Chord ring
func create(n *ThisNode) {
	if logFunctionCalls {
		log.Println("Creating network")
	}
	addSuccessor(n, n.Node)
	fillTables(n, n.Node)
}

// join a Chord ring containing node n' (ja, jp)
func join(n *ThisNode, ja *string, jp *int) {
	if logFunctionCalls {
		log.Println("Joining network at ", *ja+":"+fmt.Sprint(*jp))
	}

	// Set default adress to the adress supplied by the flags (will be replaced)
	n.Successor[0].Addr = NodeAddress(*ja + ":" + fmt.Sprint(*jp))
	n.FingerTable[0].Addr = NodeAddress(*ja + ":" + fmt.Sprint(*jp))

	succ := findSuccessor(n, n.Id)
	addSuccessor(n, succ)
	fillTables(n, succ)

	notify(n, succ)
}

func fillTables(n *ThisNode, entry Node) {
	for i := 0; i < len(n.FingerTable); i++ {
		n.FingerTable[i] = entry
	}
	//for i := 0; i < len(n.Successor); i++ {
	//	n.Successor[i] = entry
	//}
}

func findSuccessor(n *ThisNode, searchId Key) Node {
	if logFunctionCalls2 {
		log.Println("Find successor ", searchId)
	}
	// Try to find in this node
	succ, isRelayAddress := findSuccessorIteration(n, searchId)

	// If relay then repeat while we still get relay adresses to the next node
	for isRelayAddress {
		c, err := getFindSuccessor(succ.Addr, string(searchId))
		if err != nil {
			log.Println("error findSuccessor", err)
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
	succ := getSuccessor(n)

	// If r is between c and s, or if successor wraps around
	if succ.Id != "" && isCircleBetweenIncludingEnd(searchId, curr, succ.Id) {
		// The return- is between current- and successor- address' -> found successor
		return succ, false

	} else if succ.Id == "" {
		// If successor id is empty we have to send request to the address of
		// successor as the closestPrecNode will not work unless id is defined
		return succ, true
	} else {
		// Iteratively send findSuccessor to next node to continue searching
		return closestPrecedingNode(n, searchId)
		//return succ, true
	}
}

// search the local table for the highest predecessor of id
func closestPrecedingNode(n *ThisNode, id Key) (Node, bool) {

	// init this to n.node to allow isCircleBetween to work for successor list
	returnFinger := n.Node

	// Check the fingertable
	for i := (len(n.FingerTable) - 1); i >= 0; i-- {
		finger := n.FingerTable[i]
		//fmt.Println(i, finger.Id, id)
		if isCircleBetween(finger.Id, n.Id, id) {
			// Check if the node is alive otherwise continue looking
			_, err := sendMessage(finger.Addr, HandlePing, "")
			if err != nil {
				//log.Println(err)
				continue
			}
			returnFinger = finger
			break
		}
	}

	// Check if there is a better (alive) node than the found finger
	for _, successorNode := range n.Successor {
		if isCircleBetween(successorNode.Id, returnFinger.Id, id) {
			_, err := sendMessage(successorNode.Addr, HandlePing, "")
			if err != nil {
				//log.Println(err)
				continue
			}
			return successorNode, true
		}
	}

	// return the finger if it was found
	if returnFinger.Addr != n.Addr {
		return returnFinger, true
	}

	return n.Node, false
}

/***** Fix ring *****/

// called periodically. verifies n’s immediate
// successor, and tells the successor about n.
func stabilize(n *ThisNode, tc *int) {
	for {
		if logFunctionCalls {
			log.Println("Stabilizing")
		}

		succ := getSuccessor(n)
		x, err := getPredecessor(succ.Addr)
		if err != nil {
			log.Println("error stabilize", err)
		}
		if x.Addr == "" {
			notify(n, x)
			continue
		}

		// x in (n, successor) or this node is its own successor
		if isCircleBetween(x.Id, n.Id, succ.Id) ||
			(x.Id != "" && n.Id == succ.Id) {
			addSuccessor(n, x)
			notify(n, x)
		}
		time.Sleep(time.Duration(*tc) * time.Millisecond)
	}
}

// Send request to our successor to tell that we might be its predecessor
func notify(n *ThisNode, succ Node) {
	if logFunctionCalls {
		log.Println("Notifying")
	}
	sendMessage(succ.Addr, HandleNotify, string(n.Addr)+"/"+string(n.Id))
	// replicate bucket
	replicateThisBucketElems(n)
	// replicate files
	postReplicate(n, false)
}

// called periodically. refreshes finger table entries.
// next stores the index of the next finger to fix.
func fixFingers(n *ThisNode, tff *int) {
	for {
		if logFunctionCalls {
			log.Println("Fixing fingers")
		}

		for i := 0; i < len(n.FingerTable); i++ {
			fingerTableEntry := jump(n.Id, i)

			// TODO isCercleBetween correct??
			if i > 0 && isCircleBetween(fingerTableEntry, n.Id, n.FingerTable[i-1].Id) {
				// Avoid unneccesary requests
				n.FingerTable[i] = n.FingerTable[i-1]
			} else {
				n.FingerTable[i] = findSuccessor(n, fingerTableEntry)
			}
		}
		time.Sleep(time.Duration(*tff) * time.Millisecond)
	}
}

func fixSuccessorList(n *ThisNode, tfs *int) {
	for {
		if logFunctionCalls {
			log.Println("Fixing successors")
		}

		//Fixing the successor list
		for i := 0; i < len(n.Successor); i++ {
			if i == 0 {
				//succ := findSuccessor(n, jump(n.Id, 1))
				//n.Successor[i] = succ
			} else {
				succ := findSuccessor(n, jump(n.Successor[i-1].Id, 1))
				n.Successor[i] = succ
			}
		}
		//fmt.Println("fix", n.Successor[0].Addr)
		time.Sleep(time.Duration(*tfs) * time.Millisecond)
	}
}

// Returns the first alive node in successor list
func getSuccessor(n *ThisNode) Node {
	for i, succ := range n.Successor {
		msg, _ := sendMessage(succ.Addr, HandlePing, "")

		if string(msg) == "200 OK" {
			// Move successors to have the first alive one at 0
			removeSuccessor(n, i)
			return succ
		}
	}

	// If we get here we are alone, set successor 0 to self
	//removeSuccessor(n, len(n.Successor))
	return n.Node
}

// adds successor to front of list
func addSuccessor(n *ThisNode, succ Node) {
	// Move Successor list back one step to add x
	for i := len(n.Successor) - 1; i > 0; i-- {
		n.Successor[i-1] = n.Successor[i]
	}
	n.Successor[0] = succ
	//fmt.Println("add", n.Successor[0].Addr)
}

func removeSuccessor(n *ThisNode, i int) {
	for j := 0; j < i; j++ {
		if i+j < len(n.Successor) {
			n.Successor[j] = n.Successor[i+j]
			n.Successor[i+j] = Node{}
		} else {
			n.Successor[j] = Node{}
		}
	}

	// If first node is empty we are alone, then put ourself as successor
	if n.Successor[0].Addr == "" {
		n.Successor[0] = n.Node
	}
}

// n.id + 2^i where i is the index of fingertable
func jump(id Key, fingerentry int) Key {
	const keySize = sha1.Size * 8 // this is 160 for some reason

	var two = big.NewInt(2)
	var hashMod = new(big.Int).Exp(big.NewInt(2), big.NewInt(keySize), nil)

	var n big.Int
	n.SetString(string(id), 16)

	fingerentryminus1 := big.NewInt(int64(fingerentry) - 1)
	jump := new(big.Int).Exp(two, fingerentryminus1, nil)
	sum := new(big.Int).Add(&n, jump)

	return Key(BigIntToHexStr(new(big.Int).Mod(sum, hashMod)))
}

// called periodically. checks whether predecessor has failed
func checkPredecessor(n *ThisNode, tcp *int) {
	for {
		if logFunctionCalls {
			log.Println("Checking predecessor")
		}

		if n.Predecessor.Addr == "" {
			continue
		}

		msg, err := sendMessage(n.Predecessor.Addr, HandlePing, "")
		if err != nil || string(msg) != "200 OK" {
			log.Println("Predecessor has failed", err)
			// This node is now responsible for the elements
			// replicate so that there exists two replicas of bucket elements
			replicatePredecessorsBucketElems(n)
			// Replicate files and take responsibility from failed predecessor
			postReplicate(n, true)

			// Set the predecessor to nil to not get stuck in loop
			n.Predecessor = Node{}
		}
		time.Sleep(time.Duration(*tcp) * time.Millisecond)
	}
}

func replicateSingleBucketElem(n *ThisNode, filename string, storingNode Key) {
	_, err := sendMessage(getSuccessor(n).Addr, HandleReplicate, filename+"/"+string(storingNode))
	if err != nil {
		log.Println("error replicate single", err)
	}
}

func replicateThisBucketElems(n *ThisNode) {
	pred := string(n.Predecessor.Id)

	for filename, storingNode := range n.Bucket {
		id := hashString(filename)
		if id > pred {
			replicateSingleBucketElem(n, filename, storingNode)
		}
	}
}

func replicatePredecessorsBucketElems(n *ThisNode) {
	for filename, storingNode := range n.Bucket {
		replicateSingleBucketElem(n, filename, storingNode)

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

// is n in (pre, succ)
func isCircleBetween(n Key, pre Key, succ Key) bool {
	return (
	// If between consecutive numbers
	(pre < n && n < succ) ||
		// Connecting end of circle with beginning
		(succ < pre &&
			// Between largest id and 0 or Between 0 and smallest id
			(pre < n || n < succ)))
}

// is n in (pre, succ]
func isCircleBetweenIncludingEnd(n Key, pre Key, succ Key) bool {
	return (pre < n && n <= succ) ||
		(succ <= pre && (pre < n || n < succ))
}

func BigIntToHexStr(bigInt *big.Int) string {
	return fmt.Sprintf("%x", bigInt)
}

func BigIntToStr(bigInt *big.Int) string {
	return fmt.Sprintf("%v", bigInt)
}
