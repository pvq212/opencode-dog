FROM golang:1.24-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/opencode-gitlab-bot ./cmd/server

FROM node:22-bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata git openssh-client golang && \
    rm -rf /var/lib/apt/lists/* && \
    addgroup --system appgroup && \
    adduser --system --ingroup appgroup appuser

RUN go install github.com/opencode-ai/opencode@latest 2>/dev/null || true

WORKDIR /app

COPY --from=builder /bin/opencode-gitlab-bot /app/opencode-gitlab-bot
COPY migrations/ /app/migrations/

RUN mkdir -p /app/config /home/appuser/.ssh && \
    chown -R appuser:appgroup /app /home/appuser

USER appuser

ENV OPENCODE_CONFIG_DIR=/app/config
ENV HOME=/home/appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/opencode-gitlab-bot"]
