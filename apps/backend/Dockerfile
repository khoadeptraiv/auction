FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY apps/backend/go.mod apps/backend/go.sum ./apps/backend/

# Copy the entire apps directory to access local dependencies (user, auction, bidding, etc.)
COPY apps/backend ./apps/backend

# Set working directory to the backend service
WORKDIR /app/apps/backend

# Download dependencies
RUN go mod download

# Build the application
RUN go build -o auction-server main.go

# Production stage
FROM alpine:latest

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/apps/backend/auction-server .

# Install ca-certificates for external API calls (Clerk, etc.)
RUN apk --no-cache add ca-certificates

# Expose the port
EXPOSE 8080

# Run the application
CMD ["./auction-server"]
