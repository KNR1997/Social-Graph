# syntax=docker/dockerfile:1

# Pinned to the Go version in go.mod. Bump both together, or the build breaks
# in CI rather than on your machine.
FROM golang:1.25-alpine AS builder

WORKDIR /src

# Dependencies first, in their own layer, so code edits do not re-download them.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# Static binaries: the final stage has no libc.
# The migrate binary ships too, so a deploy can run migrations as its own step
# using the same embedded migration files the service was built from.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api && \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /out/api /api
COPY --from=builder /out/migrate /migrate

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/api"]
