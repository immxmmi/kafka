FROM golang:1.22 AS builder
WORKDIR /app/src
COPY src/ ./
RUN go mod download
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential pkg-config librdkafka-dev ca-certificates && \
    rm -rf /var/lib/apt/lists/*

ENV CGO_ENABLED=1 \
    GOOS=linux \
    CGO_CFLAGS_ALLOW="-Wno-error=unused-command-line-argument"

RUN go build -tags dynamic -o ../kafka-consumer main.go

FROM debian:bookworm-slim
WORKDIR /
COPY --from=builder /app/kafka-consumer /
COPY --from=builder /app/src/static /static
RUN chmod +x /kafka-consumer
RUN apt-get update && apt-get install -y --no-install-recommends librdkafka1 ca-certificates && \
    rm -rf /var/lib/apt/lists/*
EXPOSE 8080
CMD ["/kafka-consumer"]