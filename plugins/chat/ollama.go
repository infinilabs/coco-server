package chat

import (
	"context"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/coco/lib/langchaingo/agents"
	"infini.sh/coco/lib/langchaingo/chains"
	"infini.sh/coco/lib/langchaingo/embeddings"
	"infini.sh/coco/lib/langchaingo/llms"
	"infini.sh/coco/lib/langchaingo/llms/ollama"
	"infini.sh/coco/lib/langchaingo/memory"
	"net/url"

	"infini.sh/coco/lib/langchaingo/schema"
	"infini.sh/coco/lib/langchaingo/vectorstores"
	"infini.sh/coco/lib/langchaingo/vectorstores/qdrant"
)

var (
	collectionName = "langchaingo-ollama-rag"
	qdrantUrl      = "http://localhost:6333"
	ollamaServer   = "http://localhost:11434"
)

// GetOllamaEmbedder 获取 ollama 嵌入器
func getollamaEmbedder() *embeddings.EmbedderImpl {
	// 创建一个新的ollama模型，模型名为"nomic-embed-text:latest"
	ollamaEmbedderModel, err := ollama.New(
		ollama.WithModel("nomic-embed-text:latest"),
		ollama.WithServerURL(ollamaServer))
	if err != nil {
		log.Error("创建ollama模型失败: %v", err)
	}
	// 使用创建的ollama模型创建一个新的嵌入器
	ollamaEmbedder, err := embeddings.NewEmbedder(ollamaEmbedderModel)
	if err != nil {
		log.Error("创建ollama嵌入器失败: %v", err)
	}
	return ollamaEmbedder
}

// getOllamaMistral 获取 ollama 模型
func getOllamaMistral() *ollama.LLM {
	// 创建一个新的ollama模型，模型名为"mistral"
	llm, err := ollama.New(
		ollama.WithModel("mistral"),
		ollama.WithServerURL(ollamaServer))
	if err != nil {
		log.Error("创建ollama模型失败: %v", err)
	}
	return llm
}

// getOllamaLLM2 获取 ollama 模型
func getOllamaLlama2() *ollama.LLM {
	// 创建一个新的ollama模型，模型名为"llama2-chinese:13b"
	llm, err := ollama.New(
		ollama.WithModel("llama2-chinese:13b"),
		ollama.WithServerURL(ollamaServer))
	if err != nil {
		log.Error("创建ollama模型失败: %v", err)
	}
	return llm
}

// getStore 获取存储对象
func getStore() *qdrant.Store {
	// 解析URL
	qdUrl, err := url.Parse(qdrantUrl)
	if err != nil {
		log.Error("解析URL失败: %v", err)
	}
	// 创建新的qdrant存储
	store, err := qdrant.New(
		qdrant.WithURL(*qdUrl),                    // 设置URL
		qdrant.WithAPIKey(""),                     // 设置API密钥
		qdrant.WithCollectionName(collectionName), // 设置集合名称
		qdrant.WithEmbedder(getollamaEmbedder()),  // 设置嵌入器
	)
	if err != nil {
		log.Error("创建qdrant存储失败: %v", err)
	}
	return &store
}

// storeDocs 将文档存储到向量数据库
func storeDocs(docs []schema.Document, store *qdrant.Store) error {
	// 如果文档数组长度大于0
	if len(docs) > 0 {
		// 添加文档到存储
		_, err := store.AddDocuments(context.Background(), docs)
		if err != nil {
			return err
		}
	}
	return nil
}

// useRetriaver 函数使用检索器
func useRetriaver(store *qdrant.Store, prompt string, topk int) ([]schema.Document, error) {
	// 设置选项向量
	optionsVector := []vectorstores.Option{
		vectorstores.WithScoreThreshold(0.80), // 设置分数阈值
	}

	// 创建检索器
	retriever := vectorstores.ToRetriever(store, topk, optionsVector...)

	// 搜索
	docRetrieved, err := retriever.GetRelevantDocuments(context.Background(), prompt)

	if err != nil {
		return nil, fmt.Errorf("检索文档失败: %v", err)
	}

	// 返回检索到的文档
	return docRetrieved, nil
}

// GetAnswer 获取答案
func GetAnswer(ctx context.Context, llm llms.Model, docRetrieved []schema.Document, prompt string) (string, error) {
	// 创建一个新的聊天消息历史记录
	history := memory.NewChatMessageHistory()
	// 将检索到的文档添加到历史记录中
	for _, doc := range docRetrieved {
		history.AddAIMessage(ctx, doc.PageContent)
	}
	// 使用历史记录创建一个新的对话缓冲区
	conversation := memory.NewConversationBuffer(memory.WithChatHistory(history))

	executor := agents.NewExecutor(
		agents.NewConversationalAgent(llm, nil),
		nil,
		agents.WithMemory(conversation),
	)
	// 设置链调用选项
	options := []chains.ChainCallOption{
		chains.WithTemperature(0.8),
	}
	// 运行链
	res, err := chains.Run(ctx, executor, prompt, options...)
	if err != nil {
		return "", err
	}

	return res, nil
}

// Translate 将文本翻译为中文
func Translate(llm llms.Model, text string) (string, error) {
	completion, err := llms.GenerateFromSinglePrompt(
		context.TODO(),
		llm,
		"将如下这句话翻译为中文，只需要回复翻译后的内容，而不需要回复其他任何内容。需要翻译的英文内容是: \n"+text,
		llms.WithTemperature(0.8))
	if err != nil {
		return "", err
	}
	return completion, nil
}
