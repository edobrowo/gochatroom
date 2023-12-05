package main

import (
	"bufio"
	"fmt"
	"net"
	"os"

	"github.com/edobrowo/gochatroom/pkg/client"
)

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

	// Must specify username and CLIChat interface before starting the client
	client := client.Client{Username: username, IO: &client.CLIChat{Username: username}}

	ip := net.ParseIP(ServerHost)
	addr := net.TCPAddr{IP: ip, Port: ServerPort, Zone: ""}

	// Client code controls request/response loop
	err := client.Connect(addr)
	if err != nil {
		fmt.Println(err)
		return
	}

	return
}
