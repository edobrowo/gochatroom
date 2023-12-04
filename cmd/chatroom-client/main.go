package main

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"github.com/edobrowo/gochatroom/pkg/client"
)

// TODO : comments :)
// TODO : testing
// TODO : readme

const (
	ServerHost = "127.0.0.1"
	ServerPort = 9988
)

func ValidateUsername(s string) (bool, string) {
	if s == "" {
		return false, "username cannot be empty"
	}
	if len(s) > 8 {
		return false, "username must be 8 characters or less :)"
	}
	return true, ""
}

func main() {

	scanner := bufio.NewScanner(os.Stdin)
	var username string

	fmt.Println("Enter username: ")

	for {
		scanner.Scan()
		username = scanner.Text()

		if valid, desc := ValidateUsername(username); valid {
			break
		} else {
			fmt.Println("Username invalid: ", desc)
		}
	}

	client := client.Client{Username: username, IO: client.CLIChat{}}

	ip := net.ParseIP(ServerHost)
	addr := net.TCPAddr{IP: ip, Port: ServerPort, Zone: ""}

	err := client.Connect(addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	return
}
