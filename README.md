# Broadcaster HTTP Proxy

Broadcaster is a small HTTP proxy designed for Kubernetes environments. It accepts incoming HTTP requests and fans them out to all Pods that belong to a headless service (DNS-based service discovery). Use it when you need to trigger identical requests across all backend Pods (for cache invalidation, control requests, etc.).

**Key features**
- Supports all HTTP methods (GET, POST, PUT, DELETE, PURGE, BAN, etc.)
- Forwards full request headers and body to each Pod
- Per-Pod request timeout and retry policy
- Uses headless service DNS to discover Pod IPs dynamically
- Includes a health endpoint that treats the service as healthy if at least one Pod is reachable

## Repository layout

```
.
├── Dockerfile
├── helm/
│   └── broadcaster/
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
├── main.go
└── README.md
```

## Quick start

Build and run locally:

```bash
go build -o broadcaster ./
BACKEND_HOST=cache-headless \
BACKEND_PORT=6081 \
./broadcaster
```

Run with Docker:

```bash
docker build -t broadcaster:latest .
docker run --rm -e BACKEND_HOST=cache-headless -e BACKEND_PORT=6081 -p 8080:8080 broadcaster:latest
```

Example request (fan-out will forward this to all backend Pods):

```bash
curl -X PURGE http://localhost:8080/some/path
```

## Configuration (environment variables)

- **`BACKEND_HOST`**: DNS name of the headless service (required). Example: `cache-headless.my-namespace.svc.cluster.local`.
- **`BACKEND_PORT`**: Port on which backend Pods listen (default: `6081`).
- **`RETRIES`**: Number of retry attempts per Pod (default: `2`). Retries are attempted per Pod when a request fails or returns an HTTP status >= 400.
- **`TIMEOUT`**: Per-request timeout for contacting each Pod, in Go duration format (default: `3s`). Example values: `500ms`, `2s`, `1m`.

All configuration is read from environment variables at process start.

## Endpoints

- `GET /healthz` — returns `200 OK` when at least one backend Pod is reachable and not returning server errors; otherwise `503 Service Unavailable`.
- `ANY /...` — main fan-out endpoint: any HTTP method and path is forwarded to all backend Pods discovered via DNS.

## Helm / Kubernetes

This repository includes a Helm chart under `helm/broadcaster` for deploying Broadcaster into a cluster.

Basic usage (from repo root):

```bash
helm install my-broadcaster ./helm/broadcaster --set image.repository=your-registry/broadcaster --set image.tag=latest
```

You should set `env.BACKEND_HOST` in the chart values to point to your headless service DNS name.

## Docker image

Build and push to a registry:

```bash
docker build -t your-registry/broadcaster:latest .
docker push your-registry/broadcaster:latest
```

## Notes and behavior

- The server listens on `:8080` by default and exposes the fan-out API and `GET /healthz`.
- Request bodies are read once from the incoming request and replayed to each backend Pod.
- The fan-out is performed concurrently across Pod IPs; any Pod that exhausts all retries is reported as a failure and will cause the broadcast call to return an error.

## Contributing

Contributions and issues are welcome — open a PR or issue with improvements.

---

