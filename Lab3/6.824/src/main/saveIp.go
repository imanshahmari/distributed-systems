package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"6.824/mr"
)

func main() {
	url := "https://api.ipify.org?format=text"
	fmt.Printf("Getting IP address from  ipify ...\n")
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	ip, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	mr.SaveIp(string(ip) + ":1234")
}
