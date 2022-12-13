package main

import (
	"crypto/sha1"
	"encoding/hex"
	"flag"
	"fmt"
	"math/big"
	"time"
)

/*
TODOs:
	- Change http post to sftp
	- Make secure with https
	- Replication of files
	- Encrypt files before sending
	- Update readme
	- Fix docker to create multiple servers with different ips
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

	Bucket map[string]Key
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
	tff := flag.Int("tff", 1000, "time in ms between fix fingers")
	tcp := flag.Int("tcp", 1000, "time in ms between check predecessor")
	r := flag.Int("r", 4, "number of successors")
	i := flag.String("i", "", "id of client (optional)")

	testSuccessor := flag.String("testSuccessor", "", "set successor address manually")
	testPre := flag.String("testPre", "", "set predecessor address manually")

	flag.Parse()

	// Check for missing/wrong flags
	if /*a == "" ||*/ *p == 0 || (*ja != "localhost" && *jp == 0) || *ts == 0 || *tff == 0 || *tcp == 0 || *r == 0 {
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
	n.Bucket = make(map[string]Key)
	n.Successor = make([]Node, *r)
	n.FingerTable = make([]Node, 160)

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
	if logFunctionCalls {
		fmt.Println(time.Now().Format("15:04:05:0001"), "Creating network")
	}
	addSuccessor(n, n.Node)
	fillFingerTable(n, n.Node)
}

// join a Chord ring containing node n' (ja, jp)
func join(n *ThisNode, ja *string, jp *int) {
	if logFunctionCalls {
		fmt.Println(time.Now().Format("15:04:05:0001"), "Joining network at ", *ja+":"+fmt.Sprint(*jp))
	}

	// Set default adress to the adress supplied by the flags (will be replaced)
	n.Successor[0].Addr = NodeAddress(*ja + ":" + fmt.Sprint(*jp))
	n.FingerTable[0].Addr = NodeAddress(*ja + ":" + fmt.Sprint(*jp))

	succ := findSuccessor(n, n.Id)
	addSuccessor(n, succ)
	fillFingerTable(n, succ)

	notify(n, succ)
}

func fillFingerTable(n *ThisNode, entry Node) {
	for i := 0; i < len(n.FingerTable); i++ {
		n.FingerTable[i] = entry
	}
}

func findSuccessor(n *ThisNode, searchId Key) Node {
	if logFunctionCalls2 {
		fmt.Println(time.Now().Format("15:04:05:0001"), "Find successor ", searchId)
	}
	// Try to find in this node
	succ, isRelayAddress := findSuccessorIteration(n, searchId)

	// If relay then repeat while we still get relay adresses to the next node
	for isRelayAddress {
		c, err := getFindSuccessor(succ.Addr, string(searchId))
		if err != nil {
			fmt.Println("error findSuccessor", err)
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
	if succ != "" && isCircleBetweenIncludingEnd(searchId, curr, succ) {
		// The return- is between current- and successor- address' -> found successor
		return n.Successor[0], false

	} else if succ == "" {
		// If successor id is empty we have to send request to the address of
		// successor as the closestPrecNode will not work unless id is defined
		return n.Successor[0], true
	} else {
		// Iteratively send findSuccessor to next node to continue searching
		return closestPrecedingNode(n, searchId)
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
				//fmt.Println(err)
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
				//fmt.Println(err)
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
		time.Sleep(time.Duration(*tc) * time.Millisecond)
		if logFunctionCalls {
			fmt.Println(time.Now().Format("15:04:05:0001"), "Stabilizing")
		}

		succ := getSuccessor(n)
		x, err := getPredecessor(succ.Addr)
		if err != nil {
			fmt.Println("error stabilize", err)
		}
		if x.Addr == "" {
			continue
		}

		// x in (n, successor) or this node is its own successor
		if isCircleBetween(x.Id, n.Id, succ.Id) ||
			(x.Id != "" && n.Id == succ.Id) {
			addSuccessor(n, x)
			notify(n, x)
		}
	}
}

// Send request to our successor to tell that we might be its predecessor
func notify(n *ThisNode, succ Node) {
	if logFunctionCalls {
		fmt.Println(time.Now().Format("15:04:05:0001"), "Notifying")
	}
	sendMessage(succ.Addr, HandleNotify, string(n.Addr)+"/"+string(n.Id))
}

// called periodically. refreshes finger table entries.
// next stores the index of the next finger to fix.
func fixFingers(n *ThisNode, tff *int) {
	for {
		time.Sleep(time.Duration(*tff) * time.Millisecond)
		if logFunctionCalls {
			fmt.Println(time.Now().Format("15:04:05:0001"), "Fixing fingers")
		}

		for i := 0; i < len(n.FingerTable); i++ {
			//x := jump(n.Addr, i)
			fingerTableEntry := jump(n.Id, i)

			n.FingerTable[i] = findSuccessor(n, fingerTableEntry)
		}
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
	removeSuccessor(n, len(n.Successor))
	return n.Node
}

// adds successor to front of list
func addSuccessor(n *ThisNode, succ Node) {
	// Move Successor list back one step to add x
	for i := len(n.Successor) - 1; i > 0; i-- {
		n.Successor[i-1] = n.Successor[i]
	}
	n.Successor[0] = succ
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

func jump2(id Key, fingerentry int) Key {

	var n big.Int
	n.SetString(string(id), 16)
	/*
		fmt.Println("BEFORE")
		fmt.Println(BigIntToStr(&n))
		fmt.Println(BigIntToHexStr(&n))
	*/

	finger := big.NewInt(int64(fingerentry))
	sum := new(big.Int).Add(&n, finger)
	/*
		fmt.Println("AFTER")
		fmt.Println(BigIntToStr(sum))
		fmt.Println(BigIntToHexStr(sum))
	*/

	return Key(BigIntToHexStr(sum))
}

// called periodically. checks whether predecessor has failed
func checkPredecessor(n *ThisNode, tcp *int) {
	for {
		time.Sleep(time.Duration(*tcp) * time.Millisecond)
		if logFunctionCalls {
			fmt.Println(time.Now().Format("15:04:05:0001"),
				"Checking predecessor")
		}

		if n.Predecessor.Addr == "" {
			continue
		}

		msg, err := sendMessage(n.Predecessor.Addr, HandlePing, "")
		if err != nil || string(msg) != "200 OK" {
			fmt.Println("Predecessor has failed", err)
			// This node is now responsible for the elements
			// replicate so that there exists two replicas of bucket elements
			replicatePredecessorsBucketElems(n)

			// Set the predecessor to nil to not get stuck in loop
			n.Predecessor = Node{}
		}
	}
}

func replicateSingleBucketElem(n *ThisNode, filename string, storingNode Key) {
	_, err := sendMessage(getSuccessor(n).Addr, HandleReplicate, filename+"/"+string(storingNode))
	if err != nil {
		fmt.Println("error replicate single", err)
	}
}

func replicatePredecessorsBucketElems(n *ThisNode) {
	pred := string(n.Predecessor.Id)

	for filename, storingNode := range n.Bucket {
		id := hashString(filename)
		if id < pred {
			replicateSingleBucketElem(n, filename, storingNode)
		}

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
