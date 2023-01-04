package main

import (
	"fmt"
	"os"

	"6.824/mr"
)

func main() {
	ip := "54.175.32.163:1234"
	f, err := os.Create("coordinatorIp.txt")
	if err != nil {
		fmt.Println(err)
	}

	f.Write([]byte(ip))

	err = mr.UploadFile("coordinatorIp.txt")
	if err != nil {
		fmt.Println(err)
	}
}
