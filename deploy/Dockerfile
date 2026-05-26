FROM golang:1.26.3-alpine AS builder

WORKDIR /app

COPY ../go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w" \
    -o /app/bin/server \
    ./cmd/server/main.go

FROM alpine:latest AS runtime

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/config /app/config
COPY --from=builder /app/api/gen /app/api/gen

RUN chown -R appuser:appgroup /app
USER appuser

FROM runtime AS server

WORKDIR /app
ENV GRPC_PORT=50051
ENV METRICS_PORT=2112
EXPOSE 50051 2112
ENTRYPOINT ["/app/server"]
