---
weight: 10
title: "Installation"
asciinema: true
---

# Coco Server User Guide

- Run Coco Server with Docker
- Manual install Coco Server


## 1. Quick Start (Recommended for Most Users)

This method is the simplest way to get Coco Server running. It uses Docker-managed volumes, which handles data persistence automatically without requiring manual directory setup on your host machine.

**Command:**
```bash
docker run -d \
  --name cocoserver \
  -p 9000:9000 \
  -v data:/app/easysearch/data \
  -v config:/app/easysearch/config \
  -v logs:/app/easysearch/logs \
  infinilabs/coco:0.7.0
```

> 🔒 **SECURITY WARNING**
>
> For initial setup convenience, the password is set to `"coco-server"`. It is **highly recommended** to replace this with a strong, unique password before running the command for the first time.

**After running the command:**
*   Coco Server will be running in the background. Access the Web UI at `http://localhost:9000`.
*   Your data, configuration, and logs are safely stored in Docker volumes named `data`, `config`, and `logs`.

---

## 2. Advanced Start (Using Host Directories)

This method provides direct access to the data, configuration, and log files on your host machine. It's suitable for users who want to easily edit configuration files or manage data directly.

**Step 1: Prepare Host Directories and Environment File**

First, create the necessary directories and a `.env` file for your password.

```bash
# Create the parent directory for all Coco Server files.
# We recommend using a standard location like /data/cocoserver.
sudo mkdir -p /data/cocoserver/{data,logs,config}

# Create an environment file to store your password securely.
# Replace "$(openssl rand -hex 10)" with a strong, secret password.
echo "EASYSEARCH_INITIAL_ADMIN_PASSWORD=$(openssl rand -hex 10)" | sudo tee /data/cocoserver/.env > /dev/null
```

**Step 2: Initialize Configuration from the Image (One-time Setup)**

This clever command runs a temporary container to copy the default configuration files from the image into your newly created local `config` directory.

```bash
docker run --rm \
  -v /data/cocoserver/config:/tmp/config \
  infinilabs/coco:0.7.0 \
  cp -a /app/easysearch/config/. /tmp/config/
```
*   `--rm`: Automatically removes the container after it exits.
*   `-v /data/cocoserver/config:/tmp/config`: Mounts your local config directory into a temporary path inside the container.
*   `cp -a ...`: The command executed inside the container. It copies all contents from the image's config directory to the mounted host directory.

**Step 3: Set Directory Permissions**

The container runs with a specific user (UID 602). You must grant this user write permissions to the host directories.

> **Note:** This step might be unnecessary on some Docker environments like Docker Desktop for Mac. Test without it first if you are unsure.

```bash
sudo chown -R 602:602 /data/cocoserver
```

**Step 4: Run the Coco Server Container**

Now, start the main container, mounting your prepared host directories and using the `.env` file for the password.

```bash
docker run -d \
  --name cocoserver \
  --hostname coco-server \
  --restart unless-stopped \
  --env-file /data/cocoserver/.env \
  -m 4g --cpus="2" \
  -p 9000:9000 \
  -v /data/cocoserver/data:/app/easysearch/data \
  -v /data/cocoserver/config:/app/easysearch/config \
  -v /data/cocoserver/logs:/app/easysearch/logs \
  -e ES_JAVA_OPTS="-Xms2g -Xmx2g" \
  infinilabs/coco:0.7.0
```
*   `--env-file`: Securely loads environment variables (like your password) from the `.env` file.

---

## 3. Upgrading Coco Server

Follow these steps to upgrade to a new version while preserving all your data. The procedure is the same whether you used the Quick Start or Advanced Start method.

**Step 1: Pull the New Image**
```bash
# Replace '0.7.0' with the new version tag.
docker pull infinilabs/coco:0.7.0
```

**Step 2: Stop and Remove the Old Container**
```bash
docker stop cocoserver && docker rm cocoserver
```

**Step 3: Start a New Container with the Same Volumes/Mounts**

Re-run your original `docker run` command, but update the image tag to the new version. **It is crucial to use the exact same `-v` volume or bind mount paths as before.**

*   **If you used Quick Start**, re-run the Quick Start command with the new image tag.
*   **If you used Advanced Start**, re-run the Advanced Start command with the new image tag.

---

## 4. Full Cleanup and Removal

These commands will completely remove the Coco Server container and its data.

> **🛑 DANGER ZONE 🛑**
>
> The following commands will **permanently delete all your Coco Server data**. This action cannot be undone. Proceed with caution.

**Step 1: Stop and Remove the Container**
```bash
docker stop cocoserver && docker rm cocoserver
```

**Step 2: Remove Data, Config, and Logs**

*   **If you used Quick Start (Docker Volumes):**
    ```bash
    docker volume rm data config logs
    ```
*   **If you used Advanced Start (Host Directories):**
    ```bash
    sudo rm -rf /data/cocoserver
    ```

**Step 3: Remove the Docker Image**
```bash
docker rmi infinilabs/coco:0.7.0
```

## Manual Installation

Follow these steps for a manual setup:

### Easysearch

Install Easysearch

```bash
docker run -itd --name easysearch -p 9200:9200 infinilabs/easysearch:1.8.3-265
```

Get the bootstrap password of the Easysearch:

```bash
docker logs easysearch | grep "admin:"
```

### Coco AI

Modify `coco.yml` with correct `env` settings, or start the coco server with the correct environments like this:

```bash
$ OLLAMA_MODEL=deepseek-r1:1.5b ES_PASSWORD=45ff432a5428ade77c7b  ./bin/coco
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
