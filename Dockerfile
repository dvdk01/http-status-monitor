
FROM golang:1.24-alpine AS builder
ENV GOOS=linux
ENV GOARCH=amd64



WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o http-status-monitor ./cmd/http-status-monitor


FROM alpine:3.22

WORKDIR /app
COPY --from=builder /app/http-status-monitor .

ENTRYPOINT ["./http-status-monitor"] 