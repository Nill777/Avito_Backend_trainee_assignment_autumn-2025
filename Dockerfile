FROM golang:1.23-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o reviewer ./cmd/app
FROM alpine:latest
RUN adduser -D appuser
USER appuser
WORKDIR /home/appuser

COPY --from=builder /app/reviewer /usr/local/bin/reviewer
COPY openapi.yaml . 

ENV PORT=8080
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/reviewer"]