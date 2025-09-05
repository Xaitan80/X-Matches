# ---------- Build stage ----------
# syntax=docker/dockerfile:1.7
FROM golang:1.23-bookworm AS build
WORKDIR /src

# Säkerställ rätt verktygskedja om go.mod kräver nyare patchnivå
ENV GOTOOLCHAIN=auto

# Robust modulhämtning (kan override:as via build-args i GH Actions)
ARG GOPROXY_DEFAULT=https://proxy.golang.org,direct
ENV GOPROXY=$GOPROXY_DEFAULT

# Kopiera mod-filer först för bättre cache
COPY go.mod go.sum ./
# Hämta moduler med BuildKit-cache för snabbare builds
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go env && go mod download -x

# Kopiera resten av koden
COPY . .

# Bygg statisk binär utan CGO (pure-Go sqlite)
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -o /out/xmatches .

# Skapa tomma kataloger att kopiera till runtime (distroless saknar shell)
RUN mkdir -p /out/app /out/data

# ---------- Runtime stage ----------
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Körbar binär
COPY --from=build /out/xmatches /usr/local/bin/xmatches

# Skapa skrivbara kataloger (kopieras från build)
COPY --from=build /out/data /data
COPY --from=build /out/app /app

# Standard-ENV (kan override:as)
ENV ADDR=:8080
ENV DB_PATH=/data/xmatches.db

# Persistent volym för SQLite
VOLUME ["/data"]

EXPOSE 8080
USER nonroot:nonroot
CMD ["/usr/local/bin/xmatches"]
