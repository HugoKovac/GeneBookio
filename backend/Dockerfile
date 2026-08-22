# syntax=docker/dockerfile:1.7
FROM golang:1.26.4 AS build

WORKDIR /app

# Deps layer: only invalidated by go.mod/go.sum changes, and shared by every
# service built from this Dockerfile (they all pass the same go.mod/go.sum).
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY pkg ./pkg

ARG CMD="api"

# Cache mounts persist across builds (including --no-cache ones) and are
# shared by every service, so only the actually-changed packages recompile.
# No -a/-installsuffix: unnecessary with CGO_ENABLED=0 and just forces a full
# stdlib rebuild every time.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/${CMD}

FROM scratch

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /app/server /server

ADD ./docs ./docs

# keys/ is gitignored (real private keys, not something a git-built image
# should bake in) — mount it as a volume at /keys instead (see
# backend/docker-compose.yaml for local dev, root docker-compose.yaml's
# keys-init service for the Dokploy deploy). PUBLIC_KEY_PATH etc. in .env are
# already relative ("keys/...") which resolves against this scratch image's
# cwd ("/") to /keys/....

ENTRYPOINT [ "/server" ]
