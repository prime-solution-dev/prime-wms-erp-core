FROM golang:1.23.2 AS builder

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

RUN apk add --no-cache ca-certificates nano tzdata \
    && cp /usr/share/zoneinfo/Asia/Bangkok /etc/localtime \
    && echo "Asia/Bangkok" > /etc/timezone

COPY --from=builder /app/wms-erp-core .
COPY --from=builder /app/.env .

EXPOSE 9115

CMD ["./wms-erp-core"]
