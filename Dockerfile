# defaulted ARGs need to be declared first
ARG BUILDER_IMAGE=registry.cn-hangzhou.aliyuncs.com/ls-2018/mygo:v1.24.1
ARG BASE_IMAGE=gcr.io/distroless/static:nonroot
# compilation stage for the manager binary
FROM --platform=${BUILDPLATFORM} ${BUILDER_IMAGE} AS builder
WORKDIR /workspace
# fetch dependencies first, for iterative development
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build,id=go-build-cache \
    --mount=type=cache,target=/root/.gopath,id=go-build-cache \
    go mod download
# copy the rest of the sources and build
COPY . .
ARG GIT_TAG GIT_COMMIT TARGETARCH CGO_ENABLED
RUN --mount=type=cache,target=/root/.cache/go-build,id=go-build-cache \
    --mount=type=cache,target=/root/.gopath,id=go-build-cache \
    make build GIT_TAG="${GIT_TAG}" GIT_COMMIT="${GIT_COMMIT}" GO_BUILD_ENV="GOARCH=${TARGETARCH} CGO_ENABLED=${CGO_ENABLED}"

# final image, implicitly --platform=${TARGETPLATFORM}
FROM ${BASE_IMAGE}
WORKDIR /
USER 65532:65532
ENTRYPOINT ["/manager"]
COPY --from=builder /workspace/bin/manager /manager
