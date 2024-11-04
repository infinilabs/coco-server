package assistant

import (
	log "github.com/cihub/seelog"
	"infini.sh/coco/lib/langchaingo/embeddings"
	"infini.sh/coco/lib/langchaingo/llms/ollama"
)

var (
	collectionName = "langchaingo-ollama-rag"
	qdrantUrl      = "http://localhost:6333"
	ollamaServer   = "http://localhost:11434"
)

func getollamaEmbedder() *embeddings.EmbedderImpl {
	// Create a new ollama model with the model name "nomic-embed-text:latest"
	ollamaEmbedderModel, err := ollama.New(
		ollama.WithModel("nomic-embed-text:latest"),
		ollama.WithServerURL(ollamaServer))
	if err != nil {
		log.Error("Failed to create ollama model: %v", err)
	}
	// Use the created ollama model to create a new embedder
	ollamaEmbedder, err := embeddings.NewEmbedder(ollamaEmbedderModel)
	if err != nil {
		log.Error("Failed to create ollama embedder: %v", err)
	}
	return ollamaEmbedder
}

//func getOllamaMistral() *ollama.LLM {
//	// Create a new ollama model with the model name "mistral"
//	llm, err := ollama.New(
//		ollama.WithModel("mistral"),
//		ollama.WithServerURL(ollamaServer))
//	if err != nil {
//		log.Error("Failed to create ollama model: %v", err)
//	}
//	return llm
//}
//
//func getOllamaLlama2() *ollama.LLM {
//	// Create a new ollama model with the model name "llama2-chinese:13b"
//	llm, err := ollama.New(
//		ollama.WithModel("llama2-chinese:13b"),
//		ollama.WithServerURL(ollamaServer))
//	if err != nil {
//		log.Error("Failed to create ollama model: %v", err)
//	}
//	return llm
//}
//
//func getStore() *qdrant.Store {
//	// Parse URL
//	qdUrl, err := url.Parse(qdrantUrl)
//	if err != nil {
//		log.Error("Failed to parse URL: %v", err)
//	}
//	// Create a new qdrant store
//	store, err := qdrant.New(
//		qdrant.WithURL(*qdUrl),                    // Set URL
//		qdrant.WithAPIKey(""),                     // Set API key
//		qdrant.WithCollectionName(collectionName), // Set collection name
//		qdrant.WithEmbedder(getollamaEmbedder()),  // Set embedder
//	)
//	if err != nil {
//		log.Error("Failed to create qdrant store: %v", err)
//	}
//	return &store
//}
//
//func storeDocs(docs []schema.Document, store *qdrant.Store) error {
//	// If the document array length is greater than 0
//	if len(docs) > 0 {
//		// Add documents to the store
//		_, err := store.AddDocuments(context.Background(), docs)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}
//
//func useRetriever(store *qdrant.Store, prompt string, topk int) ([]schema.Document, error) {
//	// Set vector options
//	optionsVector := []vectorstores.Option{
//		vectorstores.WithScoreThreshold(0.80), // Set score threshold
//	}
//
//	// Create a retriever
//	retriever := vectorstores.ToRetriever(store, topk, optionsVector...)
//
//	// Search
//	docRetrieved, err := retriever.GetRelevantDocuments(context.Background(), prompt)
//
//	if err != nil {
//		return nil, fmt.Errorf("Failed to retrieve documents: %v", err)
//	}
//
//	// Return the retrieved documents
//	return docRetrieved, nil
//}
//
//func GetAnswer(ctx context.Context, llm llms.Model, docRetrieved []schema.Document, prompt string) (string, error) {
//	// Create a new chat message history
//	history := memory.NewChatMessageHistory()
//	// Add retrieved documents to history
//	for _, doc := range docRetrieved {
//		history.AddAIMessage(ctx, doc.PageContent)
//	}
//	// Use history to create a new conversation buffer
//	conversation := memory.NewConversationBuffer(memory.WithChatHistory(history))
//
//	executor := agents.NewExecutor(
//		agents.NewConversationalAgent(llm, nil),
//		nil,
//		agents.WithMemory(conversation),
//		//agents.WithCallbacksHandler(callbacks.StreamLogHandler{}),
//	)
//
//	executor.CallbacksHandler = callbacks.StreamLogHandler{}
//
//	// Set chain call options
//	options := []chains.ChainCallOption{
//		chains.WithTemperature(0.8),
//		//chains.WithCallback(callbacks.StreamLogHandler{}),
//	}
//
//	// Run chain
//	res, err := chains.Run(ctx, executor, prompt, options...)
//	if err != nil {
//		return "", err
//	}
//
//	return res, nil
//}
//
//func Translate(llm llms.Model, text string) (string, error) {
//	completion, err := llms.GenerateFromSinglePrompt(
//		context.TODO(),
//		llm,
//		"Translate the following sentence into Chinese. Only reply with the translated content, without any additional information. The English content to be translated is:\n"+text,
//		llms.WithTemperature(0.8))
//	if err != nil {
//		return "", err
//	}
//	return completion, nil
//}