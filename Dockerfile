# Dockerfile Server Builder
# Builder
FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git ca-certificates

WORKDIR /builder

# 1. Copy module
COPY go.mod ./
RUN go mod tidy

# 2. Copy source code
COPY . .

# 3. Build binary
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /bin/server ./cmd/server/main.go
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -ldflags="-s -w" -o /bin/client ./cmd/client/main.go

# Stage 2: Final Image
FROM scratch

# Copy SSL certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy binary from stage builder
COPY --from=builder /bin/server /server
COPY --from=builder /bin/client /client

# Run server
CMD ["/server"]