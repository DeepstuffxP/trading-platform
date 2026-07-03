# Trading Platform

A real-time order distribution system built in Go. Orders placed via a REST API are published to a NATS message bus and instantly broadcast to all connected browser clients over WebSockets. Service metrics are instrumented with Prometheus and visualized in Grafana. The entire stack runs with a single `docker-compose up` command.

## Architecture

```
Browser → POST /orders → Gin API → NATS (orders.new) → WebSocket Hub → All connected clients
                                                      ↑
                                          Prometheus scrapes /metrics
                                                      ↑
                                          Grafana reads Prometheus
```

When an order is placed:
1. The Gin REST API validates the request and generates a unique order ID
2. The order event is published to NATS on the `orders.new` subject
3. The WebSocket Hub, subscribed to `orders.new`, receives the event and broadcasts it to every connected browser tab simultaneously
4. Prometheus scrapes `/metrics` every 15 seconds — tracking total orders, active WebSocket connections, and publish failures
5. Grafana reads from Prometheus and displays live dashboards

## Stack

- **Go** — application runtime
- **Gin** — HTTP router and REST API
- **NATS** — message bus (pub/sub, decouples order intake from delivery)
- **gorilla/websocket** — WebSocket server for live browser connections
- **Prometheus** — metrics collection
- **Grafana** — metrics visualization
- **Docker Compose** — orchestrates all services

## Project Structure

```
trading-platform/
├── main.go           # Gin server, NATS connection, HTTP and WebSocket handlers
├── hub.go            # WebSocket Hub — manages connected clients with mutex protection
├── metrics.go        # Prometheus metrics definitions
├── prometheus.yml    # Prometheus scrape config
├── docker-compose.yml
├── Dockerfile
└── static/
    ├── index.html
    ├── app.js
    └── style.css
```

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Trading dashboard (frontend) |
| POST | `/orders` | Place an order — publishes to NATS |
| GET | `/ws` | WebSocket connection for live order feed |
| GET | `/metrics` | Prometheus metrics endpoint |

### POST /orders

Request body:
```json
{
  "instrument": "RELIANCE",
  "quantity": 10,
  "price": 2450.50
}
```

Response:
```json
{
  "status": "order placed",
  "order_id": "ORD-58291"
}
```

Validation: `quantity` must be at least 1, `price` must be greater than 0.

## Running Locally

**Prerequisites:** Docker and Docker Compose installed.

```bash
git clone https://github.com/DeepstuffxP/trading-platform.git
cd trading-platform
docker-compose up --build
```

Once running:

| Service | URL |
|---------|-----|
| Trading dashboard | http://localhost:8080 |
| Prometheus | http://localhost:9090 |
| Grafana | http://localhost:3000 |

Grafana login: `admin` / `admin`

To add Prometheus as a data source in Grafana: Connections → Data sources → Prometheus → URL: `http://prometheus:9090` → Save & Test.

## Metrics

Three metrics are exposed at `/metrics`:

| Metric | Type | Description |
|--------|------|-------------|
| `trading_orders_total` | Counter | Total orders placed since startup |
| `trading_ws_connections_active` | Gauge | Current number of live WebSocket connections |
| `trading_order_publish_failures_total` | Counter | Total failed NATS publish attempts |

## Concurrency Design

Each browser connecting to `/ws` runs in its own goroutine, automatically managed by Go's `net/http` internals. The Hub maintains a `map[*websocket.Conn]bool` of all connected clients — protected by a `sync.Mutex` to prevent concurrent map writes from racing goroutines, which would panic at runtime. The NATS subscriber callback also runs in a background goroutine managed by the NATS client library.

## Shutting Down

```bash
docker-compose down
```
