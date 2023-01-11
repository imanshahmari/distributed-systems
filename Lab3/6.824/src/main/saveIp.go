package main

import (
	"fmt"
	"os"

	"6.824/mr"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: go run saveIp.go ip:port\n")
		os.Exit(1)
	}

	f, err := os.Create("coordinatorIp.txt")
	if err != nil {
		fmt.Println(err)
	}

	f.Write([]byte(os.Args[1]))

	err = mr.UploadFile("coordinatorIp.txt")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Saved ip:", os.Args[1])
}
