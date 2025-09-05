# ---------- Build stage ----------
FROM golang:1.22-bookworm AS build
WORKDIR /app

# Cache vänligt: hämta moduler först
COPY go.mod go.sum ./
RUN go mod download

# Kopiera resten
COPY . .

# CGO krävs för mattn/go-sqlite3
ENV CGO_ENABLED=1
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Bygg binär (migrations + web bäddas in via go:embed)
RUN go build -ldflags="-s -w" -o /xmatches ./...

# ---------- Runtime stage ----------
FROM debian:bookworm-slim

# Tidszoner + cert
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Körbar binär
COPY --from=build /xmatches /usr/local/bin/xmatches

# Standard-ENV
ENV ADDR=:8080
ENV DB_PATH=/data/xmatches.db

# Persistent volym för SQLite
VOLUME ["/data"]

EXPOSE 8080
CMD ["xmatches"]
