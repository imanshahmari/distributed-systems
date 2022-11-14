package main

import (
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path"
)

var (
	openThreads = 0
)

func main() {

	arguments := os.Args
	if len(arguments) == 1 {
		fmt.Println("Please provide port number")
		return
	}
	PORT := ":" + arguments[1]
	//fmt.Fprintf(os.Stdout, path.Base("/a/b"))

	//http.HandleFunc("/", handler)
	//http.ListenAndServe(":9000", nil)

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

		// TODO: Use waitgroups wg.Add(1)
		for openThreads >= 10 {
			// TODO: wait
		}
		openThreads++

		go handle(conn)
	}
}

func handle(conn net.Conn) {
	// Anonymous function to decrement openThreads at the end (defer must call function)
	defer func() { openThreads -= 1 }()
	defer conn.Close()

	req := http.Request.readRequest(conn)

	switch req.method {
	case "GET":
		getHandler(conn, req)
		break
	case "POST":
		postHandler(conn, req)
		break
	default:
		sendResponse(501, nil, req)
	}

}

func getHandler(req http.Request, pathStr string) {

	data, err := os.ReadFile(path.Base(pathStr))
	if err != nil {
		// TODO: Responde with 404 error not found
		//conn.Write([]byte("404"))
		sendResponse(404, nil, req)
	}
	//fmt.Fprintf(w, html)
	//conn.Write(data)
	sendResponse(200, data, req)
}

func postHandler(req http.Request, pathStr string, data []byte) {
	// TODO: Is this the correct fs mode???
	err := os.WriteFile(path.Base(pathStr), data, fs.ModeAppend)
	if err != nil {
		// TODO: Responde with correct error
		//conn.Write([]byte("error"))
		sendResponse(500, nil, req)
	}

	//fmt.Fprintf(w, html)

	// TODO: Send success message
	sendResponse(200, nil, req)
	//conn.Write([]byte("200 Success"))
}

func sendResponse(statusCode int, body []byte, req http.Request) {
	status := ""
	switch statusCode {
	case 200:
		status = "200 OK"
		break
	case 404:
		status = "404 Not Found"
		break
	case 500:
		status = "500 Internal Server Error"
		break
	case 501:
		status = "501 Not Implemented"
		break
	}

	t := &http.Response{
		Status:        status,
		StatusCode:    statusCode,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          body,
		ContentLength: int64(len(body)),
		Request:       req,
		Header:        make(http.Header, 0),
	}

	// TODO: send response
}
