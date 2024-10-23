package chat

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"infini.sh/coco/lib/langchaingo/documentloaders"
	"infini.sh/coco/lib/langchaingo/schema"
	"infini.sh/coco/lib/langchaingo/textsplitter"
)

// TextToChunks 函数将文本文件转换为文档块
func TextToChunks(dirFile string, chunkSize, chunkOverlap int) ([]schema.Document, error) {
	file, err := os.Open(dirFile)
	if err != nil {
		return nil, err
	}
	// 创建一个新的文本文档加载器
	docLoaded := documentloaders.NewText(file)
	// 创建一个新的递归字符文本分割器
	split := textsplitter.NewRecursiveCharacter()
	// 设置块大小
	split.ChunkSize = chunkSize
	// 设置块重叠大小
	split.ChunkOverlap = chunkOverlap
	// 加载并分割文档
	docs, err := docLoaded.LoadAndSplit(context.Background(), split)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

// GetUserInput 获取用户输入
func GetUserInput(promptString string) (string, error) {
	fmt.Print(promptString, ": ")
	var Input string
	reader := bufio.NewReader(os.Stdin)

	Input, _ = reader.ReadString('\n')

	Input = strings.TrimSuffix(Input, "\n")
	Input = strings.TrimSuffix(Input, "\r")

	return Input, nil
}
