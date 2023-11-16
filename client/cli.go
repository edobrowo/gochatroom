package main

import (
	"bufio"
	"fmt"
	"os"
)

type CLIChat struct{}

func (cli CLIChat) GetInput(sender chan<- Message) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		userInput := scanner.Text()
		msg := Message{Content: userInput}
		sender <- msg
	}
}

func (cli CLIChat) DisplayOutput(receiver <-chan Message) {
	for {
		msg := <-receiver
		fmt.Println(msg.Content)
	}
}
