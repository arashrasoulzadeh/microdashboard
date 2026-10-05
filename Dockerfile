FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache gcc musl-dev sqlite-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -o microdashboard .

FROM alpine:3.19

RUN apk add --no-cache sqlite-libs ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/microdashboard .
COPY --from=builder /app/internal/ui/static ./internal/ui/static
COPY --from=builder /app/internal/ui/templates ./internal/ui/templates

ENV HTTP_PORT=8080 \
    LISTEN_IP=0.0.0.0 \
    DB_PATH=/data/microdashboard.db \
    ADMIN_TOKEN=admin-change-me

EXPOSE 8080

VOLUME ["/data"]

CMD ["./microdashboard"]