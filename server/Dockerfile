# Use the official Go image as a parent image
FROM golang:1.22

# Set the working directory inside the container
WORKDIR /app

# Copy the local code to the container
COPY .. .

# Download Go modules
RUN go mod download

# Build the Go app
RUN go build -o main .

# Run the compiled binary
CMD ["./main"]
