# 📘 Broadcaster HTTP Proxy

**Broadcaster** is a simple HTTP proxy for Kubernetes that receives any HTTP request and broadcasts it to all Pods of a headless service.  

- Supports **any HTTP method** (GET, POST, PUT, DELETE, PURGE, BAN)  
- Forwards **full request body and headers**  
- Supports **retry and timeout per Pod**  
- DNS headless service automatically updates with new or removed Pods  
- Kubernetes-ready with **readiness/liveness probes**  

---

## 📂 Repository 
```
.
├── Dockerfile
├── helm
│   └── broadcaster
│       ├── Chart.yaml
│       ├── templates
│       └── values.yaml
├── main.go
└── README.md
```
---

## ⚡ Environment Variables

| Name           | Description                               | Default |
|----------------|-------------------------------------------|---------|
| `BACKEND_HOST` | DNS of the headless service (required)    | -       |
| `BACKEND_PORT` | Pod port                                  | 6081    |
| `RETRIES`      | Number of retry attempts per Pod          | 2       |
| `TIMEOUT`      | Timeout per request (Go duration format)  | 3s      |

---

## 🐳 Build Docker Image

```bash
docker build -t your-registry/broadcaster:latest .
docker push your-registry/broadcaster:latest
```


## 🔹 How It Works
```
        ┌─────────────┐
        │   Client    │
        └─────┬───────┘
              │ HTTP request
              ▼
        ┌─────────────┐
        │ Broadcaster │
        │   /fan-out  │
        └─────┬───────┘
              │ DNS lookup
              ▼
┌───────────────────────────────┐
│ Headless Service (ClusterIP:0)│
└───────┬─────────┬─────────────┘
        │         │
        ▼         ▼
   Pod IP1      Pod IP2 ... Pod IPn
```

	•	Broadcaster performs fan-out requests to all Pods of the headless service
	•	Forwards request body and headers
	•	Health check (/healthz) ensures at least one Pod is reachable
