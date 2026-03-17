# Localspeed-CLI

Localspeed-CLI is a high-fidelity network benchmarking tool for local and wide-area networks. It performs multi-protocol speed tests combining HTTP (for bandwidth benchmarking) and WebRTC/WebSocket (for accurate jitter and packet loss measurement).

## Features

- **Multi-protocol Speedtest**: Combines HTTP, WebSocket, and WebRTC for accurate network measurement.
- **High-fidelity Benchmarking**: Provides detailed statistics including Download/Upload speeds, Idle Latency, Jitter, and Packet Loss.
- **WebRTC DataChannels**: Measures real-world UDP-like packet drops and jitter using WebRTC signaling.
- **Detailed Scoring**: Evaluates your network for Gaming, Streaming, and Browsing quality.
- **Export to CSV**: Save real-time granular test results every 500ms to a CSV file.
- **Cross-Platform**: Available for Windows, Linux, macOS, and Android (ARM64). Docker images are also provided.

---

## Server Usage

The server acts as the benchmark target, handling WebRTC signaling and HTTP streams.

### Environment Variables

You can configure the server using the following environment variables:

- `SERVER_NAME`: The name of the server displayed to the client (Default: `Localspeedtest`)
- `SERVER_PORT`: Port to listen on (Default: `7520`)
- `SERVER_ENABLE_TLS`: Set to `true` to enable HTTPS/WSS (Default: `false`)
- `SERVER_TLS_CERT`: Path to the `.crt` certificate file (if TLS is enabled)
- `SERVER_TLS_KEY`: Path to the `.key` certificate file (if TLS is enabled)

### Running via Docker

You can easily run the server using Docker:

```bash
docker run -d \
  -p 7520:7520 \
  -e SERVER_NAME="My Local Server" \
  --name localspeed-server \
  ernestoyoofi/localspeed-cli:latest
```

### Running via Binary

If you downloaded the server binary from the release page:

```bash
SERVER_NAME="My Local Server" SERVER_PORT=7520 ./server-localspeed-linux-amd64
```

---

## Client Usage

The client executes the speed test against the running Localspeed server.

### Basic Command

```bash
localspeed-cli http://192.168.1.2:7520
```
*(You can alias `localspeed-cli` to `speedtest` based on your binary mapping).*

### Available Flags

- `--unsecure`: Skip TLS verification (useful for self-signed certificates).
- `--save [PATH]`: Save the raw benchmark results to a CSV file every 500ms.
- `--sample [SIZE]`: Data size to use for testing. Available sizes: `512KB`, `1MB`, `5MB`, `10MB`, `20MB`, `50MB`, `100MB`, `200MB`, `500MB`, `1GB` (Default: `10MB`).

### Examples

**Standard Speedtest:**
```bash
speedtest http://192.168.1.2:7520
```

**Speedtest with specific sample size and saving to CSV:**
```bash
speedtest --sample 100MB --save result.csv http://192.168.1.1:7520
```

**Testing against an HTTPS/WSS server with a self-signed cert:**
```bash
speedtest --unsecure https://benchmark.mylocal.network
```

---

## Installation

Grab the latest pre-compiled binaries from the **[Releases](https://github.com/ernestoyoofi/localspeed-cli/releases)** page.

Available architectures:
- Windows (amd64, 386)
- Linux (amd64, 386, arm64, arm)
- macOS / Darwin (amd64, arm64)
- Android (arm64)

### Build from Source

Ensure you have Go 1.24+ installed:

```bash
# Clone the repository
git clone https://github.com/ernestoyoofi/localspeed-cli.git
cd localspeed-cli

# Build the Server
go build -o bin/localspeed-server ./cmd/server/main.go

# Build the Client
go build -o bin/localspeed-cli ./cmd/client/main.go
```
