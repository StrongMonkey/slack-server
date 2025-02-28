# Use the official Go image as the base image
FROM golang:1.21-alpine

# Set the working directory inside the container
WORKDIR /app

# Copy go.mod and go.sum files (if they exist)
COPY go.* ./

# Copy the source code
COPY *.go ./

# Build the Go app
RUN go build -o main .

# Expose the port the app runs on
EXPOSE 8088

# Command to run the executable
CMD ["./main"] 