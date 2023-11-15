package main

import (
	"fmt"
	"net"
)

func main() {
	server, err := net.Listen("tcp", "127.0.0.1:9988")
	if err != nil {
		fmt.Println(err)
		return
	}

	defer server.Close()

	for {
		connection, err := server.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		go HandleClient(connection)
	}
}

// Need to gracefully handle EOF when a connection closes
func HandleClient(connection net.Conn) {
	buffer := make([]byte, 1024)

	for {
		numBytes, err := connection.Read(buffer)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("Received: ", string(buffer[:numBytes]))
		_, err = connection.Write([]byte("Echo:" + string(buffer[:numBytes])))
	}

	// connection.Close()
}
