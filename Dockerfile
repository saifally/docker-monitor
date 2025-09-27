FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy source code and dependencies
COPY go.mod ./
COPY . .

# Install dependencies
RUN go mod tidy && go mod download

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o docker-monitor .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS connections
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/docker-monitor .

# Note: Running as root to access Docker socket
# In production, consider proper user/group management

# Set default environment variables
ENV MONITOR_FREQUENCY=30s
ENV COMPOSE_PROJECT_NAME=""

CMD ["./docker-monitor"]