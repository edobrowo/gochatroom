package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func TCPJoinHostPort(addr net.TCPAddr) string {
	return fmt.Sprintf("%v:%v", addr.IP, addr.Port)
}

type Client struct {
	ServerAddr net.TCPAddr
	Connection net.Conn
}

type ChatClient interface {
	Connect(addr net.TCPAddr) error
	Run() error
	SendMsgs()
	ReceiveMsgs()
}

type ClientError struct {
	Message string
}

func (err *ClientError) Error() string {
	return err.Message
}

func (client *Client) Connect(addr net.TCPAddr) error {
	if addr.IP == nil {
		return &net.AddrError{Err: "Address cannot be empty", Addr: TCPJoinHostPort(client.ServerAddr)}
	}
	if addr.Port == 0 {
		return &net.AddrError{Err: "Port cannot be 0", Addr: TCPJoinHostPort(client.ServerAddr)}
	}

	client.ServerAddr = addr

	connection, err := net.Dial("tcp", TCPJoinHostPort(client.ServerAddr))
	if err != nil {
		return err
	}

	client.Connection = connection

	return nil
}

func (client *Client) Run() error {
	if client.Connection == nil {
		return &ClientError{Message: "Client connection must be specified"}
	}

	sendChan := make(chan string)
	recvChan := make(chan string)
	errChan := make(chan ClientError)

	go client.SendMsgs(sendChan, errChan)
	go client.ReceiveMsgs(recvChan, errChan)

	go GetUserInput(sendChan)
	go DisplayOutput(recvChan)

	for {
		clientErr, ok := <-errChan
		if ok {
			break
		} else {
			close(sendChan)
			close(recvChan)
			fmt.Println(clientErr)
			return &clientErr
		}
	}

	return nil
}

func (client Client) SendMsgs(sendChan <-chan string, errChan chan<- ClientError) {
	for {
		msg := <-sendChan
		n, err := client.Connection.Write([]byte(msg))
		if err != nil {
			errChan <- ClientError{Message: "Could not send message"}
			break
		}
		if n != len(msg) {
			errMsg := fmt.Sprintf("Incorrect number of bytes written. %v bytes written instead of %v\n", n, len(msg))
			errChan <- ClientError{Message: errMsg}
			break
		}
	}
}

// use something other than error channel (i.e., a state channel). Can do this once simple commands are added
// then there can be an exit command that would actually necessitate such a channel
func (client Client) ReceiveMsgs(recvChan chan<- string, errChan chan<- ClientError) {
	buffer := make([]byte, 1024)

	for {
		n, err := client.Connection.Read(buffer)
		if err != nil {
			fmt.Println(err)
			errChan <- ClientError{Message: "Could not receive message from server"}
			break
		}

		msg := string(buffer[:n])
		recvChan <- msg
	}
}

func GetUserInput(sendChan chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		userInput := scanner.Text()
		sendChan <- userInput
	}
}

func DisplayOutput(recvChan <-chan string) {
	for {
		msg := <-recvChan
		fmt.Println(msg)
	}
}
