---
weight: 10
title: "Installation"
asciinema: true
---

# Getting Started

## Prerequisite

- [Docker](https://docs.docker.com/engine/install/)
- [Ollama](https://ollama.com/)
- [Easysearch](https://hub.docker.com/r/infinilabs/easysearch/)

## Quickstart Script

### Download the Server
   Get the appropriate [executables](https://github.com/infinilabs/coco-server/releases/) for your platform.

### Run the Quick Setup
   Simply execute the installation script:

```
./Install.sh
```

## Manual Installation

Follow these steps for a manual setup:


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
ollama pull llama3.2
ollama pull llama3.2-vision
ollama pull llama2-chinese:13b
ollama pull mistral:latest
ollama pull nomic-embed-text:latest
```

### Easysearch

Install Easysearch
```
docker run -itd --name easysearch -p 9200:9200 infinilabs/easysearch:1.8.3-265
```

Get the bootstrap password of the Easysearch:
```
docker logs easysearch | grep "admin:"
```

### Coco AI

Modify `coco.yml` with correct `env` settings, or start the coco server with the correct environments like this:

```
➜  coco git:(main) ✗ OLLAMA_MODEL=llama3.2:1b ES_PASSWORD=45ff432a5428ade77c7b  ./bin/coco
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

Enjoy~