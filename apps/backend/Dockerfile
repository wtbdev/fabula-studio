# syntax=docker/dockerfile:1
FROM golang:1.26-alpine AS builder

ENV GOPROXY=https://goproxy.cn,direct \
    HTTP_PROXY= \
    HTTPS_PROXY= \
    ALL_PROXY= \
    http_proxy= \
    https_proxy= \
    all_proxy=
WORKDIR /src
COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download

COPY apps/backend/ .
RUN CGO_ENABLED=0 GOOS=linux go build -o /fabula-server ./cmd/server/

FROM alpine:3.20

ENV HTTP_PROXY= \
    HTTPS_PROXY= \
    ALL_PROXY= \
    http_proxy= \
    https_proxy= \
    all_proxy=
WORKDIR /app
COPY --from=builder /fabula-server .
COPY test/ /app/test/

EXPOSE 8080

ENTRYPOINT ["/app/fabula-server"]
