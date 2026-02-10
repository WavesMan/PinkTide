ARG DOCKER_REGISTRY=docker.io/library/
FROM ${DOCKER_REGISTRY}golang:1.22-alpine AS builder

WORKDIR /src
ARG GOPROXY
ENV GOPROXY=$GOPROXY
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags "-X PinkTide/internal/server.BuildVersion=$VERSION" -o /out/pt-server ./cmd/pt-server

ARG DOCKER_REGISTRY=docker.io/library/
FROM ${DOCKER_REGISTRY}alpine:3.20

RUN apk add --no-cache ca-certificates
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
COPY --from=builder /out/pt-server /app/pt-server
COPY --from=builder /src/ui /app/ui
RUN mkdir -p /certs && chown -R app:app /certs

USER app
ENV PT_LISTEN_ADDR=:8080
ENV PT_TLS_CERT_DIR=/certs
EXPOSE 8080 8081
VOLUME ["/certs"]
ENTRYPOINT ["/app/pt-server"]
