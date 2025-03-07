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

```bash
# --------------------------------------------------
# Coco Server Docker 部署脚本及相关命令
# --------------------------------------------------

# 快速启动 Coco Server (使用默认配置)
#
# 命令解释:
#   docker run:       创建并运行一个新的 Docker 容器
#   -d:               在后台运行容器 (detached mode)
#   --name cocoserver:  为容器指定一个名称 (cocoserver)
#   -p 9000:9000:     将容器的 9000 端口映射到主机的 9000 端口 (Web UI 端口)
#   infinilabs/coco:0.2.0-1992:  使用的 Docker 镜像名称和标签 (版本号)
docker run -d --name cocoserver -p 9000:9000 infinilabs/coco:0.2.0-1992

# 停止并删除 Coco Server 容器
#
# 命令解释:
#   docker stop cocoserver:   停止名为 cocoserver 的容器
#   &&:                     逻辑与操作符，确保前一个命令成功执行后才执行下一个命令
#   docker rm cocoserver:     删除名为 cocoserver 的容器 (只有停止的容器才能被删除)
docker stop cocoserver && docker rm cocoserver

# 删除 Coco Server Docker 镜像
#
# 命令解释:
#   docker rmi infinilabs/coco:0.2.0-1992: 删除名为 infinilabs/coco，标签为 0.2.0-1992 的镜像
#   注意: 只有当没有容器使用该镜像时，才能删除镜像。如果有容器正在使用，需要先停止并删除容器。
docker rmi infinilabs/coco:0.2.0-1992

# --------------------------------------------------
# 个性化自定义配置启动 Coco Server
# (根据需求自行调整参数)
# --------------------------------------------------

# 命令解释:
#   docker run:       创建并运行一个新的 Docker 容器
#   -d:               在后台运行容器 (detached mode)
#   --name cocoserver:  为容器指定一个名称 (cocoserver)
#   --hostname coco-server: 设置容器的主机名为 coco-server
#   --restart unless-stopped:  设置容器的重启策略为 "除非手动停止，否则自动重启"
#   -m 4g:            限制容器的内存使用量为 4GB
#   --cpus="2":       限制容器可以使用的 CPU 核心数为 2
#   -p 9000:9000:     将容器的 9000 端口映射到主机的 9000 端口 (Web UI 端口)
#   -v $(pwd)/cocoserver/data:/app/easysearch/data:  将主机的 ./cocoserver/data 目录挂载到容器的 /app/easysearch/data 目录 (用于持久化数据)
#        - $(pwd) 获取当前目录的绝对路径
#        - 确保 ./cocoserver/data 目录存在。  你可以手动创建，或者使用 mkdir -p cocoserver/data 命令创建。
#   -v $(pwd)/cocoserver/logs:/app/easysearch/logs:  将主机的 ./cocoserver/logs 目录挂载到容器的 /app/easysearch/logs 目录 (用于存储日志)
#        - 确保 ./cocoserver/logs 目录存在。 你可以手动创建，或者使用 mkdir -p cocoserver/logs 命令创建。
#   -e EASYSEARCH_INITIAL_ADMIN_PASSWORD=coco-server: 设置 Easysearch 管理员的初始密码为 coco-server (重要: 建议修改为强密码)
#   -e ES_JAVA_OPTS="-Xms2g -Xmx2g":  设置 Easysearch 的 JVM 参数:
#        - -Xms2g:  设置 JVM 初始堆内存大小为 2GB
#        - -Xmx2g:  设置 JVM 最大堆内存大小为 2GB
#   infinilabs/coco:0.2.0-1992: 使用的 Docker 镜像名称和标签

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
           infinilabs/coco:0.2.0-1992
```

### Download the Server
   Get the appropriate [executables](https://coco.rs/) for your platform.

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
ollama pull deepseek-r1:1.5b 
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

Enjoy~
