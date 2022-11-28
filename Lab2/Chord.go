package main

import (
	"bufio"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"os"
)

type Key big.Int

type NodeAddress string

type Node struct {
	Id 			   Key
	Address        NodeAddress
	FingerTable    map[Key]NodeAddress
	Predecessor_id Key
	Predecessor    NodeAddress
	//Successor      []NodeAddress


	Bucket map[Key]string
}

// struct to unmarshal json into
type Communication struct {
	function string  `json:"function"`
	var1     string  `json:"var1"`
	var2     string  `json:"var2"`
	id       big.Int `json:"id"`
}

var (
	n Node
)

func main() {

	// Get the command line arguments
	a := flag.String("a", "", "ip of client")
	p := flag.Int("p", 0, "port of client")
	ja := flag.String("ja", "", "ip of existing node")
	jp := flag.Int("jp", 0, "port of existing node")
	ts := flag.Int("ts", 0, "time in ms between stabilize")
	tff := flag.Int("tff", 0, "time in ms between fix fingers")
	tcp := flag.Int("tcp", 0, "time in ms between check predecessor")
	r := flag.Int("r", 0, "number of successors")
	i := flag.String("i", "", "id of client (optional)")

	// Supress warning
	_ = i

	flag.Parse()

	if *a == "" || *p == 0 || (*ja != "" && *jp == 0) || *ts == 0 || *tff == 0 || *tcp == 0 || *r == 0 {
		fmt.Print("Flag missing, usage:\n",
			"-a    <String> ip of client\n",
			"-p    <Number> port of client\n",
			"--ja  <String> ip of existing node\n",
			"--jp  <Number> port of existing node\n",
			"--ts  <Number> time in ms between stabilize\n",
			"--tff <Number> time in ms between fix fingers\n",
			"--tcp <Number> time in ms between check predecessor\n",
			"-r    <Number> number of successors\n",
			"-i    <String> id of client (optional)\n")
		return
	}

	n = Node{
		Address:     *a,
		FingerTable: make([]NodeAddress),
		Predecessor: nil,
		Successor:   make([]NodeAddress),
	}

	if *i != "" {
		// If the id is not defined by comand line argument, generate it from hashing ip and port
		n.Id = hash(*a + ":" + str(*p))
	} else {
		// TODO: Convert string hex to BigInt
		n.Id = 0
	}

	// Create a new Chord ring if there is no --ja defined
	if *ja == "" {
		// Create a new ring
		n.Successor_Id = n.id
	} else {
		// Join network
		n.Successor_Id = find_successor(n)
	}

	fmt.Println("Chord server started on adress: ", *a, ":", *p)

	go listen(p)

	// Start a go routine for each of the steps to make the network consistent
	go stabilize(ts)
	go fix_fingers(tff)
	go check_predecessor(tcp)

	for {
		// Read from cmd
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Enter command: ")
		input, _ := reader.ReadString('\n')
		fmt.Print(input)

		switch input {
		case "Lookup":

		case "StoreFile":

		case "PrintState":

		}
	}

}

func find_successor(n Node, i big.Int) {
	if (n.id > i && id <= n.Successor.id){

	} else{
	
	
	}

}

func listen(p *int) {
	http.HandleFunc("/", func(w http.ResponseWriter, request *http.Request) {
		var dat Communication

		if request.Body == nil {
			http.Error(w, "Please send a request body", 400)
			return
		}

		err := json.Unmarshal(bufio.NewReader(request.Body), &dat)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}

		switch dat.function {
		case "find_successor":
			find_successor(, dat.id)
		}

	})

	err := http.ListenAndServe(":"+string(*p), nil)
	if err != nil {
		fmt.Println(err)
	}
}

func handler() {

}

func stabilize(tc *int) {
	return
}

func fix_fingers(tff *int) {
	return
}

func check_predecessor(tcp *int) {
	return
}

func lookup() {
	return
}

func store_file() {
	return
}

func print_state() {
	return
}

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

// ask node n to find the successor of id
//func find_successor(id) {
//	if finger_table[id] {
//		return successor
//	} else {
//		// forward the query around the circle
//		return successor.find_successor(id)
//	}
//}
