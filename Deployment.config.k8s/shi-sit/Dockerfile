FROM golang:1.25.0 AS builder

WORKDIR /app

# GOPRIVATE makes the go tool skip the public proxy for this module path and
# fetch it straight from GitHub over git, which is why the token has to be here.
# The token stays in this builder stage only: the final alpine stage copies just
# the compiled binary and the .env, so nothing lands in the shipped image.
ARG GIT_TOKEN
ENV GOPRIVATE=github.com/prime-solution-dev/*
RUN if [ -n "$GIT_TOKEN" ]; then \
        git config --global url."https://x-access-token:${GIT_TOKEN}@github.com/".insteadOf "https://github.com/"; \
    fi

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

RUN go build -o prime-wms-erp-core ./cmd

FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates nano

COPY --from=builder /app/prime-wms-erp-core .
COPY --from=builder /app/.env .

EXPOSE 9115

CMD ["./prime-wms-erp-core"]
