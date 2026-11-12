# syntax=docker/dockerfile:1

# ---- build ----------------------------------------------------------------
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Dependencies first so they stay cached while application code changes.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- runtime --------------------------------------------------------------
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /out/api /app/api
COPY migrations/ /app/migrations/

# Run unprivileged.
RUN addgroup -S -g 10001 app && adduser -S -u 10001 -G app app
USER app

EXPOSE 8081

ENTRYPOINT ["/app/api"]
