# syntax=docker/dockerfile:1

# --- Build stage -------------------------------------------------------------
FROM golang:1.26-alpine AS build

WORKDIR /src

# Cache dependencies first.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build both binaries statically so they run on a minimal base image.
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/worker ./cmd/worker

# --- Runtime stage -----------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates && adduser -D -u 10001 pulse

COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/worker /usr/local/bin/worker

USER pulse

# Default to the API; docker-compose overrides the command for the worker.
ENTRYPOINT ["api"]
