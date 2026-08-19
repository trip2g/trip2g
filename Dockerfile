# Build frontend
FROM node:25-bookworm-slim AS frontend

# mam and its packages were unpinned: the builder resolves them from upstream
# HEAD and pulls again on every start, so rebuilding the same commit could ship
# a different frontend. Detached HEAD both fixes the revision and stops the pull
# — mol/build/ensure/git/git.ts skips a package that is not on a branch.
# To move: bump the SHA here, rebuild, and check the UI before deploying.
ARG MAM_REV=d02777f535e59ae48c52e314b112b2b3fff7c35f
ARG MOL_REV=7e252b5c3b6c3fd8e70d365555da8c4759baf1a7
ARG NODE_REV=ea57f554020e88e79955c23faf7abc856f6a5949

RUN apt update && \
  apt install -y git && \
  git clone https://github.com/hyoo-ru/mam.git /mam && \
  git -C /mam checkout --quiet $MAM_REV && \
  git clone https://github.com/hyoo-ru/mam_mol.git /mam/mol && \
  git -C /mam/mol checkout --quiet $MOL_REV && \
  git clone https://github.com/hyoo-ru/mam_node.git /mam/node && \
  git -C /mam/node checkout --quiet $NODE_REV

WORKDIR /mam

# the builder tries to resolve $input, $filter, etc vars as deps
# so we need to create empty dirs to avoid build errors
RUN mkdir fragment && \
    mkdir filter && \
    mkdir id && \
    mkdir note && \
    mkdir limit && \
    mkdir format && \
    mkdir -p version/id && \
    mkdir input

COPY ./assets/ui ./trip2g

# typescript 7 dropped convertCompilerOptionsFromJson which the mam builder uses;
# pin via overrides so the `npm update` inside `npm start` can't bump it
RUN npm pkg set overrides.typescript=6.0.3 && npm install

RUN npm start trip2g && \
    npm start trip2g/user && \
    npm start trip2g/space && \
    npm start trip2g/forms && \
    npm start trip2g/admin

# Build server binary
FROM golang:1.26 AS builder

WORKDIR /app

RUN apt update && apt install -y zip && rm -rf /var/lib/apt/lists/*

# Download dependencies first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend
COPY --from=frontend /mam/trip2g ./assets/ui

# Build for target architecture
# TARGETARCH is automatically set by Docker buildx (amd64, arm64, etc)
ARG TARGETARCH
ARG GIT_COMMIT=dev
RUN go generate ./onboarding-vault && \
    GOOS=linux GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build \
    -o /trip2g \
    -ldflags="-s -w -X main.GitCommit=${GIT_COMMIT}" \
    ./cmd/server

# Secondary binary: the fleet agent host (static, LLM-only — no interpreters here;
# code-execution roles add their own runtime, see docs/dev/fleet_packaging.md).
RUN GOOS=linux GOARCH=${TARGETARCH} CGO_ENABLED=0 \
    go build -o /fleet -ldflags="-s -w" ./cmd/fleet

# Build final image
FROM alpine:latest

# Install git and CA certificates
# git for gitapi
RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY --from=builder /trip2g /trip2g
# Secondary: run as a sidecar/second container — `docker run <image> /fleet`.
COPY --from=builder /fleet /fleet

CMD ["/trip2g"]
