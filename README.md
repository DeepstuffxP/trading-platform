# Trading Platform

A real-time order distribution system built in Go. Orders placed via a REST API are published to a NATS JetStream message bus, persisted to Postgres, cached in Redis, and instantly broadcast to all connected browser clients over WebSockets. Service metrics are instrumented with Prometheus and visualized in Grafana. The entire stack runs with a single `docker compose up` command.

## Architecture

```
Browser → POST /orders → Gin API → JetStream (orders.new) → Durable Consumer → WebSocket Hub → All connected clients
                              ↓                                                        ↑
                           Postgres                                             Redis (late-joiner cache)
                         (order history)
                              ↑
              Prometheus scrapes /metrics → Grafana
```

When an order is placed:

1. The Gin REST API validates the request and generates a unique order ID
2. The order event is published to NATS JetStream on the `orders.new` subject — persisted to disk, guaranteed delivery
3. The latest price for that instrument is written to Redis
4. The order is saved to Postgres
5. The durable JetStream consumer receives the event and the WebSocket Hub broadcasts it to every connected browser tab simultaneously
6. When a new client connects via WebSocket, Redis immediately sends them the last known price for every instrument — no waiting for the next order

If the app crashes before the consumer acknowledges a message, JetStream automatically redelivers it on restart — no orders are lost.

## Stack

- **Go** — application runtime
- **Gin** — HTTP router and REST API
- **NATS JetStream** — persistent message bus with durable consumers and guaranteed delivery
- **gorilla/websocket** — WebSocket server for live browser connections
- **Redis** — latest price cache for late-joining clients
- **Postgres** — order history persistence (raw `database/sql`, no ORM)
- **Prometheus** — metrics collection
- **Grafana** — metrics visualization
- **Docker Compose** — orchestrates all services

## Project Structure

```
trading-platform/
├── main.go           # Gin server, JetStream setup, HTTP and WebSocket handlers
├── hub.go            # WebSocket Hub — mutex-protected client map, broadcast with dead connection cleanup
├── redis.go          # Redis client init, latest price cache
├── db.go             # Postgres connection, table creation, raw SQL queries
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
| POST | `/orders` | Place an order — publishes to JetStream, writes to Redis and Postgres |
| GET | `/orders` | Fetch last 50 orders from Postgres |
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

### GET /orders

Returns last 50 orders ordered by most recent:
```json
[
  {
    "id": 1,
    "order_id": "ORD-58291",
    "instrument": "RELIANCE",
    "quantity": 10,
    "price": 2450.50,
    "created_at": "2025-01-01T10:00:00Z"
  }
]
```

## Running Locally

**Prerequisites:** Docker and Docker Compose installed.

```bash
git clone https://github.com/DeepstuffxP/trading-platform.git
cd trading-platform
docker compose up --build
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

| Metric | Type | Description |
|--------|------|-------------|
| `trading_orders_total` | Counter | Total orders placed since startup |
| `trading_ws_connections_active` | Gauge | Current number of live WebSocket connections |
| `trading_order_publish_failures_total` | Counter | Total failed NATS publish attempts |

## Concurrency Design

Each browser connecting to `/ws` runs in its own goroutine. The Hub maintains a `map[*websocket.Conn]bool` protected by a `sync.Mutex` to prevent concurrent map writes from racing goroutines. Dead connections are collected during broadcast and cleaned up after releasing the lock — avoiding a block on slow or disconnected clients. The JetStream consumer callback runs in a background goroutine managed by the NATS client library.

## Shutting Down

```bash
docker compose down
```

Note: Redis holds latest prices in memory only. Prices are lost on `docker compose down` and repopulated as new orders come in.
