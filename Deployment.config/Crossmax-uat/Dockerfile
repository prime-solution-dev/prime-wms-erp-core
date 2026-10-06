FROM golang:1.25 AS builder

ARG GIT_TOKEN

WORKDIR /app

ENV GOPRIVATE=github.com/prime-solution-dev/*
ENV GOPROXY=https://proxy.golang.org,direct

RUN git config --global url."https://${GIT_TOKEN}@github.com/".insteadOf "https://github.com/"

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
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

RUN go build -o wms-erp-core ./cmd

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates nano

COPY --from=builder /app/wms-erp-core .
COPY --from=builder /app/.env .

EXPOSE 9115

CMD ["./wms-erp-core"]
