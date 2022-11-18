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

	// proxy 80 server:81
	arguments := os.Args
	if len(arguments) == 2 {
		fmt.Println("Please provide port number and proxy url (with port)")
		return
	}

	PORT := ":" + arguments[1]
	proxyURL := arguments[2]

	fmt.Println("Server started on port", PORT, "with forwarding to", proxyURL)

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
		go handle(proxyURL, conn, wg)
	}
}

func handle(proxyURL string, conn net.Conn, wg *sync.WaitGroup) {
	// Anonymous function to decrement openThreads at the end (defer must call function)
	defer func() { openThreads -= 1 }()
	defer conn.Close()
	defer wg.Done()

	req, err := http.ReadRequest(bufio.NewReader(conn))
	if err != nil {
		log.Println(err)
	}

	if req.Method == "GET" {
		getHandler(proxyURL, *req, conn)
	} else {
		sendResponse(501, nil, *req, conn)
		return
	}

}

func getHandler(proxyURL string, req http.Request, conn net.Conn) {
	fmt.Println(proxyURL + req.URL.Path)

	res, err := http.Get("http://" + proxyURL + req.URL.Path)

	if err != nil {
		sendResponse(404, []byte(err.Error()), req, conn)
		return
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		sendResponse(500, nil, req, conn)
		return
	}

	sendResponse(200, body, req, conn)

	//res.Write(conn)
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
