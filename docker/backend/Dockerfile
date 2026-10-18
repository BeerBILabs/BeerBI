# Step 1: Build the application
# Pure Go (modernc.org/sqlite, CGO_ENABLED=0): the builder runs on the build
# platform and cross-compiles for the target, so multi-arch images need no QEMU
# and no C toolchain.
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
ENV CGO_ENABLED=0

# Copy dependency information and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /bin/bot ./

# Test stage: used by compose.test.yaml (`just test`), never part of the final image.
# gofmt, go vet and go test run when the container starts, so a failure makes it exit non-zero.
# Only the race detector needs CGO on Linux, so build-base is installed here and nowhere else.
FROM builder AS test
RUN apk add --no-cache build-base
ENV CGO_ENABLED=1
CMD ["sh", "-c", "test -z \"$(gofmt -l .)\" || { echo 'gofmt needed:'; gofmt -l .; exit 1; }; go vet ./... && go test -race -cover ./..."]

# Step 2: Create the final image
FROM alpine:3.24
RUN apk add --no-cache ca-certificates wget curl
COPY --from=builder /bin/bot /bin/bot
VOLUME /data
WORKDIR /data
# The binary has no default database path; keep the database on the volume.
ENV DB_PATH=/data/bot.db
EXPOSE 8080
ENTRYPOINT ["/bin/bot"]
