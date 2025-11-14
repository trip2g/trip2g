FROM golang:1.24 AS builder

WORKDIR /app

# Download dependencies first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build for target architecture
# TARGETARCH is automatically set by Docker buildx (amd64, arm64, etc)
ARG TARGETARCH
RUN GOOS=linux GOARCH=${TARGETARCH} CGO_ENABLED=0 go build \
    -o /trip2g \
    -ldflags="-s -w" \
    ./cmd/server

FROM alpine:latest

# Install git and CA certificates
RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY --from=builder /trip2g /trip2g

CMD ["/trip2g"]
