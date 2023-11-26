package main

import (
	"fmt"
	"net"
)

type ServerError struct {
	Message string
}

func (err *ServerError) Error() string {
	return err.Message
}

func TCPJoinHostPort(addr net.TCPAddr) string {
	return fmt.Sprintf("%v:%v", addr.IP, addr.Port)
}

type ClientConn struct {
	ID         int
	Connection net.Conn
	ClientAddr net.IP
	Username   string
}

type Server struct {
	ServerAddr  net.TCPAddr
	Connections []ClientConn
	Msgs        chan Message
	NextID      int
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

		client, err := server.AddClient(connection)
		if err != nil {
			return err
		}

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
	server.Msgs = make(chan Message)

	return nil
}

func (server *Server) SendMsgs() error {
	for {
		msg := <-server.Msgs
		for _, client := range server.Connections {
			buf, err := Serialize(msg)
			if err != nil {
				return err
			}

			_, err = client.Connection.Write(buf)
			if err != nil {
				return err
			}
			fmt.Println("Sending: ", msg)
		}
	}
}

func (server *Server) AddClient(conn net.Conn) (ClientConn, error) {
	addr := conn.RemoteAddr()
	if addr.Network() != "tcp" {
		return ClientConn{}, &ServerError{Message: "Client network must be TCP"}
	}

	client := ClientConn{ID: server.NextID, Connection: conn, ClientAddr: net.ParseIP(addr.String())}

	server.Connections = append(server.Connections, client)

	if server.NextID >= 1000000 {
		return ClientConn{}, &ServerError{Message: "Ran out of IDs"}
	}
	server.NextID = server.NextID + 1

	return client, nil
}

func (server *Server) HandleClient(client ClientConn) error {
	buffer := make([]byte, 1024)

	for {
		_, err := client.Connection.Read(buffer)
		if err != nil {
			// TODO : move to state-system similar to client
			err = server.CloseClient(client)
			return err
		}

		msg, err := Parse(buffer)
		if err != nil {
			// TODO : move to state-system similar to client
			err = server.CloseClient(client)
			return err
		}

		if client.Username != msg.SenderName {
			client.Username = msg.SenderName
		}

		server.Msgs <- msg
	}
}

func (server *Server) CloseClient(client ClientConn) error {
	err := client.Connection.Close()
	if err != nil {
		return err
	}

	fmt.Println(server.Connections)

	for i, cc := range server.Connections {
		if cc.ID == client.ID {
			server.Connections[i] = server.Connections[len(server.Connections)-1]
			server.Connections = server.Connections[:len(server.Connections)-1]
		}
	}

	fmt.Println(server.Connections)
	return nil
}
