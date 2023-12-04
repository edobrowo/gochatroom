package client

import (
	"bufio"
	"fmt"
	"os"

	"github.com/edobrowo/gochatroom/pkg/response"
)

type CLIChat struct{}

// TODO : make the CLI nicer

func (cli CLIChat) GetInput(sender chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		userInput := scanner.Text()
		sender <- userInput
	}
}

func (cli CLIChat) DisplayOutput(receiver <-chan response.Response) {
	for {
		res := <-receiver
		fmt.Printf("%v: %v\n", res.SenderName, res.Content)
	}
}
