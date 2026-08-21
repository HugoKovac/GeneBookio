# syntax=docker/dockerfile:1.7
FROM node:22-alpine AS build

WORKDIR /app

# Deps layer: only invalidated by package.json/package-lock.json changes.
COPY package.json package-lock.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci

COPY . .

# Overridable at build time (see docker-compose.yaml). Defaults match plain
# local usage (npm run build) — a Dokploy deploy that path-routes this app
# under e.g. /admin/ passes VITE_BASE_PATH=/admin/ and VITE_API_URL=/admin/api.
ARG VITE_BASE_PATH=/
ARG VITE_API_URL=/api
ENV VITE_BASE_PATH=${VITE_BASE_PATH}
ENV VITE_API_URL=${VITE_API_URL}

RUN npm run build

FROM nginx:1.27-alpine

ARG VITE_BASE_PATH=/
ARG VITE_API_URL=/api

COPY --from=build /app/dist /tmp/dist
COPY nginx.conf.template /tmp/nginx.conf.template

# Resolved once at image build time (both ARGs are already fixed into the
# built JS/CSS above, so there's no need for runtime envsubst): the dist is
# physically nested under its base path so plain root-relative serving works,
# and the api proxy location/prefix-strip is baked into the same path.
RUN set -e; \
    base_path="${VITE_BASE_PATH:-/}"; \
    case "$base_path" in */) ;; *) base_path="$base_path/";; esac; \
    api_path="${VITE_API_URL:-/api}"; \
    case "$api_path" in */) ;; *) api_path="$api_path/";; esac; \
    mkdir -p "/usr/share/nginx/html$base_path"; \
    cp -r /tmp/dist/. "/usr/share/nginx/html$base_path"; \
    sed -e "s#__BASE_PATH__#$base_path#g" -e "s#__API_PATH__#$api_path#g" \
        /tmp/nginx.conf.template > /etc/nginx/conf.d/default.conf; \
    rm -rf /tmp/dist /tmp/nginx.conf.template

EXPOSE 80
