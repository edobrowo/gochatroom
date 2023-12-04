package request

import (
	"bytes"
	"encoding/binary"
	"strings"
)

type RequestType int

const (
	RequestType_Message RequestType = 0
	RequestType_Command RequestType = 1
	RequestType_Status  RequestType = 2
)

type CommandType int

const (
	Command_Whisper CommandType = 1
	Command_Ping    CommandType = 2
	Command_Unknown CommandType = 3
)

type StatusType int

const (
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

	requestIsCommand := strings.HasPrefix(str, "/")

	commands := map[string]CommandType{
		"whisper": Command_Whisper,
		"w":       Command_Whisper,
		"tell":    Command_Whisper,
		"msg":     Command_Whisper,
		"ping":    Command_Ping,
	}

	if requestIsCommand {
		i := strings.IndexByte(str, ' ')
		if i == -1 {
			i = len(str)
		}

		commandStr := str[1:i]
		command, ok := commands[commandStr]
		if !ok {
			command = Command_Unknown
		}

		req.ReqType = RequestType_Command

		switch command {
		case Command_Whisper:
			req.CmdType = Command_Whisper

			j := strings.IndexByte(str[i+1:], ' ')
			if j == -1 {
				j = len(str)
			}

			req.ReceiverName = str[i : i+j]
			req.Content = str[j:]
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
