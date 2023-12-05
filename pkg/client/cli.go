package client

import (
	"bufio"
	"fmt"
	"os"

	"github.com/edobrowo/gochatroom/pkg/response"
)

type CLIChat struct {
	Username string
}

func (cli *CLIChat) GetInput(sender chan<- string) {
	scanner := bufio.NewScanner(os.Stdin)

	for {
		scanner.Scan()
		userInput := scanner.Text()
		sender <- userInput
	}
}

func (cli *CLIChat) DisplayOutput(receiver <-chan response.Response) {
	for {
		res := <-receiver
		var str string

		switch res.ResType {
		case response.ResponseType_Message:
			str = fmt.Sprintf("%v: %v", res.SenderName, res.Content)
			break
		case response.ResponseType_Whisper:
			if res.ReceiverName == cli.Username {
				str = fmt.Sprintf("from %v: %v", res.SenderName, res.Content)
			} else if res.SenderName == cli.Username {
				str = fmt.Sprintf("to %v: %v", res.ReceiverName, res.Content)
			}
			break
		case response.ResponseType_ServerPriv:
			str = fmt.Sprintf("from SERVER: %v", res.Content)
			break
		case response.ResponseType_ServerAll:
			str = fmt.Sprintf("SERVER: %v", res.Content)
			break
		default:
			str = "Unknown response"
		}

		fmt.Println(str)
	}
}
