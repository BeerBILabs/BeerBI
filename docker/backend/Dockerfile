# Step 1: Build the application
FROM golang:1.27-alpine AS builder
WORKDIR /src

# Install build dependencies
RUN apk add --no-cache git build-base

# Copy dependency information and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN go build -o /bin/bot ./

# Test stage: used by compose.test.yaml (`just test`), never part of the final image.
# gofmt, go vet and go test run when the container starts, so a failure makes it exit non-zero.
# -race needs CGO (build-base is installed in the builder stage) and works on musl.
FROM builder AS test
CMD ["sh", "-c", "test -z \"$(gofmt -l .)\" || { echo 'gofmt needed:'; gofmt -l .; exit 1; }; go vet ./... && go test -race -cover ./..."]

# Step 2: Create the final image
FROM alpine:3.24
RUN apk add --no-cache ca-certificates wget curl
COPY --from=builder /bin/bot /bin/bot
VOLUME /data
WORKDIR /data
EXPOSE 8080
ENTRYPOINT ["/bin/bot"]
