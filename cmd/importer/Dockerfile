ARG BUILDER_IMAGE
ARG BASE_IMAGE
# Build the manager binary
FROM --platform=${BUILDPLATFORM} ${BUILDER_IMAGE} AS builder

ARG CGO_ENABLED
ARG TARGETARCH

WORKDIR /workspace

# Copy the go source
COPY . .

# Build
RUN --mount=type=cache,target=/root/.cache/go-build,id=go-build-cache \
    --mount=type=cache,target=/root/.gopath,id=go-build-cache \
    make importer-build GO_BUILD_ENV='CGO_ENABLED=${CGO_ENABLED} GOOS=linux GOARCH=${TARGETARCH}'

FROM --platform=${BUILDPLATFORM} ${BASE_IMAGE}
WORKDIR /
COPY --from=builder /workspace/bin/importer .
USER 65532:65532

ENTRYPOINT ["/importer"]
