# syntax=docker/dockerfile:1

FROM golang:1.25.5-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG TARGETOS=linux
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
  go build -trimpath -ldflags="-s -w" -o /out/lms-cn-api ./cmd/api

FROM alpine:3.22 AS runner
RUN apk add --no-cache ca-certificates tzdata \
  && addgroup -S -g 10001 lms \
  && adduser -S -D -H -u 10001 -G lms lms
WORKDIR /app
COPY --from=builder /out/lms-cn-api ./lms-cn-api
COPY --from=builder --chown=lms:lms /src/migrations ./migrations
ENV APP_ENV=production
ENV PORT=8080
ENV API_PREFIX=/api/v1
ENV MIGRATIONS_PATH=/app/migrations
ENV SEED_DEMO_DATA=false
USER lms
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -q -O - "http://127.0.0.1:${PORT}${API_PREFIX}/health/live" > /dev/null || exit 1
ENTRYPOINT ["/app/lms-cn-api"]
