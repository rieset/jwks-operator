# ----
# builder stage: This stage builds the JWKS Operator binary.
# NOTE: Using Google Container Registry mirror to avoid Docker Hub rate limits
# ----
FROM mirror.gcr.io/library/golang:1.24-alpine AS builder

# Set working directory
WORKDIR /workspace

# Install build dependencies
RUN apk add --no-cache git make

# Copy go.mod and go.sum files first for better layer caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate deepcopy code before building
# Install controller-gen and ensure it's in PATH
ENV GOBIN=/go/bin
ENV PATH=$PATH:/go/bin
RUN go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0 || \
    go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.13.0 || \
    go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
RUN controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager cmd/manager/main.go

# ----
# production stage: This is the final image used to run the JWKS Operator in production.
# It contains only the compiled Go binary, minimal runtime dependencies, and configuration.
# The image uses a non-root user, includes CA certificates and timezone data.
# NOTE: Using Google Container Registry mirror to avoid Docker Hub rate limits
# ----
FROM mirror.gcr.io/library/alpine:3.21 AS production

# Add necessary packages
RUN apk --no-cache add ca-certificates tzdata

# Set working directory
WORKDIR /

# Copy the binary from the builder stage
COPY --from=builder /workspace/manager /manager

# Copy the configuration file
COPY --from=builder /workspace/config.yaml /config.yaml

# Create a non-root user with numeric UID for Kubernetes runAsNonRoot
RUN adduser -D -u 65532 -g '' appuser && \
    chown -R 65532:65532 /manager /config.yaml && \
    chmod 644 /config.yaml && \
    chmod 755 /manager
USER 65532

# Command to run the application
ENTRYPOINT ["/manager"]

