# Build context is the repo root (see docker-compose.yml), not backend/,
# so this Dockerfile can also COPY the root-level typst-version.txt below
# (single source of truth for the pinned typst version, see ADR-0012 and
# issue #55) — hence the backend/ prefixes on these COPYs.
FROM golang:1.26 AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/. .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

# Render (backend/internal/generation/render.go) shells out to the typst
# CLI, which distroless/static can't provide — it has no package manager
# and, unlike the statically-linked Go server, typst needs the
# Liberation Sans font template/cv.typ and cover-letter.typ set. A plain
# debian base with fonts-liberation trades distroless's minimalism for
# that, deliberately (see ADR-0012). The ATS-parsability check (see
# backend/internal/generation/parsability.go) shells out to pdftotext the
# same way, so poppler-utils is pinned to bookworm's version here for the
# same reason typst's release is pinned by version — the check's behavior
# shouldn't silently drift between environments.
#
# The pinned typst version comes from typst-version.txt (repo root), the
# single source of truth the tailor-cv skill's preflight check also reads
# (see issue #55) — bump the version there, not here.
FROM debian:bookworm-slim
COPY typst-version.txt /tmp/typst-version.txt
RUN . /tmp/typst-version.txt \
    && apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl xz-utils fonts-liberation poppler-utils=22.12.0-2+deb12u3 \
    && curl -fsSL "https://github.com/typst/typst/releases/download/v${TYPST_VERSION}/typst-x86_64-unknown-linux-musl.tar.xz" \
      -o /tmp/typst.tar.xz \
    && tar -xf /tmp/typst.tar.xz -C /tmp \
    && mv "/tmp/typst-x86_64-unknown-linux-musl/typst" /usr/local/bin/typst \
    && apt-get purge -y curl xz-utils \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/* /tmp/typst* /tmp/typst-version.txt
COPY --from=build /out/server /server
ENTRYPOINT ["/server"]
