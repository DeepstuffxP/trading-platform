package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ordersPlaced = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trading_orders_total",
		Help: "Total number of orders placed",
	})

	activeConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "trading_ws_connections_active",
		Help: "Number of active WebSocket connections",
	})

	orderPublishFailures = promauto.NewCounter(prometheus.CounterOpts{
		Name: "trading_order_publish_failures_total",
		Help: "Total number of failed NATS publishes",
	})
)
