package core

type MessageSender interface {
	SendMessage(msg *MessageChunk) error
}
