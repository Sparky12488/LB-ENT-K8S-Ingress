# Stage 1: Build
FROM golang:1.24-alpine AS builder
WORKDIR /app
# Pre-copy modules for caching
COPY go.mod go.sum ./
RUN go mod download
# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o manager ./cmd/manager/main.go

# Stage 2: Final image
FROM alpine:3.18
WORKDIR /
COPY --from=builder /app/manager .
# Important: Appliance communication usually needs CA certs
RUN apk add --no-cache ca-certificates
ENTRYPOINT ["./manager"]