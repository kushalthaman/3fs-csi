# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM golang:1.22 as builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w" -o /out/3fs-csi-node ./cmd/node

# Base image with minimal tools; optionally include hf3fs binary via build arg
FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates util-linux udev procps iproute2 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /
COPY --from=builder /out/3fs-csi-node /usr/local/bin/3fs-csi-node

# Optionally copy hf3fs client if provided at build-time path /opt/3fs/bin/hf3fs_fuse_main
ARG BUNDLE_HF3FS=false
ARG HF3FS_SRC=
RUN if [ "$BUNDLE_HF3FS" = "true" ] && [ -n "$HF3FS_SRC" ]; then mkdir -p /opt/3fs/bin && cp "$HF3FS_SRC" /opt/3fs/bin/hf3fs_fuse_main && chmod +x /opt/3fs/bin/hf3fs_fuse_main; fi

ENTRYPOINT ["/usr/local/bin/3fs-csi-node"]

