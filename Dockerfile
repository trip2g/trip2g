FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN make build-amd64

FROM scratch

WORKDIR /app

COPY --from=builder /app/tmp/amd64 /trip2g

CMD ["/trip2g"]
