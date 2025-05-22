package common

type MessageSender interface {
	SendMessage(msg *MessageChunk) error
}
