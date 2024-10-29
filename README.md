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


## Getting Started

### Ollama

Install Ollama
```
curl -fsSL https://ollama.com/install.sh | sh
```

Start Ollama server
```
OLLAMA_HOST=0.0.0.0:11434 ollama serve
```

Pull the following models
```
ollama pull nomic-embed-text:latest
ollama pull llama2-chinese:13b
ollama pull llama3.2:latest
ollama pull llama3.2:1b
ollama pull mistral:latest
```

### Easysearch

Install Easysearch
```
docker run -itd --name easysearch -p 9200:9200 infinilabs/easysearch:1.8.3-265
```

Setup Easysearch
```
curl -X PUT -u admin:YOUR_PASSWORD https://localhost:9200/coco   -H 'Content-Type: application/json'   --data-raw '{
   "settings": {
     "index.knn": true
   },
   "mappings": {
     "properties": {
       "content_embedding": {
         "type": "knn_dense_float_vector",
         "knn": {
           "dims": 50,
           "model": "lsh",
           "similarity": "cosine",
           "L": 99,
           "k": 1
         }
       }
     }
   }
 }'
```

### Coco AI

```
➜  coco git:(main) ✗ ./bin/coco
   ___  ___  ___  ___     _     _____
  / __\/___\/ __\/___\   /_\    \_   \
 / /  //  // /  //  //  //_\\    / /\/
/ /__/ \_// /__/ \_//  /  _  \/\/ /_
\____|___/\____|___/   \_/ \_/\____/
[COCO] Coco AI - search, connect, collaborate – all in one place.
[COCO] 1.0.0_SNAPSHOT#001, 2024-10-23 08:37:05, 2025-12-31 10:10:10, 9b54198e04e905406db90d145f4c01fca0139861
[10-23 17:17:36] [INF] [env.go:179] configuration auto reload enabled
[10-23 17:17:36] [INF] [env.go:185] watching config: /Users/medcl/go/src/infini.sh/coco/config
[10-23 17:17:36] [INF] [app.go:285] initializing coco, pid: 13764
[10-23 17:17:36] [INF] [app.go:286] using config: /Users/medcl/go/src/infini.sh/coco/coco.yml
[10-23 17:17:36] [INF] [api.go:196] local ips: 192.168.3.10
[10-23 17:17:36] [INF] [api.go:360] api listen at: http://0.0.0.0:2900
[10-23 17:17:36] [INF] [module.go:136] started module: api
[10-23 17:17:36] [INF] [module.go:155] started plugin: statsd
[10-23 17:17:36] [INF] [module.go:161] all modules are started
[10-23 17:17:36] [INF] [instance.go:78] workspace: /Users/medcl/go/src/infini.sh/coco/data/coco/nodes/csai3njq50k2c4tcb4vg
[10-23 17:17:36] [INF] [app.go:511] coco is up and running now.
```


## Assistant API Reference

### Retrieve Chat History (sessions)

```shell
//request
curl -XGET http://localhost:2900/chat/_history'

//response
{
    "sessions": [
        {
            "session_id": "csai3njq50k2c4tcb4vg",
            "created": "2024-10-23T17:17:36.000Z",
            "updated": "2024-10-23T17:17:36.000Z",
            "status": "active",
            "title": "John Doe",
            "summary": "Hello",
        }
    ]
}
```

### Create a Chat Session

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:2900/chat/_new'

//response
{
    "session_id": "csai3njq50k2c4tcb4vg",
    "created": "2024-10-23T17:17:36.000Z",
    "status": "active",
}
```

### Open a Existing Chat Session

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:2900/chat/csai3njq50k2c4tcb4vg/_open'

//response
{
    "session_id": "csai3njq50k2c4tcb4vg",
    "status": "active",
}
```


### Retrieve a Chat History

```shell
//request
curl -XGET http://localhost:2900/chat/csai3njq50k2c4tcb4vg/_history'

//response
{
    "messages": [
        {
            "sequence": 1234,
            "created": "2024-10-23T17:17:36.000Z",
            "message": "Hello",
            "response": "Hi, how can I help you?"
        }
    ]
}
```

### Send a Message

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:2900/chat/csai3njq50k2c4tcb4vg/_send' -d '{"message":"Hello"}'

//response
{
    "sequence": 1234, //sort each chat message by this field
    "created": "2024-10-23T17:17:36.000Z",
    "message": "Hello",
    "response": "Hi, how can I help you?"
}
```

### Close a Chat Session

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:2900/chat/csai3njq50k2c4tcb4vg/_close'

//response
{
    "session_id": "csai3njq50k2c4tcb4vg",
    "status": "closed",
}
```
