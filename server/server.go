package main

import (
	"fmt"
	"log"
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

type ServerStatusCode int

const (
	Idle ServerStatusCode = iota
	Listening
	Closing
	ErrorState
	Unknown
)

type ServerStatus struct {
	Code  ServerStatusCode
	Error error
}

type ClientConn struct {
	ID         int
	Connection net.Conn
	ClientAddr string
	Username   string
	SendQueue  chan Message
}

type Server struct {
	ServerAddr  net.TCPAddr
	Listener    net.Listener
	Connections []ClientConn
	NextID      int
	Msgs        chan Message
	Status      chan ServerStatus
	Done        chan ServerStatus
	ClientDone  chan int
	Log         *log.Logger
}

func (server *Server) Monitor() {
	for {
		switch statusVal := <-server.Status; statusVal.Code {
		case Idle:
			server.Log.Println("Server created, idling")
		case Listening:
			server.Log.Println("Listening on ", TCPJoinHostPort(server.ServerAddr))
		case Closing:
			server.Done <- ServerStatus{Code: Closing}
			return
		case ErrorState:
			server.Done <- ServerStatus{Code: ErrorState, Error: statusVal.Error}
			return
		case Unknown:
			fallthrough
		default:
			server.Log.Fatalln("Unknown status")
			server.Done <- ServerStatus{Code: Unknown}
			return
		}
	}
}

func (server *Server) Reset() error {
	for _, client := range server.Connections {
		close(client.SendQueue)
		err := client.Connection.Close()
		if err != nil {
			server.Log.Fatalln(err)
			return err
		}
	}
	server.Connections = make([]ClientConn, 0)

	if server.Msgs != nil {
		close(server.Msgs)
	}
	server.Msgs = make(chan Message)

	server.NextID = 0

	return nil
}

func (server *Server) Close() error {
	server.Log.Println("Closing server")

	close(server.Status)
	close(server.Done)
	close(server.Msgs)

	for _, client := range server.Connections {
		close(client.SendQueue)
		err := client.Connection.Close()
		if err != nil {
			return err
		}
	}

	err := server.Listener.Close()
	if err != nil {
		return err
	}

	return nil
}

func (server *Server) Listen(addr net.TCPAddr) {
	if server.Status == nil {
		server.Log.Println("Creating status channel")
		server.Status = make(chan ServerStatus)
		go server.Monitor()
	}

	if server.Done == nil {
		server.Log.Println("Creating close channel")
		server.Done = make(chan ServerStatus)
	}

	if server.ClientDone == nil {
		server.Log.Println("Creating client close channel")
		server.ClientDone = make(chan int)
		go server.RemoveClient()
	}

	server.Status <- ServerStatus{Code: Idle}

	err := server.Reset()
	if err != nil {
		server.Log.Fatalln("Could not reset server: ", err)
		server.Status <- ServerStatus{Code: ErrorState, Error: err}
		return
	}

	server.ServerAddr = addr
	listener, err := net.Listen("tcp", TCPJoinHostPort(server.ServerAddr))
	if err != nil {
		server.Log.Fatalln("Listener could not be created: ", err)
		server.Status <- ServerStatus{Code: ErrorState, Error: err}
		return
	}
	server.Listener = listener
	server.Status <- ServerStatus{Code: Listening}

	go server.AcceptClients()
	go server.HandleMessages()

	result := <-server.Done
	if result.Code == ErrorState {
		server.Log.Fatalln("Closing server: ", result.Error)
	}

	err = server.Close()
	if err != nil {
		server.Log.Fatalln("Server close failure: ", err)
	}
}

func (server *Server) AcceptClients() {
	for {
		connection, err := server.Listener.Accept()
		if err != nil {
			server.Log.Fatalln("Listener accept failure: ", err)
			server.Status <- ServerStatus{Code: ErrorState, Error: err}
			return
		}

		err = server.AddClient(connection)
		if err != nil {
			server.Log.Fatalln("Add client failure: ", err)
			server.Status <- ServerStatus{Code: ErrorState, Error: err}
			return
		}
	}
}

func (server *Server) HandleMessages() {
	for {
		msg := <-server.Msgs
		var receiver string
		if msg.MsgType == 0 {
			receiver = "ALL"
		}
		server.Log.Printf("Received message: type %v, from %v, to %v, content \"%v\"", msg.MsgType, msg.SenderName, receiver, msg.Content)
		for _, client := range server.Connections {
			client.SendQueue <- msg
		}
	}
}

func (client *ClientConn) Send(done chan<- int) {
	for {
		msg, ok := <-client.SendQueue
		if !ok {
			done <- client.ID
			return
		}

		buf, err := Serialize(msg)
		if err != nil {
			done <- client.ID
			return
		}

		_, err = client.Connection.Write(buf)
		if err != nil {
			done <- client.ID
			return
		}
	}
}

func (client *ClientConn) Receive(msgs chan<- Message, done chan<- int) {
	buffer := make([]byte, 1024)

	for {
		_, err := client.Connection.Read(buffer)
		if err != nil {
			done <- client.ID
			return
		}

		msg, err := Parse(buffer)
		if err != nil {
			done <- client.ID
			return
		}

		if client.Username != msg.SenderName {
			client.Username = msg.SenderName
		}

		msgs <- msg
	}
}

func (server *Server) AddClient(conn net.Conn) error {
	addr := conn.RemoteAddr()
	if addr.Network() != "tcp" {
		return &ServerError{Message: "Client network must be TCP"}
	}

	client := ClientConn{ID: server.NextID, Connection: conn, ClientAddr: addr.String(), SendQueue: make(chan Message)}

	server.Connections = append(server.Connections, client)

	if server.NextID >= 1000000 {
		return &ServerError{Message: "Ran out of IDs"}
	}
	server.NextID = server.NextID + 1

	server.Log.Println("Client connected: ", client.ClientAddr)

	go server.Connections[len(server.Connections)-1].Send(server.ClientDone)
	go server.Connections[len(server.Connections)-1].Receive(server.Msgs, server.ClientDone)

	// TODO : add server messages (e.g., "User has connected!")

	return nil
}

func (server *Server) RemoveClient() {
	for {
		id := <-server.ClientDone
		for i, cc := range server.Connections {
			if cc.ID == id {
				server.Connections[i] = server.Connections[len(server.Connections)-1]
				server.Connections = server.Connections[:len(server.Connections)-1]
				close(cc.SendQueue)
				err := cc.Connection.Close()
				if err != nil {
					server.Log.Fatalln("Client connection could not be closed: ", err)
					server.Status <- ServerStatus{Code: ErrorState, Error: err}
					return
				}
				server.Log.Printf("Client (ID = %v, username = %v, address = %v) disconnected\n", cc.ID, cc.Username, cc.ClientAddr)
			}
		}
	}
}
