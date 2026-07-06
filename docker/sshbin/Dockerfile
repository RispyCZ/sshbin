ARG NODE_IMAGE=node:24-slim GOLANG_IMAGE=golang:1.26.3 RUNTIME_IMAGE=gcr.io/distroless/static-debian13
FROM ${NODE_IMAGE} AS frontend_build
ENV CI=true
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/*
RUN corepack enable
WORKDIR /build
COPY web/ ./
RUN pnpm install --frozen-lockfile && pnpm build

FROM ${GOLANG_IMAGE} AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend_build /build/dist/ web/dist/
RUN CGO_ENABLED=0 go build -o /out/sshbin ./cmd/sshbin

FROM ${RUNTIME_IMAGE}
COPY --from=build /out/sshbin /sshbin
WORKDIR /data
ENTRYPOINT ["/sshbin"]