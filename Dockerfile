#  Build Stage 
FROM golang:1.25-alpine AS build

WORKDIR /app

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static Go binary for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o chat ./cmd

#  Deploy Stage 
FROM gcr.io/distroless/base-debian12

WORKDIR /app

# Copy binary from build stage
COPY --from=build /app/chat /app/chat

# Expose application port
EXPOSE 8080

# Set entrypoint
ENTRYPOINT ["/app/chat"]
