# ---------- Build stage ----------
FROM golang:1.23-bookworm AS build
WORKDIR /app

# Säkerställ rätt verktygskedja om go.mod kräver nyare patchnivå
ENV GOTOOLCHAIN=auto

# Robust modulhämtning (kan override:as via build-args i GH Actions)
ARG GOPROXY_DEFAULT=https://proxy.golang.org,direct
ENV GOPROXY=$GOPROXY_DEFAULT

# Kopiera mod-filer först för bättre cache
COPY go.mod go.sum ./
# Visa env och hämta moduler med extra loggar (-x)
RUN go env && go mod download -x

# Kopiera resten av koden
COPY . .

# CGO krävs för github.com/mattn/go-sqlite3
ENV CGO_ENABLED=1
RUN apt-get update && apt-get install -y --no-install-recommends \
    build-essential ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Bygg endast main i rot (viktigt: inte ./... eftersom det ger "multiple packages to non-directory")
RUN go build -ldflags="-s -w" -o /xmatches .

# ---------- Runtime stage ----------
FROM debian:bookworm-slim

# Tidszoner + cert (bra i container)
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Körbar binär
COPY --from=build /xmatches /usr/local/bin/xmatches

# Standard-ENV (kan override:as)
ENV ADDR=:8080
ENV DB_PATH=/data/xmatches.db

# Persistent volym för SQLite
VOLUME ["/data"]

EXPOSE 8080
CMD ["xmatches"]
