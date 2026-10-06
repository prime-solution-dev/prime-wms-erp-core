# syntax=docker/dockerfile:1
FROM golang:1.25 AS builder

WORKDIR /app

ENV GOPRIVATE=github.com/prime-solution-dev/*
ENV GOPROXY=https://proxy.golang.org,direct

COPY go.mod go.sum ./

RUN --mount=type=secret,id=github_token,required=true \
    --mount=type=tmpfs,target=/root \
    set -eu; \
    test -s /run/secrets/github_token; \
    umask 077; \
    printf 'machine github.com\nlogin x-access-token\npassword %s\n' \
        "$(cat /run/secrets/github_token)" > /root/.netrc; \
    for i in 1 2 3 4 5; do \
        if go mod download; then \
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

RUN go build -o prime-wms-erp-core ./cmd

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates nano

COPY --from=builder /app/prime-wms-erp-core .
COPY --from=builder /app/.env .

EXPOSE 9115

CMD ["./prime-wms-erp-core"]
