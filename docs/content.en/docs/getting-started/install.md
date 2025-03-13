---
weight: 10
title: "Installation"
asciinema: true
---

# Getting Started

## Quickstart with Docker

```bash
# --------------------------------------------------
# Coco Server Docker Deployment Script and Commands
# --------------------------------------------------

# Quick Start Coco Server (using default configuration)
#
# Command Explanation:
#   docker run:       Create and run a new Docker container
#   -d:               Run the container in the background (detached mode)
#   --name cocoserver:  Assign a name to the container (cocoserver)
#   -p 9000:9000:     Map the container's port 9000 to the host's port 9000 (Web UI port)
#   infinilabs/coco:0.2.1-1998:  The Docker image name and tag (version) to use
docker run -d --name cocoserver -p 9000:9000 infinilabs/coco:0.2.1-1998

# Stop and Remove Coco Server Container
#
# Command Explanation:
#   docker stop cocoserver:   Stop the container named cocoserver
#   &&:                     Logical AND operator, ensures the previous command succeeds before executing the next
#   docker rm cocoserver:     Remove the container named cocoserver (only stopped containers can be removed)
docker stop cocoserver && docker rm cocoserver

# Remove Coco Server Docker Image
#
# Command Explanation:
#   docker rmi infinilabs/coco:0.2.1-1998: Remove the image named infinilabs/coco with the tag 0.2.1-1998
#   Note: You can only remove an image if no containers are using it.  If a container is using the image, you must first stop and remove the container.
docker rmi infinilabs/coco:0.2.1-1998

# --------------------------------------------------
# Customized Configuration to Start Coco Server
# (Adjust parameters according to your needs)
# --------------------------------------------------

# Create data and logs directories and set ownership for the Easysearch user (UID/GID 602).
mkdir -p $(pwd)/cocoserver/{data,logs}
sudo chown -R 602:602 $(pwd)/cocoserver

# Command Explanation:
#   docker run:       Create and run a new Docker container
#   -d:               Run the container in the background (detached mode)
#   --name cocoserver:  Assign a name to the container (cocoserver)
#   --hostname coco-server: Set the container's hostname to coco-server
#   --restart unless-stopped:  Set the container's restart policy to "restart unless manually stopped"
#   -m 4g:            Limit the container's memory usage to 4GB
#   --cpus="2":       Limit the number of CPU cores the container can use to 2
#   -p 9000:9000:     Map the container's port 9000 to the host's port 9000 (Web UI port)
#   -v $(pwd)/cocoserver/data:/app/easysearch/data:  Mount the host's ./cocoserver/data directory to the container's /app/easysearch/data directory (for data persistence)
#        - $(pwd) gets the absolute path of the current directory.
#        - Make sure the ./cocoserver/data directory exists. You can create it manually or use the `mkdir -p cocoserver/data` command.
#   -v $(pwd)/cocoserver/logs:/app/easysearch/logs:  Mount the host's ./cocoserver/logs directory to the container's /app/easysearch/logs directory (for storing logs)
#        - Make sure the ./cocoserver/logs directory exists. You can create it manually or use the `mkdir -p cocoserver/logs` command.
#   -e EASYSEARCH_INITIAL_ADMIN_PASSWORD=coco-server: Set the initial password for the Easysearch administrator to coco-server (Important: Change this to a strong password)
#   -e ES_JAVA_OPTS="-Xms2g -Xmx2g":  Set the JVM parameters for Easysearch:
#        - -Xms2g:  Set the initial JVM heap size to 2GB
#        - -Xmx2g:  Set the maximum JVM heap size to 2GB
#   infinilabs/coco:0.2.1-1998: The Docker image name and tag to use

docker run -d \
           --name cocoserver \
           --hostname coco-server \
           --restart unless-stopped \
           -m 4g \
           --cpus="2" \
           -p 9000:9000 \
           -v $(pwd)/cocoserver/data:/app/easysearch/data \
           -v $(pwd)/cocoserver/logs:/app/easysearch/logs \
           -e EASYSEARCH_INITIAL_ADMIN_PASSWORD=coco-server \
           -e ES_JAVA_OPTS="-Xms2g -Xmx2g" \
           infinilabs/coco:0.2.1-1998
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
ollama pull deepseek-r1:1.5b 
```

> Ollama is not required, you may use other online LLM services as your wish.

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
➜  coco git:(main) ✗ OLLAMA_MODEL=deepseek-r1:1.5b ES_PASSWORD=45ff432a5428ade77c7b  ./bin/coco
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

Once the Coco Server is running, you are ready to [setup it up](./setup.md) through UI based management console.