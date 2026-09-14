# =========================================================
# Stage 1: Builder
# =========================================================

FROM golang:1.25.0 AS builder

WORKDIR /app

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

COPY go.mod go.sum ./

RUN set -e; \
    for i in 1 2 3 4 5; do \
        if GOPROXY=https://proxy.golang.org,direct go mod download; then \
            echo "Go modules downloaded successfully"; \
            break; \
        fi; \
        echo "go mod download failed - retry $i/5"; \
        sleep 10; \
        if [ "$i" = "5" ]; then \
            echo "go mod download failed after 5 attempts"; \
            exit 1; \
        fi; \
    done

COPY . .

COPY ./cmd/.env /app/.env

RUN go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/wms-document-core \
    ./cmd


# =========================================================
# Stage 2: Runtime
# =========================================================

FROM debian:bookworm-slim

WORKDIR /app

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        fontconfig \
        tzdata \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder \
    /out/wms-document-core \
    /app/wms-document-core

COPY --from=builder \
    /app/assets \
    /app/assets

# เก็บ .env สำหรับ Docker runtime
COPY --from=builder \
    /app/.env \
    /app/.env

RUN mkdir -p /usr/share/fonts/truetype/sukhumvit \
    && cp /app/assets/fonts/*.ttf /usr/share/fonts/truetype/sukhumvit/ \
    && fc-cache -f

ENV FONT_PATH_PREFIX=/app/

EXPOSE 9106

CMD ["./wms-document-core"]
