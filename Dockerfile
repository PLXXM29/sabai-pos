# Sabai POS ships as a single container: one Go binary that serves both the API
# and the compiled web app, embedded into the executable at build time.
#
# Why one image rather than the conventional nginx-plus-API pair — the refresh
# token is an httpOnly cookie, and a split origin would mean SameSite=None (which
# Safari drops) plus a CORS allow-list to keep correct. See backend/internal/web.
#
#   docker build -t sabai-pos .
#   docker run --rm -p 8080:8080 -e DATABASE_URL=... -e JWT_ACCESS_SECRET=... sabai-pos

# ── stage 1 · the web app ───────────────────────────────────────────────────────
FROM node:22-alpine AS ui
WORKDIR /ui
# Manifests first: dependency installs are cached until they actually change.
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
# `npm run build` type-checks before bundling, so a type error fails the image
# build rather than shipping.
RUN npm run build

# ── stage 2 · the server ────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
# Swap the committed placeholder page for the real bundle before compiling; the
# //go:embed in internal/web picks up whatever is in this directory.
RUN rm -rf internal/web/dist
COPY --from=ui /ui/dist ./internal/web/dist
# Fail loudly here rather than shipping an image that serves the placeholder.
RUN test -d internal/web/dist/assets || (echo "UI bundle missing from build context" && exit 1)
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/seed ./cmd/seed

# ── stage 3 · runtime ───────────────────────────────────────────────────────────
# distroless/static: no shell, no package manager, non-root. The binary is fully
# static and carries its own tzdata, so there is nothing else it needs.
FROM gcr.io/distroless/static-debian12:nonroot
ARG VERSION=dev
ENV APP_VERSION=${VERSION} \
    HTTP_PORT=8080
COPY --from=build /out/api /api
COPY --from=build /out/seed /seed
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
