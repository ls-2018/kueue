# Build stage
ARG BUILDER_IMAGE=registry.cn-hangzhou.aliyuncs.com/ls-2018/mygo:v1.24.1
ARG BASE_IMAGE=gcr.io/distroless/static:nonroot
FROM --platform=${BUILDPLATFORM} ${BUILDER_IMAGE} AS builder

# Copy Go modules manifests and download dependencies
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build,id=go-build-cache \
    --mount=type=cache,target=/root/.gopath,id=go-build-cache \
    go mod download

# Copy the application source code
COPY . .

# Build the application
RUN --mount=type=cache,target=/root/.cache/go-build,id=go-build-cache \
    --mount=type=cache,target=/root/.gopath,id=go-build-cache \
    CGO_ENABLED=0 go build -o /kueueviz

# Runtime stage
FROM --platform=${BUILDPLATFORM} ${BASE_IMAGE}
USER 65532:65532

# Copy the built application from the builder stage
COPY --from=builder /kueueviz /

# Set environment variables
ENV PORT=8080
# Expose the application port
EXPOSE 8080

# Command to run the application
ENTRYPOINT ["/kueueviz"]


