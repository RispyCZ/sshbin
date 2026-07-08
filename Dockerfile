# syntax=docker/dockerfile:1

# Stage 1: build the React SPA with the Vite+ (vp) toolchain.
FROM node:22-slim AS web
# vp is a Rust binary that needs system CA certs for HTTPS (registry) downloads;
# node:22-slim ships without them.
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app/web
RUN npm install -g vite-plus
COPY web/ ./
RUN vp install && vp build

# Stage 2: build the static Go binary with the SPA embedded (//go:embed all:dist).
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# The freshly built dist wins over anything copied in (also excluded by .dockerignore).
COPY --from=web /app/web/dist ./web/dist
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /sshbin ./cmd/sshbin

# Stage 3: minimal non-root runtime. sqlite is pure Go (modernc.org/sqlite),
# so the static binary needs no libc.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /sshbin /sshbin
EXPOSE 2022 8080
ENTRYPOINT ["/sshbin"]
