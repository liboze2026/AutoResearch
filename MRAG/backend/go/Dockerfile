FROM golang:1.22 AS builder
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/mrag-server ./cmd/server

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates openssh-client tzdata python3 \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /bin/mrag-server /app/mrag-server
COPY --from=builder /app/migrations /app/migrations
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh
EXPOSE 8080
CMD ["/app/docker-entrypoint.sh"]
