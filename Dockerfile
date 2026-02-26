# syntax=docker/dockerfile:1

# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

# CGO is not needed; disable it so we get a fully static binary.
ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

WORKDIR /src

# Cache dependency downloads separately from the build layer.
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source and build.
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/installer ./cmd/main.go

# ── Stage 2: export ───────────────────────────────────────────────────────────
# A scratch image that only contains the compiled binary.
# docker-compose mounts /out to the host so the binary lands there directly.
FROM scratch AS export
COPY --from=builder /out/installer /installer
