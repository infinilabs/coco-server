# Coco AI - Connect & Collaborate

**Tagline**: _"Coco AI - search, connect, collaborate – all in one place."_

Coco AI is a unified search platform that connects all your enterprise applications and data—Google Workspace, Dropbox, Confluent Wiki, GitHub, and more—into a single, powerful search interface. This repository contains the **Coco App**, built for both **desktop and mobile**. The app allows users to search and interact with their enterprise data across platforms.


## Vision

At Coco, we aim to streamline workplace collaboration by centralizing access to enterprise data. The Coco App provides a seamless, cross-platform experience, enabling teams to easily search, connect, and collaborate within their workspace.

## Use Cases

- **Unified Search Across Platforms**: Coco integrates with all your enterprise apps, letting you search documents, conversations, and files across Google Workspace, Dropbox, GitHub, etc.
- **Cross-Platform Access**: The app is available for both desktop and mobile, so you can access your workspace from anywhere.
- **Seamless Collaboration**: Coco's search capabilities help teams quickly find and share information, improving workplace efficiency.
- **Simplified Data Access**: By removing the friction between various tools, Coco enhances your workflow and increases productivity.



```
curl -fsSL https://ollama.com/install.sh | sh
ollama pull nomic-embed-text:latest
ollama pull llama2-chinese:13b
ollama pull llama3.2:latest
ollama pull llama3.2:1b
ollama pull mistral:latest

OLLAMA_HOST=0.0.0.0:11434 ollama serve
```

```
docker pull qdrant/qdrant
docker run -itd --name qdrant -p 6333:6333 qdrant/qdrant
curl -X PUT http://localhost:6333/collections/langchaingo-ollama-rag   -H 'Content-Type: application/json'   --data-raw '{
    "vectors": {
      "size": 768,
      "distance": "Dot"
    }
  }'
```