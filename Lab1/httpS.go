package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path"
)

func getHandler(w http.ResponseWriter, r *http.Request) {
	var html, err = loadFile(path.Base(r.URL.EscapedPath()))
	if err != nil {
		w.WriteHeader(404)
	}
	fmt.Fprintf(w, html)
}

func loadFile(filename string) (string, error) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(bytes), nil

}

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

	listner, err := net.Listen("tcp", PORT)
	if err != nil {
		log.Fatalln(err)
	}
	defer listner.Close()

	for {
		conn, err := listner.Accept()
		if err != nil {
			log.Println(err)
			// skip the handle function if error
			continue
		}
		//defer c.Close()

		go handle(conn)

	}

}

func handle(conn net.Conn) {
	defer conn.Close()

	request(conn)

	respond(conn)

}

func request(conn net.Conn) {
	i := 0
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if i == 0 {
			//first word of the first
			m := string.Fields(line)[0]

			if m == "GET"{

			}else if m == "POST"{

			}else{

			}
		}
		if 

	}
}

func response(conn net.Conn) {

}
