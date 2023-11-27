package main

import (
	"fmt"
	"net"
)

type ClientError struct {
	Message string
}

func (err *ClientError) Error() string {
	return err.Message
}

func TCPJoinHostPort(addr net.TCPAddr) string {
	return fmt.Sprintf("%v:%v", addr.IP, addr.Port)
}

type MessageIO interface {
	GetInput(chan<- string)
	DisplayOutput(<-chan Message)
}

type ClientStatusCode int

const (
	Connected ClientStatusCode = iota
	Sending
	Receiving
	Disconnected
	ErrorState
	Unknown
)

type ClientStatus struct {
	Code  ClientStatusCode
	Error error
}

type Client struct {
	ServerAddr net.TCPAddr
	Connection net.Conn
	Username   string
	IO         MessageIO
}

func (client *Client) Connect(addr net.TCPAddr) error {
	if client.IO == nil {
		return &ClientError{Message: "Client requires interaction handler"}
	}

	if client.Username == "" {
		return &ClientError{Message: "User requires name"}
	}

	if addr.IP == nil {
		return &net.AddrError{Err: "Address cannot be empty", Addr: TCPJoinHostPort(client.ServerAddr)}
	}

	connection, err := net.Dial("tcp", TCPJoinHostPort(addr))
	if err != nil {
		return err
	}

	client.ServerAddr = addr
	client.Connection = connection

	sender := make(chan string)
	receiver := make(chan Message)
	status := make(chan ClientStatus)

	go client.Send(sender, status)
	go client.Receive(receiver, status)

	go client.IO.GetInput(sender)
	go client.IO.DisplayOutput(receiver)

	done := make(chan ClientStatus)

	go client.Monitor(status, done)

	status <- ClientStatus{Code: Connected}

	result := <-done

	close(sender)
	close(receiver)
	close(status)
	close(done)
	client.Connection.Close()

	return result.Error
}

func (client Client) Monitor(status <-chan ClientStatus, done chan<- ClientStatus) {
	for {
		switch statusVal := <-status; statusVal.Code {
		case Connected:
			fmt.Printf("Connected to %v as %v\n", TCPJoinHostPort(client.ServerAddr), client.Username)
		case Sending:
			continue
		case Receiving:
			continue
		case Disconnected:
			done <- ClientStatus{Code: Disconnected}
			return
		case ErrorState:
			done <- ClientStatus{Code: ErrorState, Error: statusVal.Error}
			return
		case Unknown:
			fallthrough
		default:
			done <- ClientStatus{Code: Unknown}
			return
		}
	}
}

func (client Client) Send(sender <-chan string, status chan<- ClientStatus) {
	for {
		input := <-sender

		msg := Message{SenderName: client.Username, Content: input}
		buf, err := Serialize(msg)
		if err != nil {
			errMsg := "Could not seralize message"
			status <- ClientStatus{Code: ErrorState, Error: &ClientError{Message: errMsg}}
			return
		}

		status <- ClientStatus{Code: Sending}

		_, err = client.Connection.Write(buf)
		if err != nil {
			errMsg := "Could not send message"
			status <- ClientStatus{Code: ErrorState, Error: &ClientError{Message: errMsg}}
			return
		}

		/*if n != len(buf) {
			errMsg := fmt.Sprintf("Incorrect number of bytes written. %v bytes written instead of %v\n", n, len(msg.Content))
			status <- ClientStatus{Code: ErrorState, Error: &ClientError{Message: errMsg}}
			return
		}*/
	}
}

func (client Client) Receive(receiver chan<- Message, status chan<- ClientStatus) {
	buffer := make([]byte, 1024)

	for {
		_, err := client.Connection.Read(buffer)
		if err != nil {
			errMsg := "Could not receive message from server"
			status <- ClientStatus{Code: ErrorState, Error: &ClientError{Message: errMsg}}
			return
		}

		status <- ClientStatus{Code: Receiving}

		msg, err := Parse(buffer)
		if err != nil {
			errMsg := "Could not parse message from server"
			status <- ClientStatus{Code: ErrorState, Error: &ClientError{Message: errMsg}}
			return
		}

		receiver <- msg
	}
}
