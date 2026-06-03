# Use Go 1.25 to match the project version
FROM golang:1.25-alpine

WORKDIR /app

# Copy go.mod and go.sum from the parent directory
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy the entire source code
COPY . .

# Build the test executable
RUN go build -o /terminaldetection ./tests/terminaldetection

# Set the executable as the entrypoint
ENTRYPOINT ["/terminaldetection"]
