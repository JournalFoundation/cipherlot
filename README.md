# CipherLot Node

A distributed content-addressed storage node for the CipherLot network.

## Quick Start

```bash
# Run the node
docker run -p 8080:8080 registry.benac.dev/cipherlot-node

# Check status
curl http://localhost:8080/status
```

## Storage Model

CipherLot uses content-addressed storage where content is identified by its cryptographic hash (CID). This ensures data integrity and enables deduplication across the network.

## API Endpoints

- `GET /status` - Node status and metrics
- `GET /health` - Health check endpoint
- `PUT /blobs/{cid}` - Store binary content by CID
- `GET /blobs/{cid}` - Retrieve binary content by CID
- `PUT /manifests/{cid}` - Store JSON manifest by CID
- `GET /manifests/{cid}` - Retrieve JSON manifest by CID
- `POST /feeds/{author}` - Add entry to author's feed
- `GET /feeds/{author}` - Retrieve author's feed entries

## Configuration

- `DATA_ROOT` - Storage directory (default: `./data`)
- `PORT` - Server port (default: `8080`)

## Development

```bash
# Build from source
cd cipherlot-node
go build -o node ./cmd/node

# Run locally
./node

# Run tests
go test ./...
```

## Docker Build

```bash
# Build container
docker build -t cipherlot-node ./cipherlot-node

# Run with custom data directory
docker run -v /host/data:/data -p 8080:8080 cipherlot-node
```

## Architecture

The node implements a simple REST API over a file-based storage backend. Each piece of content is stored as a file named by its SHA-256 hash, ensuring content integrity and enabling efficient deduplication.

More documentation coming soon.