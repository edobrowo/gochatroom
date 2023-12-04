package response

import (
	"bytes"
	"encoding/binary"
)

type ResponseType int

const (
	ResponseType_Message ResponseType = 0
	ResponseType_Server  ResponseType = 1
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
