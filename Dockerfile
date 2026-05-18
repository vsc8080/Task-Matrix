# Step 1: Use the official Go image to build the app
FROM golang:1.26-alpine AS builder

# Step 2: Set the working directory inside the container
WORKDIR /app

# Step 3: Copy your code into the container
COPY . .

# Step 4: Build the Go app into a single executable named 'server'
RUN go build -o server main.go

# Step 5: Start a fresh, tiny Linux image for the final product
FROM alpine:latest
WORKDIR /root/

# Step 6: Copy the 'server' file from the builder stage
COPY --from=builder /app/server .

# Step 7: Tell Docker that the container listens on port 8080
EXPOSE 8080

# Step 8: Run the server!
CMD ["./server"]