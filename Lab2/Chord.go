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
	a := flag.String("a", "", "ip of client")
	p := flag.Int("p", 0, "port of client")
	ja := flag.String("ja", "", "ip of existing node")
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
	n.Bucket = make(map[string]Key)
	n.Successor = make([]Node, *r)
	n.FingerTable = make([]Node, 160)

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
	n.Successor[0] = n.Node
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
	n.Successor[0] = succ
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
	for i := (len(n.FingerTable) - 1); i >= 0; i-- {
		finger := n.FingerTable[i]
		//fmt.Println(i, finger.Id, id)
		if isCircleBetween(finger.Id, n.Id, id) {
			return finger, true
		}
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

		x, err := getPredecessor(n.Successor[0].Addr)
		if err != nil {
			fmt.Println("error stabilize", err)
		}
		if x.Addr == "" {
			continue
		}

		// x in (n, successor) or this node is its own successor
		if isCircleBetween(x.Id, n.Id, n.Successor[0].Id) ||
			(x.Id != "" && n.Id == n.Successor[0].Id) {
			n.Successor[0] = x
			notify(n, n.Successor[0])
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
	// TODO: what to do when predecessor has failed???
	for {
		if logFunctionCalls {
			fmt.Println(time.Now().Format("15:04:05:0001"), "Checking predecessor")
		}

		time.Sleep(time.Duration(*tcp) * time.Millisecond)
		if n.Predecessor.Addr == "" {
			continue
		}

		_, err := sendMessage(n.Predecessor.Addr, HandlePing, "")
		if err != nil {
			fmt.Println("Predecessor has failed", err)
		}

		/*if string(msg) != "200 OK" {
			fmt.Println("Predecessor has failed", msg)
		} else {
			fmt.Println("Predecessor alive")
		}*/
	}
}

func replicateSingleFile(n *ThisNode, filename string, storingNode Key) {
	sendMessage(n.Successor[0].Addr, HandleReplicate, filename+"/"+string(storingNode))
}

func replicatePredecessorsFiles(n *ThisNode) {

	pred := string(n.Predecessor.Id)

	for filename, storingNode := range n.Bucket {
		id := hashString(filename)
		if id < pred {
			replicateSingleFile(n, filename, storingNode)
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
