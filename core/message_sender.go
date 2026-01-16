package core

type MessageSender interface {
	//SendMessage(msg *MessageChunk) error
	SendChunkMessage(messageType, chunkType, messageChunk string, chunkSequence int) error
}
