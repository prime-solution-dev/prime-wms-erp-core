FROM golang:1.25.0

WORKDIR /app

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
COPY ./cmd/.env .env

RUN go mod tidy

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN go build -o wms-document-core ./cmd

RUN apt-get update && apt-get install -y \
    ca-certificates \
    fontconfig \
    nano \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir -p /usr/share/fonts/truetype/sukhumvit \
    && cp /app/assets/fonts/*.ttf /usr/share/fonts/truetype/sukhumvit/ \
    && fc-cache -fv

ENV FONT_PATH_PREFIX=/app/

EXPOSE 9106

CMD ["./wms-document-core"]
