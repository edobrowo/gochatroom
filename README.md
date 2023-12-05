# gochatroom
Chatroom server and client, built in Go. Supports global messages, private messages, usernames, and chat commands. Uses a custom protocol for communication on a CSM network.

## Sample commands
- /ping - Pong!
- /whisper, /w, /tell /msg - private message another user

## Building
```Bash
# Server
go build -o bin/server cmd/chatroom-server/main.go
# Client
go build -o bin/client cmd/chatroom-client/main.go
```
