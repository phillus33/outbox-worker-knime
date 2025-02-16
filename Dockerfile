   # Use the official Golang image as a build stage
   FROM golang:1.22 AS builder

   # Set the working directory
   WORKDIR /app

   # Copy go.mod and go.sum files
   COPY go.mod go.sum ./

   # Download the dependencies
   RUN go mod download

   # Copy the source code
   COPY . .

   # Build the application
   RUN go build -o main ./cmd/example/main.go

   # Use a smaller image for the final stage
   FROM alpine:latest

   # Set the working directory
   WORKDIR /root/

   # Copy the binary from the builder stage
   COPY --from=builder /app/main .

   # Command to run the executable
   CMD ["./main"]