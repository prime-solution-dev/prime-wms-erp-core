FROM golang:1.25.0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

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
