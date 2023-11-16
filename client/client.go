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
	GetInput(chan<- Message)
	DisplayOutput(<-chan Message)
}

type Client struct {
	ServerAddr net.TCPAddr
	Connection net.Conn
	User       MessageIO
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

func (client *Client) Connect(addr net.TCPAddr) error {
	if client.User == nil {
		return &ClientError{Message: "Client requires interaction handler"}
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

	sender := make(chan Message)
	receiver := make(chan Message)
	status := make(chan ClientStatus)

	go client.Send(sender, status)
	go client.Receive(receiver, status)

	go client.User.GetInput(sender)
	go client.User.DisplayOutput(receiver)

	done := make(chan ClientStatus)

	go client.Monitor(status, done)

	status <- ClientStatus{Code: Connected}

	// TODO : support reconnection

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
			fmt.Println("Connected to ", TCPJoinHostPort(client.ServerAddr))
			continue
		case Sending:
			continue
		case Receiving:
			continue
		case Disconnected:
			done <- ClientStatus{Code: Disconnected}
			return
		case ErrorState:
			done <- ClientStatus{Code: ErrorState, Error: statusVal.Error}
		case Unknown:
			fallthrough
		default:
			done <- ClientStatus{Code: Unknown}
			return
		}
	}
}

func (client Client) Send(sender <-chan Message, status chan<- ClientStatus) {
	for {
		msg := <-sender

		status <- ClientStatus{Code: Sending}

		n, err := client.Connection.Write([]byte(msg.Content))
		if err != nil {
			errMsg := "Could not send message"
			status <- ClientStatus{Code: ErrorState, Error: &ClientError{Message: errMsg}}
			return
		}
		if n != len(msg.Content) {
			errMsg := fmt.Sprintf("Incorrect number of bytes written. %v bytes written instead of %v\n", n, len(msg.Content))
			status <- ClientStatus{Code: ErrorState, Error: &ClientError{Message: errMsg}}
			return
		}
	}
}

func (client Client) Receive(receiver chan<- Message, status chan<- ClientStatus) {
	buffer := make([]byte, 1024)

	for {
		n, err := client.Connection.Read(buffer)
		if err != nil {
			errMsg := "Could not receive message from server"
			status <- ClientStatus{Code: ErrorState, Error: &ClientError{Message: errMsg}}
			break
		}

		status <- ClientStatus{Code: Receiving}

		msgContent := string(buffer[:n])
		msg := Message{Content: msgContent}
		receiver <- msg
	}
}
