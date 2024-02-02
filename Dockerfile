FROM golang:1.20-alpine AS builder

WORKDIR /app
RUN apk add git && git clone https://github.com/bincooo/single-proxy.git .
RUN go mod tidy && GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o server -trimpath

FROM alpine:3.19.0
WORKDIR /app

COPY --from=builder /app/server ./server
COPY --from=builder /app/config.yaml ./config.yaml

EXPOSE 8080

ENTRYPOINT ["sh","-c","./server"]
