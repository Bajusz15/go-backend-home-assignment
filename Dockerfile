FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /api ./cmd/api
RUN CGO_ENABLED=0 go build -o /seed ./cmd/seed

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u 10001 appuser

COPY --from=builder /api /api
COPY --from=builder /seed /seed
COPY migrations /migrations

USER appuser

EXPOSE 8080

ENTRYPOINT ["/api"]
