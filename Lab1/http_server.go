package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
)

var (
	openThreads = 0
)

func main() {
	wg := new(sync.WaitGroup)

	arguments := os.Args
	if len(arguments) == 1 {
		fmt.Println("Please provide port number")
		return
	}
	PORT := ":" + arguments[1]

	fmt.Println("Server started on port", PORT)

	// Initialize a tcp listner with the port specified
	listner, err := net.Listen("tcp", PORT)
	if err != nil {
		log.Fatalln(err)
	}

	// Closes the linstner at the end of runtime
	defer listner.Close()

	// Listen to incoming connections forever
	for {
		conn, err := listner.Accept()
		if err != nil {
			log.Println(err)
			// skip the handle function if error
			continue
		}

		for openThreads >= 10 {
			wg.Wait()
		}

		wg.Add(1)
		go handle(conn, wg)
	}
}

func handle(conn net.Conn, wg *sync.WaitGroup) {
	// Anonymous function to decrement openThreads at the end (defer must call function)
	defer func() { openThreads -= 1 }()
	defer conn.Close()
	defer wg.Done()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		log.Println(err)
	}

	switch req.Method {
	case "GET":
		getHandler(*req, conn)
	case "POST":
		postHandler(*req, conn)
	default:
		sendResponse(501, nil, *req, conn)
	}

}

func getHandler(req http.Request, conn net.Conn) {
	// Remove first slash from path
	path := req.URL.Path[1:]

	// If the path ends in slash it is implicit that we want the index.html file in this folder or empty we want the base index.html
	if len(path) == 0 || path[len(path)-1:] == "/" {
		path += "index.html"
	}
	fmt.Println(path)

	data, err := os.ReadFile(path)
	if err != nil {
		sendResponse(404, nil, req, conn)
		return
	}
	sendResponse(200, data, req, conn)
}

func postHandler(req http.Request, conn net.Conn) {

	req.ParseMultipartForm(32 << 20)
	file, handler, err := req.FormFile("uploadfile")
	if err != nil {
		sendResponse(500, nil, req, conn)
		return
	}

	localfile, err := os.Create(handler.Filename)
	if err != nil {
		log.Println(err)
	}

	_, err = io.Copy(localfile, file)
	if err != nil {
		sendResponse(500, nil, req, conn)
		return
	}

	sendResponse(200, nil, req, conn)
}

func sendResponse(statusCode int, body []byte, req http.Request, conn net.Conn) {
	status := ""
	switch statusCode {
	case 200:
		status = "200 OK"
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
		Request:       &req,
		Header:        make(http.Header, 0),
	}

	res.Write(conn)
}
