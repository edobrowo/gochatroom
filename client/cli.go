package main

import (
	"bufio"
	"fmt"
	"os"
)

type CLIChat struct{}

func (cli CLIChat) GetInput(sender chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		userInput := scanner.Text()
		sender <- userInput
	}
}

func (cli CLIChat) DisplayOutput(receiver <-chan Message) {
	for {
		msg := <-receiver
		fmt.Println(msg.Content)
	}
}
