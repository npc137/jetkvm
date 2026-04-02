FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# CGO_ENABLED=0 is intentional: modernc.org/sqlite is a pure-Go SQLite
# implementation and does not require CGO, enabling a fully static binary.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /management-backend .

# ── Runtime image ─────────────────────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app
COPY --from=builder /management-backend .

# Default database location (mount a volume here in production).
RUN mkdir -p /data

EXPOSE 8080

ENTRYPOINT ["/app/management-backend"]
