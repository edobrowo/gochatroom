package main

import (
	"bytes"
	"encoding/binary"
)

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

	// TODO : store length or pad
	_, err = buffer.WriteString(msg.SenderName[:8])
	if err != nil {
		return nil, err
	}

	_, err = buffer.WriteString(msg.ReceiverName[:8])
	if err != nil {
		return nil, err
	}

	_, err = buffer.WriteString(msg.Content)
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func Parse(buffer []byte) Message {
	return Message{}
}
