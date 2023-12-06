package response

import (
	"bytes"
	"encoding/binary"
)

type ResponseType int

const (
	// Message to be broadcast to all users
	ResponseType_Message ResponseType = 0

	// Message is private
	ResponseType_Whisper ResponseType = 1

	// Response from the server to a particular user
	ResponseType_ServerPriv ResponseType = 2

	// Response from the server to all users
	ResponseType_ServerAll ResponseType = 3

	// Indicates to the client to close its connection
	ResponseType_TerminateConnection ResponseType = 4
)

type Response struct {
	ResType      ResponseType
	SenderName   string
	ReceiverName string
	Content      string
}

func Serialize(res Response) ([]byte, error) {
	buffer := new(bytes.Buffer)

	err := binary.Write(buffer, binary.LittleEndian, uint32(res.ResType))
	if err != nil {
		return nil, err
	}

	// Strings are serialized as a length followed by character array
	err = binary.Write(buffer, binary.LittleEndian, uint32(len(res.SenderName)))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, []byte(res.SenderName))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, uint32(len(res.ReceiverName)))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, []byte(res.ReceiverName))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, uint32(len(res.Content)))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, []byte(res.Content))
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func Deserialize(buffer []byte) (Response, error) {

	reader := bytes.NewReader(buffer)
	res := Response{}

	var strBuf []byte
	var strLength int32
	var resType uint32

	err := binary.Read(reader, binary.LittleEndian, &resType)
	if err != nil {
		return Response{}, err
	}
	res.ResType = ResponseType(resType)

	// Strings are deserialized by first reading the length then reading the character array
	err = binary.Read(reader, binary.LittleEndian, &strLength)
	if err != nil {
		return Response{}, err
	}

	strBuf = make([]byte, strLength)
	err = binary.Read(reader, binary.LittleEndian, strBuf)
	if err != nil {
		return Response{}, err
	}
	res.SenderName = string(strBuf)

	err = binary.Read(reader, binary.LittleEndian, &strLength)
	if err != nil {
		return Response{}, err
	}

	strBuf = make([]byte, strLength)
	err = binary.Read(reader, binary.LittleEndian, strBuf)
	if err != nil {
		return Response{}, err
	}
	res.ReceiverName = string(strBuf)

	err = binary.Read(reader, binary.LittleEndian, &strLength)
	if err != nil {
		return Response{}, err
	}

	strBuf = make([]byte, strLength)
	err = binary.Read(reader, binary.LittleEndian, strBuf)
	if err != nil {
		return Response{}, err
	}
	res.Content = string(strBuf)

	return res, nil
}
