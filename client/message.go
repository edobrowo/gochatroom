package main

type MessageType int

const (
	MsgType_Broadcast MessageType = 0
	MsgType_Whisper   MessageType = 1
)

type Message struct {
	MsgType    MessageType
	SenderName string
	Content    string
}
