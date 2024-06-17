# Use an official Go runtime as a parent image
FROM golang:latest

# Set the working directory to /go/src/app
WORKDIR /go/src/app

# Copy the local package files to the container's workspace
COPY go.mod .
COPY go.sum .

# Download and install Go module dependencies
RUN go mod download

# Copy the rest of the application source code
COPY . .

# Build the Go application
RUN go build -o main .

# Expose port 8080 to the outside world

EXPOSE 50053

# Command to run the executable
CMD ["./main"]