# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN apk add --no-cache git
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /workspace/bff ./

# Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /workspace/bff /app/bff
RUN addgroup -S app && adduser -S app -G app
USER app
EXPOSE 8080
ENTRYPOINT ["/app/bff"]
