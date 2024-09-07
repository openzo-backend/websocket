# Stage 1: Build the Go binary using golang:latest
FROM golang:latest as builder

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod .
COPY go.sum .

# Download and install Go module dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o main .

# Stage 2: Use a smaller Ubuntu base image for runtime
# FROM ubuntu:22.04
FROM ubuntu:jammy

# Add the security repository for glibc version 2.35
RUN echo "deb http://security.ubuntu.com/ubuntu jammy-security main" | tee -a /etc/apt/sources.list

# Install necessary dependencies (like libc6 and ca-certificates)
RUN apt-get update && apt-get install --no-install-recommends -y \
    libc6=2.35-0ubuntu3.8 \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Set the working directory in the runtime image
WORKDIR /app

# Copy the built binary from the builder stage
COPY --from=builder /app/main .

# Make sure the binary has executable permissions
RUN chmod +x /app/main

# Expose necessary ports
EXPOSE 8081
EXPOSE 50052

# Command to run the application
CMD ["./main"]
