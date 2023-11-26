package main

import (
	"bytes"
	"encoding/binary"
)

// TODO : make into a shared module/package/something

type MessageType int

const (
	MsgType_Broadcast MessageType = 0
	MsgType_Whisper   MessageType = 1
)

type Message struct {
	MsgType      MessageType
	SenderName   string
	ReceiverName string
	Content      string
}

func Serialize(msg Message) ([]byte, error) {
	buffer := new(bytes.Buffer)

	err := binary.Write(buffer, binary.LittleEndian, uint32(msg.MsgType))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, uint32(len(msg.SenderName)))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, []byte(msg.SenderName))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, uint32(len(msg.ReceiverName)))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, []byte(msg.ReceiverName))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, uint32(len(msg.Content)))
	if err != nil {
		return nil, err
	}

	err = binary.Write(buffer, binary.LittleEndian, []byte(msg.Content))
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func Parse(buffer []byte) (Message, error) {

	reader := bytes.NewReader(buffer)
	msg := Message{}

	var strBuf []byte
	var strLength int32
	var msgType int32

	err := binary.Read(reader, binary.LittleEndian, &msgType)
	if err != nil {
		return Message{}, err
	}
	msg.MsgType = MessageType(msgType)

	err = binary.Read(reader, binary.LittleEndian, &strLength)
	if err != nil {
		return Message{}, err
	}

	strBuf = make([]byte, strLength)
	err = binary.Read(reader, binary.LittleEndian, strBuf)
	if err != nil {
		return Message{}, err
	}
	msg.SenderName = string(strBuf)

	err = binary.Read(reader, binary.LittleEndian, &strLength)
	if err != nil {
		return Message{}, err
	}

	strBuf = make([]byte, strLength)
	err = binary.Read(reader, binary.LittleEndian, strBuf)
	if err != nil {
		return Message{}, err
	}
	msg.ReceiverName = string(strBuf)

	err = binary.Read(reader, binary.LittleEndian, &strLength)
	if err != nil {
		return Message{}, err
	}

	strBuf = make([]byte, strLength)
	err = binary.Read(reader, binary.LittleEndian, strBuf)
	if err != nil {
		return Message{}, err
	}
	msg.Content = string(strBuf)

	return msg, nil
}
