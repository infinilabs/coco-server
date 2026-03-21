FROM debian:bookworm-slim

ENV TIKA_VERSION=3.2.3

RUN apt-get update && apt-get install -y --no-install-recommends \
        openjdk-17-jre-headless \
        chromium \
        libreoffice-core \
        libreoffice-writer \
        libreoffice-impress \
        libreoffice-calc \
        wget \
        curl \
        ca-certificates \
        fonts-liberation \
        fonts-noto-cjk \
    && rm -rf /var/lib/apt/lists/*

RUN wget -q \
        "https://archive.apache.org/dist/tika/${TIKA_VERSION}/tika-server-standard-${TIKA_VERSION}.jar" \
        -O /opt/tika-server.jar

RUN mkdir -p /opt/pigo && \
    wget -q \
        "https://raw.githubusercontent.com/esimov/pigo/master/cascade/facefinder" \
        -O /opt/pigo/facefinder

WORKDIR /app

COPY bin/coco-linux-amd64 ./coco
COPY bin/coco.yml          ./coco.yml
RUN chmod +x ./coco

COPY docker-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# API port / Web UI port
EXPOSE 2900 9000

ENTRYPOINT ["/entrypoint.sh"]
