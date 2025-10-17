# CipherLot Node

A distributed content-addressed storage node for the CipherLot network.

## Quick Start

```bash
docker run -p 8080:8080 registry.benac.dev/cipherlot-node
```

## API Endpoints

- `GET /status` - Node status and metrics
- `GET /health` - Health check
- `PUT /blobs/{cid}` - Store content by CID
- `GET /blobs/{cid}` - Retrieve content by CID
- `PUT /manifests/{cid}` - Store JSON manifest
- `GET /manifests/{cid}` - Retrieve JSON manifest
- `POST /feeds/{author}` - Add feed entry
- `GET /feeds/{author}` - Retrieve feed entries

More documentation coming soon.