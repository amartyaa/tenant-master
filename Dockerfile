# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /workspace

# Copy go mod files
COPY go.mod go.mod
COPY go.sum go.sum

# Cache go modules
RUN go mod download

# Copy source code
COPY cmd/ cmd/
COPY api/ api/
COPY internal/ internal/

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags="-w -s" -o manager cmd/main.go

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

WORKDIR /

# Copy binary from builder
COPY --from=builder /workspace/manager .

# Create non-root user
RUN addgroup -S tenant-master && adduser -S tenant-master -G tenant-master

USER tenant-master

ENTRYPOINT ["/manager"]
