# Build frontend
FROM node:25-bookworm-slim AS frontend

RUN apt update && \
  apt install -y git && \
  git clone https://github.com/hyoo-ru/mam.git /mam

WORKDIR /mam

# the builder tries to resolve $input, $filter, etc vars as deps
# so we need to create empty dirs to avoid build errors
RUN mkdir fragment && \
    mkdir filter && \
    mkdir id && \
    mkdir limit && \
    mkdir format && \
    mkdir input

COPY ./assets/ui/externaldeps ./trip2g/externaldeps/

RUN npm start trip2g/externaldeps

COPY ./assets/ui ./trip2g

RUN npm start trip2g && \
    npm start trip2g/user && \
    npm start trip2g/admin

# Build server binary
FROM golang:1.24 AS builder

WORKDIR /app

# Download dependencies first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend
COPY --from=frontend /mam/trip2g/- ./assets/ui/-
COPY --from=frontend /mam/trip2g/admin/- ./assets/ui/admin/-/
COPY --from=frontend /mam/trip2g/user/- ./assets/ui/user/-/

# Build for target architecture
# TARGETARCH is automatically set by Docker buildx (amd64, arm64, etc)
ARG TARGETARCH
RUN GOOS=linux GOARCH=${TARGETARCH} CGO_ENABLED=0 go build \
    -o /trip2g \
    -ldflags="-s -w" \
    ./cmd/server

# Build final image
FROM alpine:latest

# Install git and CA certificates
# git for gitapi
RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY --from=builder /trip2g /trip2g

CMD ["/trip2g"]
