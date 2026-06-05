# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder

ARG HTTP_PROXY
ARG HTTPS_PROXY
ENV HTTP_PROXY=${HTTP_PROXY} HTTPS_PROXY=${HTTPS_PROXY}

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download

COPY apps/backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /fabula-server ./cmd/server/

FROM alpine:3.20

ARG HTTP_PROXY
ARG HTTPS_PROXY
ENV HTTP_PROXY=${HTTP_PROXY} HTTPS_PROXY=${HTTPS_PROXY}

RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /fabula-server .
COPY test/ /app/test/

EXPOSE 8080

ENTRYPOINT ["/app/fabula-server"]
