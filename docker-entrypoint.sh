#!/bin/bash
set -e

# Start Tika server in the background
echo "[entrypoint] Starting Apache Tika ${TIKA_VERSION:-2.9.2} on port 9998..."
java -jar /opt/tika-server.jar --port 9998 &
TIKA_PID=$!

# Wait until Tika responds (up to 60s)
for i in $(seq 1 30); do
    if curl -sf http://127.0.0.1:9998/tika >/dev/null 2>&1; then
        echo "[entrypoint] Tika is ready."
        break
    fi
    if ! kill -0 "$TIKA_PID" 2>/dev/null; then
        echo "[entrypoint] Tika process exited unexpectedly." >&2
        exit 1
    fi
    echo "[entrypoint] Waiting for Tika... ($i/30)"
    sleep 2
done

# Forward signals to coco so graceful shutdown works
exec /app/coco "$@"
