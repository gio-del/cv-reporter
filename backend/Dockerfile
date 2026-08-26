FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/server ./cmd/server

# Render (backend/internal/generation/render.go) shells out to the typst
# CLI, which distroless/static can't provide — it has no package manager
# and, unlike the statically-linked Go server, typst needs the
# Liberation Sans font template/cv.typ and cover-letter.typ set. A plain
# debian base with fonts-liberation trades distroless's minimalism for
# that, deliberately (see ADR-0012).
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl xz-utils fonts-liberation \
    && curl -fsSL https://github.com/typst/typst/releases/download/v0.15.1/typst-x86_64-unknown-linux-musl.tar.xz \
      -o /tmp/typst.tar.xz \
    && tar -xf /tmp/typst.tar.xz -C /tmp \
    && mv /tmp/typst-x86_64-unknown-linux-musl/typst /usr/local/bin/typst \
    && apt-get purge -y curl xz-utils \
    && apt-get autoremove -y \
    && rm -rf /var/lib/apt/lists/* /tmp/typst*
COPY --from=build /out/server /server
ENTRYPOINT ["/server"]
