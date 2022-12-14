package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

type HandleFunction string

// Enum to only allow accepted functions in code
const (
	HandleFindSucc    HandleFunction = "findsuccessor"
	HandlePing        HandleFunction = "ping"
	HandleStoreFile   HandleFunction = "storefile"
	HandleNotify      HandleFunction = "notify"
	HandlePredecessor HandleFunction = "predecessor"
	HandleLookup      HandleFunction = "lookup"
	HandleReplicate   HandleFunction = "replicate"
	HandleGetFile     HandleFunction = "getfile"
	HandlePostFile    HandleFunction = "postfile"
)

// Response type
type Communication struct {
	Node        Node `json:"nodedata"`
	IsRelayAddr bool `json:"isrelayaddr"`
}

var (
	openThreads    = 0
	maxThreads     = 20
	logSendRecieve = false
)

/***** Server *****/

func listen(n *ThisNode, port int) {
	wg := new(sync.WaitGroup)

	//log.Println("Server started on port", port)

	// Initialize a tcp listner with the port specified
	listner, err := net.Listen("tcp", ":"+fmt.Sprint(port))
	if err != nil {
		log.Fatalln("error listen start", err)
	}

	// Closes the linstner at the end of runtime
	defer listner.Close()

	// Listen to incoming connections forever
	for {
		conn, err := listner.Accept()
		if err != nil {
			log.Println("error listen accept", err)
			// skip the handle function if error
			continue
		}

		for openThreads >= maxThreads {
			wg.Wait()
		}

		wg.Add(1)
		go handle(n, &conn, wg)
	}
}

func handle(n *ThisNode, conn *net.Conn, wg *sync.WaitGroup) {
	// Anonymous function to decrement openThreads at the end (defer must call function)
	defer func() { openThreads -= 1 }()
	defer (*conn).Close()
	defer wg.Done()

	req, err := http.ReadRequest(bufio.NewReader(*conn))
	if err != nil {
		// stop handling if request is invalid
		return
	}

	p := strings.Split(req.URL.Path, "/")
	if logSendRecieve {
		log.Println("Recieved: ", p)
	}

	// Convert first argument to function
	switch HandleFunction(p[1]) {
	case HandleFindSucc:
		handleFindSuccessor(n, Key(p[2]), req, conn)

	case HandleNotify:
		handleNotify(n, NodeAddress(p[2]), Key(p[3]), req, conn)

	case HandlePredecessor:
		handlePredecessor(n, req, conn)

	case HandlePing:
		sendResponse(200, nil, req, conn)

	case HandleStoreFile:
		// Replicate on successor then store
		replicateSingleBucketElem(n, p[2], Key(p[3]))
		fallthrough
	case HandleReplicate:
		// When replicating we only store
		handleStoreFile(n, p[2], Key(p[3]), req, conn)

	case HandleGetFile:
		handleGetFile(p[2], req, conn)
	case HandlePostFile:
		handlePostFile(n, p[2], req, conn)

	case HandleLookup:
		handleLookup(n, p[2], req, conn)

	default:
		// Other functions not allowed
		sendResponse(400, nil, req, conn)
	}

}

func handleFindSuccessor(n *ThisNode, id Key, req *http.Request, conn *net.Conn) {
	succ, isRelayAddr := findSuccessorIteration(n, id)
	//fmt.Print(".")
	msg := Communication{
		Node:        succ,
		IsRelayAddr: isRelayAddr,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		log.Println("error handleFindSuccessor", err)
	}

	sendResponse(200, body, req, conn)
}

func handleLookup(n *ThisNode, filename string, req *http.Request, conn *net.Conn) {
	storingNode := n.Bucket[filename]
	sendResponse(200, []byte(storingNode), req, conn)
}

// n' thinks it might be our predecessor.
/*
if (predecessor is nil or n' ??? (predecessor, n))
predecessor = n';
*/
func handleNotify(n *ThisNode, address NodeAddress, id Key, req *http.Request, conn *net.Conn) {
	nPrime := Node{
		Addr: address,
		Id:   id,
	}

	// If predecessor is nil or n' is in range of (predecessor, n)
	if n.Predecessor.Addr == "" || n.Predecessor.Id == "" ||
		isCircleBetween(nPrime.Id, n.Predecessor.Id, n.Id) {
		n.Predecessor = nPrime
	}

	sendResponse(200, nil, req, conn)
}

func handlePredecessor(n *ThisNode, req *http.Request, conn *net.Conn) {
	// return this nodes predecessor
	msg := Communication{
		Node:        n.Predecessor,
		IsRelayAddr: false,
	}

	body, err := json.Marshal(msg)
	if err != nil {
		log.Println("error handlePredecessor", err)
	}

	sendResponse(200, body, req, conn)
}

// Handles the cli command storefile
func handleStoreFile(n *ThisNode, filename string, id Key, req *http.Request, conn *net.Conn) {
	n.Bucket[filename] = id
	sendResponse(200, nil, req, conn)
}

// Retrieve file data from a node
func handleGetFile(filePath string, req *http.Request, conn *net.Conn) {

	checkFiletype(filePath, req, conn)

	data, err := os.ReadFile(filePath)
	if err != nil {
		sendResponse(404, nil, req, conn)
		return
	}
	sendResponse(200, data, req, conn)
}

// Storing the file data
func handlePostFile(n *ThisNode, filePath string, req *http.Request, conn *net.Conn) {
	err := checkFiletype(filePath, req, conn)
	if err != nil {
		sendResponse(400, nil, req, conn)
	}

	localfile, err := os.Create("test_" + filePath)
	if err != nil {
		log.Println(err)
	}

	_, err = io.Copy(localfile, req.Body)
	if err != nil {
		sendResponse(500, nil, req, conn)
		return
	}

	n.FilesOnThisNode[filePath] = false
	sendResponse(200, nil, req, conn)
}

func checkFiletype(filePath string, req *http.Request, conn *net.Conn) error {
	_filePath := strings.Split(filePath, "/")
	s := strings.Split(_filePath[len(_filePath)-1], ".")
	extention := s[len(s)-1]

	allowedExtensions := []string{"html", "txt", "gif", "jpeg", "jpg", "css"}

	for _, x := range allowedExtensions {
		if x == extention {
			return nil
		}
	}
	return fmt.Errorf("filetype not allowed")
}

//*/

func sendResponse(statusCode int, body []byte, req *http.Request, conn *net.Conn) {
	status := ""
	switch statusCode {
	case 200:
		status = "200 OK"
	case 400:
		status = "400 Bad Request"
	case 404:
		status = "404 Not Found"
	case 500:
		status = "500 Internal Server Error"
	case 501:
		status = "501 Not Implemented"
	}
	if body == nil {
		body = []byte(status)
	}

	reader := bytes.NewReader(body)

	res := &http.Response{
		Status:        status,
		StatusCode:    statusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          io.NopCloser(reader),
		ContentLength: int64(len(body)),
		Request:       req,
		Header:        make(http.Header, 0),
	}

	res.Header.Set("Content-Type", http.DetectContentType(body))

	res.Write(*conn)
}

/***** Client *****/

func sendMessage(address NodeAddress, function HandleFunction, msg string) ([]byte, error) {
	if address == "" {
		return nil, fmt.Errorf("empty address")
	}

	url := "http://" + string(address) + "/" + string(function) + "/" + msg
	if logSendRecieve {
		log.Println("Sent: ", strings.Split(url, "/"))
	}

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// Successfully got response
	return body, nil
}

func postReplicate(n *ThisNode, predecessor bool) {
	url := "http://" + string(n.Successor[0].Addr) + "/" + string(HandlePostFile) + "/"

	for filename, responsability := range n.FilesOnThisNode {
		if !responsability && predecessor {
			// We take over responsability and replicates to next node
			err := postReplicateOne(url, filename)
			if err != nil {
				fmt.Println(err)
				continue
			}

			// Take over responsability and send the updated node id to chord ring bucket
			storeFile(n, filename)

		} else if responsability && !predecessor {
			// Replicate to have a backup of the data in the system
			err := postReplicateOne(url, filename)
			if err != nil {
				fmt.Println(err)
				continue
			}
		}
	}
}

func postReplicateOne(url string, filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	contentType := http.DetectContentType(data)
	reader := bytes.NewReader(data)

	_, err = http.Post(url+filename, contentType, reader)
	if err != nil {
		return err
	}

	return nil
}

// Parse the respons from findSuccessor
func getFindSuccessor(address NodeAddress, msg string) (Communication, error) {
	body, err := sendMessage(address, HandleFindSucc, msg)
	if err != nil {
		return Communication{}, err
	}

	var data Communication
	err = json.Unmarshal(body, &data)
	if err != nil {
		//println(string(body))
		return Communication{}, err
	}

	//fmt.Println(data)

	// Successfully got response
	return data, nil
}

func getPredecessor(address NodeAddress) (Node, error) {
	body, err := sendMessage(address, HandlePredecessor, "")
	if err != nil {
		return Node{}, err
	}

	var data Communication
	err = json.Unmarshal(body, &data)
	if err != nil {
		//println(string(body))
		return Node{}, err
	}

	//fmt.Println(data)

	// Successfully got response
	return data.Node, nil

}
