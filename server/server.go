package main

import (
	"fmt"
	"net"
)

func TCPJoinHostPort(addr net.TCPAddr) string {
	return fmt.Sprintf("%v:%v", addr.IP, addr.Port)
}

type ClientConn struct {
	Connection net.Conn
}

type Server struct {
	ServerAddr  net.TCPAddr
	Connections []ClientConn
	Msgs        chan string
}

type ChatServer interface {
	Listen(addr net.TCPAddr) error
	Reset() error
	SendMsgs() error
	AddClient() ClientConn
	HandleClient(client ClientConn) error
}

func (server *Server) Listen(addr net.TCPAddr) error {

	err := server.Reset()
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", TCPJoinHostPort(addr))
	if err != nil {
		return err
	}

	defer listener.Close()

	go server.SendMsgs()

	for {
		connection, err := listener.Accept()
		if err != nil {
			return err
		}

		client := server.AddClient(connection)

		go server.HandleClient(client)
	}
}

func (server *Server) Reset() error {
	for _, client := range server.Connections {
		err := client.Connection.Close()
		if err != nil {
			return err
		}
	}

	server.Connections = nil
	server.Connections = make([]ClientConn, 0)

	server.Msgs = nil
	server.Msgs = make(chan string)

	return nil
}

func (server *Server) SendMsgs() error {
	for {
		msg := <-server.Msgs
		for _, client := range server.Connections {
			buffer := []byte(msg)
			_, err := client.Connection.Write(buffer)
			if err != nil {
				return err
			}
			fmt.Println("Sending: ", msg)
		}
	}
}

func (server *Server) AddClient(conn net.Conn) ClientConn {
	client := ClientConn{Connection: conn}

	server.Connections = append(server.Connections, client)

	return client
}

func (server *Server) HandleClient(client ClientConn) error {
	buffer := make([]byte, 1024)

	for {
		n, err := client.Connection.Read(buffer)
		if err != nil {
			return err
		}

		server.Msgs <- string(buffer[:n])
	}
}
