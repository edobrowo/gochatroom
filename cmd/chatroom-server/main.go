package main

import (
	"log"
	"net"
	"os"

	"github.com/edobrowo/gochatroom/pkg/server"
)

const (
	ServerHost = "127.0.0.1"
	ServerPort = 9988
)

func main() {

	server := server.Server{}

	ip := net.ParseIP(ServerHost)
	addr := net.TCPAddr{IP: ip, Port: ServerPort, Zone: ""}

	server.Log = log.New(os.Stdout, "gochatroom-server:", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

	// Server code controls request/response loop
	server.Listen(addr)
}
