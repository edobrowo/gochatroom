package server

import (
	"fmt"
	"log"
	"net"

	"github.com/edobrowo/gochatroom/pkg/request"
	"github.com/edobrowo/gochatroom/pkg/response"
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
	Connection    net.Conn
	ClientAddr    string
	Username      string
	ResponseQueue chan response.Response
}

type Server struct {
	ServerAddr  net.TCPAddr
	Listener    net.Listener
	Connections []ClientConn
	Reqs        chan request.Request
	Status      chan ServerStatus
	Done        chan ServerStatus
	ClientDone  chan string
	Log         *log.Logger
}

// Controls the server state machine
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

// Cleans up and re-initializes server resources; used on boot or if the address is changed
func (server *Server) Reset() error {
	for _, client := range server.Connections {
		close(client.ResponseQueue)
		err := client.Connection.Close()
		if err != nil {
			server.Log.Fatalln(err)
			return err
		}
	}
	server.Connections = make([]ClientConn, 0)

	if server.Reqs != nil {
		close(server.Reqs)
	}
	server.Reqs = make(chan request.Request)

	return nil
}

// Cleans up server resources; used on server close
func (server *Server) Close() error {
	server.Log.Println("Closing server")

	close(server.Status)
	close(server.Done)
	close(server.Reqs)

	for _, client := range server.Connections {
		close(client.ResponseQueue)
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

// Primary listening loop
func (server *Server) Listen(addr net.TCPAddr) {
	// State machine channel, functionally a wrapper for done channel
	if server.Status == nil {
		server.Log.Println("Creating status channel")
		server.Status = make(chan ServerStatus)
		go server.Monitor()
	}

	// Indicates that the server should terminate
	if server.Done == nil {
		server.Log.Println("Creating close channel")
		server.Done = make(chan ServerStatus)
	}

	// Tells the server to close a client connection
	if server.ClientDone == nil {
		server.Log.Println("Creating client close channel")
		server.ClientDone = make(chan string)
		go server.RemoveClient()
	}

	server.Status <- ServerStatus{Code: Idle}

	err := server.Reset()
	if err != nil {
		server.Log.Fatalln("Could not reset server: ", err)
		server.Status <- ServerStatus{Code: ErrorState, Error: err}
		return
	}

	// Begin listening for client connections
	server.ServerAddr = addr
	listener, err := net.Listen("tcp", TCPJoinHostPort(server.ServerAddr))
	if err != nil {
		server.Log.Fatalln("Listener could not be created: ", err)
		server.Status <- ServerStatus{Code: ErrorState, Error: err}
		return
	}
	server.Listener = listener
	server.Status <- ServerStatus{Code: Listening}

	// Used to add new client connections to the server
	go server.AcceptClients()

	// Central goroutine to handle all requests
	go server.HandleRequests()

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

func BuildMessageResponse(req request.Request) response.Response {
	res := response.Response{}
	res.SenderName = req.SenderName
	res.Content = req.Content
	return res
}

func BuildCommandResponse(req request.Request) response.Response {
	res := response.Response{}
	res.SenderName = req.SenderName

	switch req.CmdType {
	case request.Command_Whisper:
		res.ResType = response.ResponseType_Whisper
		res.ReceiverName = req.ReceiverName
		res.Content = req.Content
		break
	case request.Command_Ping:
		res.ResType = response.ResponseType_ServerPriv
		res.ReceiverName = req.SenderName
		res.Content = "Pong!"
		break
	default:
		res.ResType = response.ResponseType_ServerPriv
		res.ReceiverName = req.SenderName
		res.Content = "Unknown command"
		break
	}

	return res
}

func BuildStatusResponse(req request.Request) response.Response {
	res := response.Response{}
	res.SenderName = req.SenderName

	switch req.StType {
	case request.Status_Register:
		res.ResType = response.ResponseType_ServerAll
		res.Content = fmt.Sprintf("%v has connected", req.SenderName)
		break
	default:
		res.ResType = response.ResponseType_ServerPriv
		res.ReceiverName = req.SenderName
		res.Content = "Unknown command"
		break
	}

	return res
}

// Builds a response conditionally based on a request
func BuildResponse(req request.Request) response.Response {
	res := response.Response{}

	switch req.ReqType {
	case request.RequestType_Message:
		res = BuildMessageResponse(req)
		break
	case request.RequestType_Command:
		res = BuildCommandResponse(req)
		break
	case request.RequestType_Status:
		res = BuildStatusResponse(req)
		break
	default:
		res.ResType = response.ResponseType_ServerPriv
		res.ReceiverName = req.SenderName
		res.Content = "Invalid request"
		break
	}

	return res
}

func (server *Server) SendResponse(res response.Response) {
	// Send only to the requesting user
	if res.ResType == response.ResponseType_ServerPriv {
		for _, client := range server.Connections {
			if client.Username == res.ReceiverName {
				client.ResponseQueue <- res
			}
		}
		return
	}

	// Send only to the sending user, and to receiving user if valid
	if res.ResType == response.ResponseType_Whisper {
		var sender, receiver ClientConn
		for i, client := range server.Connections {
			if client.Username == res.SenderName {
				sender = server.Connections[i]
			}
			if client.Username == res.ReceiverName {
				receiver = server.Connections[i]
			}
		}

		if receiver.Username == "" {
			res.ResType = response.ResponseType_ServerPriv
			res.Content = fmt.Sprintf("User %v does not exist", res.ReceiverName)

			// Just send to sender since receiver is invalid
			sender.ResponseQueue <- res
		} else {
			sender.ResponseQueue <- res
			receiver.ResponseQueue <- res
		}

		return
	}

	// Otherwise send to all users (in the case of ResponseType_Message and ResponseType_ServerAll)
	for _, client := range server.Connections {
		client.ResponseQueue <- res
	}
}

func (server *Server) HandleRequests() {
	for {
		req := <-server.Reqs

		server.Log.Printf("request: type %v from %v", req.ReqType, req.SenderName)

		res := BuildResponse(req)

		// If the user is registering, find their connection and set the username field
		if req.ReqType == request.RequestType_Status && req.StType == request.Status_Register {
			for i := range server.Connections {
				if server.Connections[i].ClientAddr == req.ClientAddr {
					server.Connections[i].Username = req.SenderName
					server.Log.Printf("Registered user (username = %v, address = %v)\n", server.Connections[i].Username, server.Connections[i].ClientAddr)
				}
			}
		}

		server.SendResponse(res)
	}
}

func (client *ClientConn) Send(done chan<- string) {
	for {
		res, ok := <-client.ResponseQueue
		if !ok {
			done <- client.Username
			return
		}

		buf, err := response.Serialize(res)
		if err != nil {
			done <- client.Username
			return
		}

		_, err = client.Connection.Write(buf)
		if err != nil {
			done <- client.Username
			return
		}
	}
}

func (client *ClientConn) Receive(reqs chan<- request.Request, done chan<- string) {
	buffer := make([]byte, 1024)

	for {
		_, err := client.Connection.Read(buffer)
		if err != nil {
			done <- client.Username
			return
		}

		req, err := request.Deserialize(buffer)
		if err != nil {
			done <- client.Username
			return
		}

		req.ClientAddr = client.ClientAddr

		reqs <- req
	}
}

func (server *Server) AddClient(conn net.Conn) error {
	addr := conn.RemoteAddr()
	if addr.Network() != "tcp" {
		return &ServerError{Message: "Client network must be TCP"}
	}

	client := ClientConn{Connection: conn, ClientAddr: addr.String(), ResponseQueue: make(chan response.Response)}

	server.Connections = append(server.Connections, client)

	server.Log.Println("Client connected: ", client.ClientAddr)

	// Each client has a Send goroutine for sending responses to
	go server.Connections[len(server.Connections)-1].Send(server.ClientDone)

	// Each client has a Receive goroutine for receiving requests from
	go server.Connections[len(server.Connections)-1].Receive(server.Reqs, server.ClientDone)

	return nil
}

func (server *Server) RemoveClient() {
	for {
		username := <-server.ClientDone
		for i, cc := range server.Connections {
			if cc.Username == username {
				// Use a simple replace-with-last policy
				server.Connections[i] = server.Connections[len(server.Connections)-1]
				server.Connections = server.Connections[:len(server.Connections)-1]

				close(cc.ResponseQueue)
				err := cc.Connection.Close()
				if err != nil {
					server.Log.Fatalln("Client connection could not be closed: ", err)
					server.Status <- ServerStatus{Code: ErrorState, Error: err}
					return
				}

				server.Log.Printf("Client (username = %v, address = %v) disconnected\n", username, cc.ClientAddr)

				// Manually send a response to all users indicating that a user has disconnected
				disconnectResponse := response.Response{ResType: response.ResponseType_ServerAll, Content: fmt.Sprintf("%v has disconnected", cc.Username)}
				server.SendResponse(disconnectResponse)
			}
		}
	}
}
