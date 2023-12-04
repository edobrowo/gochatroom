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
		res.ResType = response.ResponseType_Message
		res.ReceiverName = req.ReceiverName
		res.Content = req.Content
		break
	case request.Command_Ping:
		res.ResType = response.ResponseType_Server
		res.ReceiverName = req.SenderName
		res.Content = "Pong!"
		break
	default:
		res.ResType = response.ResponseType_Server
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
		res.ResType = response.ResponseType_Server
		res.ReceiverName = req.SenderName
		res.Content = "Registered successfully"
		break
	default:
		res.ResType = response.ResponseType_Server
		res.ReceiverName = req.SenderName
		res.Content = "Unknown command"
		break
	}

	return res
}

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
		res.ResType = response.ResponseType_Server
		res.ReceiverName = req.SenderName
		res.Content = "Invalid request"
		break
	}

	return res
}

func (server *Server) SendResponse(res response.Response) { // TODO : return to correct user set
	for _, client := range server.Connections {
		client.ResponseQueue <- res
	}
}

func (server *Server) HandleRequests() {
	for {
		req := <-server.Reqs

		server.Log.Printf("Received message: type %v, from %v, content \"%v\"", req.ReqType, req.SenderName, req.Content)

		res := BuildResponse(req)

		if req.ReqType == request.RequestType_Status && req.StType == request.Status_Register {
			for _, cc := range server.Connections {
				if cc.ClientAddr == req.ClientAddr {
					cc.Username = req.SenderName
					server.Log.Printf("Registed user (username = %v, address = %v)\n", cc.Username, cc.ClientAddr)
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

	go server.Connections[len(server.Connections)-1].Send(server.ClientDone)
	go server.Connections[len(server.Connections)-1].Receive(server.Reqs, server.ClientDone)

	return nil
}

func (server *Server) RemoveClient() {
	for {
		username := <-server.ClientDone // TODO : investigate why username isn't being set
		for i, cc := range server.Connections {
			if cc.Username == username {
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

				disconnectResponse := response.Response{ResType: response.ResponseType_Server, Content: fmt.Sprintf("%v has disconnected", cc.Username)}
				server.SendResponse(disconnectResponse)
			}
		}
	}
}
