package assistant

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
)

// The TextToChunks function converts a text file into document chunks
func TextToChunks(dirFile string, chunkSize, chunkOverlap int) ([]schema.Document, error) {
	file, err := os.Open(dirFile)
	if err != nil {
		return nil, err
	}
	// Create a new text document loader
	docLoaded := documentloaders.NewText(file)
	// Create a new recursive character text splitter
	split := textsplitter.NewRecursiveCharacter()
	// Set the chunk size
	split.ChunkSize = chunkSize
	// Set the chunk overlap size
	split.ChunkOverlap = chunkOverlap
	// Load and split the document
	docs, err := docLoaded.LoadAndSplit(context.Background(), split)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// GetUserInput retrieves user input
func GetUserInput(promptString string) (string, error) {
	fmt.Print(promptString, ": ")
	var Input string
	reader := bufio.NewReader(os.Stdin)

	Input, _ = reader.ReadString('\n')

	Input = strings.TrimSuffix(Input, "\n")
	Input = strings.TrimSuffix(Input, "\r")

	return Input, nil
}
