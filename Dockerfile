FROM golang:1.24.0 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

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
