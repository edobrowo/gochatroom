package request

import (
	"bytes"
	"encoding/binary"
	"strings"
)

type RequestType int

const (
	// Message request is a message from one user to be displayed to one or more other users
	RequestType_Message RequestType = 0

	// Command request is a directive to the server to perform some action, or to limit the number of users who receives a message
	RequestType_Command RequestType = 1

	// Status request is a directive to the server to perform a housekeeping task, namely registering a username
	RequestType_Status RequestType = 2
)

type CommandType int

const (
	// Private message another user
	Command_Whisper CommandType = 1

	// Pong!
	Command_Ping CommandType = 2

	Command_Unknown CommandType = 3
)

type StatusType int

const (
	// Associates a connection to a username
	Status_Register StatusType = 0
)

type Request struct {
	ReqType      RequestType
	CmdType      CommandType
	StType       StatusType
	SenderName   string
	ReceiverName string
	Content      string
	ClientAddr   string
}

func Parse(str string) Request {
	req := Request{}

	// Strings are commands if they are prefixed with / without trimming
	requestIsCommand := strings.HasPrefix(str, "/")

	commands := map[string]CommandType{
		"whisper": Command_Whisper,
		"w":       Command_Whisper,
		"tell":    Command_Whisper,
		"msg":     Command_Whisper,
		"ping":    Command_Ping,
	}

	if requestIsCommand {
		req.ReqType = RequestType_Command

		tokens := strings.Split(str[1:], " ")

		command, ok := commands[tokens[0]]
		if !ok {
			command = Command_Unknown
		}

		switch command {
		case Command_Whisper:
			req.CmdType = Command_Whisper

			if len(tokens) < 2 {
				req.ReceiverName = ""
				req.Content = ""
			} else if len(tokens) < 3 {
				req.ReceiverName = tokens[1]
				req.Content = ""
			} else {
				req.ReceiverName = tokens[1]
				req.Content = strings.Join(tokens[2:], " ")
			}

			break
		case Command_Ping:
			req.CmdType = Command_Ping
			break
		default:
			req.CmdType = Command_Unknown
			break
		}
	} else {
		req.ReqType = RequestType_Message
		req.Content = str
	}

	return req
}

func Serialize(req Request) ([]byte, error) {
	buffer := new(bytes.Buffer)

	err := binary.Write(buffer, binary.LittleEndian, uint32(req.ReqType))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, uint32(req.CmdType))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, uint32(req.StType))
	if err != nil {
		return nil, err
	}

	// Strings are serialized as a length followed by character array
	err = binary.Write(buffer, binary.LittleEndian, uint32(len(req.SenderName)))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, []byte(req.SenderName))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, uint32(len(req.ReceiverName)))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, []byte(req.ReceiverName))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, uint32(len(req.Content)))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, []byte(req.Content))
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func Deserialize(buffer []byte) (Request, error) {

	reader := bytes.NewReader(buffer)
	req := Request{}

	var strBuf []byte
	var strLength int32
	var reqType uint32
	var cmdType uint32
	var stType uint32

	err := binary.Read(reader, binary.LittleEndian, &reqType)
	if err != nil {
		return Request{}, err
	}
	req.ReqType = RequestType(reqType)

	err = binary.Read(reader, binary.LittleEndian, &cmdType)
	if err != nil {
		return Request{}, err
	}
	req.CmdType = CommandType(cmdType)

	err = binary.Read(reader, binary.LittleEndian, &stType)
	if err != nil {
		return Request{}, err
	}
	req.StType = StatusType(stType)

	// Strings are deserialized by first reading the length then reading the character array
	err = binary.Read(reader, binary.LittleEndian, &strLength)
	if err != nil {
		return Request{}, err
	}

	strBuf = make([]byte, strLength)
	err = binary.Read(reader, binary.LittleEndian, strBuf)
	if err != nil {
		return Request{}, err
	}
	req.SenderName = string(strBuf)

	err = binary.Read(reader, binary.LittleEndian, &strLength)
	if err != nil {
		return Request{}, err
	}

	strBuf = make([]byte, strLength)
	err = binary.Read(reader, binary.LittleEndian, strBuf)
	if err != nil {
		return Request{}, err
	}
	req.ReceiverName = string(strBuf)

	err = binary.Read(reader, binary.LittleEndian, &strLength)
	if err != nil {
		return Request{}, err
	}

	strBuf = make([]byte, strLength)
	err = binary.Read(reader, binary.LittleEndian, strBuf)
	if err != nil {
		return Request{}, err
	}
	req.Content = string(strBuf)

	return req, nil
}
