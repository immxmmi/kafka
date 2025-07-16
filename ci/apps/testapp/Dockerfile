FROM golang:1.22 AS builder
WORKDIR /app

# Systembibliotheken für CGo + librdkafka
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential pkg-config librdkafka-dev ca-certificates && \
    rm -rf /var/lib/apt/lists/*

COPY src/ ./
RUN go mod download

# Aktivieren von CGO + dynamisches Linking
ENV CGO_ENABLED=1 \
    GOOS=linux \
    CGO_CFLAGS_ALLOW="-Wno-error=unused-command-line-argument"

RUN go build -tags dynamic -o kafka-access-check main.go

# Runtime
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends librdkafka1 ca-certificates && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /
COPY --from=builder /app/kafka-access-check /
RUN chmod +x /kafka-access-check
CMD ["/kafka-access-check"]