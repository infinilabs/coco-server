FROM debian:bookworm-slim

RUN sed -i 's/deb.debian.org/mirrors.aliyun.com/g' /etc/apt/sources.list.d/debian.sources


RUN apt-get update && apt-get install -y --no-install-recommends \
        openjdk-17-jre-headless \
        chromium \
        libreoffice-core \
        libreoffice-writer \
        libreoffice-impress \
        libreoffice-calc \
        curl \
        ca-certificates \
        fonts-liberation \
        fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/*

COPY deps/tika-server-standard-3.2.3.jar /opt/tika-server.jar

RUN mkdir -p /opt/pigo
COPY deps/facefinder /opt/pigo/facefinder

WORKDIR /app

COPY bin/coco-linux-amd64 ./coco
COPY bin/coco.yml          ./coco.yml
COPY config/               ./config/
RUN chmod +x ./coco

COPY docker-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# API port / Web UI port
EXPOSE 2900 9000

ENTRYPOINT ["/entrypoint.sh"]
