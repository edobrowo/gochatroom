package main

import (
	"fmt"
	"net"
)

const (
	ServerHost = "127.0.0.1"
	ServerPort = 9988
)

func main() {

	server := Server{}

	ip := net.ParseIP(ServerHost)
	addr := net.TCPAddr{IP: ip, Port: ServerPort, Zone: ""}

	err := server.Listen(addr)
	if err != nil {
		fmt.Println(err)
		return
	}
}
