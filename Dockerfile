# Build stage for Go binaries
FROM golang:alpine AS builder

WORKDIR /build

# Copy the go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the entire project
COPY . .

# Provide a build argument to determine which component to build 
# e.g., cmd/api, cmd/broker, cmd/worker, test_app
ARG COMPONENT_PATH=cmd/api

# Build the specific component
RUN cd ${COMPONENT_PATH} && go build -o /build/bin/app .

# Final stage
FROM alpine:latest

WORKDIR /app
# Copy the built binary
COPY --from=builder /build/bin/app /app/app

# Expose ports that might be used (API uses 8080)
EXPOSE 8080

ENTRYPOINT ["/app/app"]
