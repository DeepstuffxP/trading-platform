package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Order struct {
	Instrument string  `json:"instrument" binding:"required"`
	Quantity   int     `json:"quantity" binding:"required,gte=1"`
	Price      float64 `json:"price" binding:"required,gt=0"`
}

func main() {
	hub := newHub()

	// nats + jetstrm
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatal("NATS connection failed:", err)
	}
	defer nc.Close()
	log.Println("connected to NATS")

	js, err := nc.JetStream()
	if err != nil {
		log.Fatal("JetStream context failed:", err)
	}

	// create stream
	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "ORDERS",
		Subjects: []string{"orders.new"},
	})
	if err != nil && err != nats.ErrStreamNameAlreadyInUse {
		log.Fatal("stream creation failed:", err)
	}
	log.Println("JetStream stream ready")

	// durable consumer — survives restarts, replays unacked messages
	_, err = js.Subscribe("orders.new", func(msg *nats.Msg) {
		log.Println("received from JetStream, broadcasting:", string(msg.Data))
		hub.broadcast(string(msg.Data))
		msg.Ack()
	}, nats.Durable("ws-broadcaster"), nats.ManualAck())
	if err != nil {
		log.Fatal("JetStream subscribe failed:", err)
	}

	// Redis
	initRedis()

	// Postgres
	initDB()

	//  HTTP
	r := gin.Default()

	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// websocket endpoint
	r.GET("/ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Println("upgrade failed:", err)
			return
		}
		hub.addClient(conn)
		defer func() {
			hub.removeClient(conn)
			conn.Close()
		}()

		//  late - joiner
		keys, err := rdb.Keys(ctx, "latest_price:*").Result()
		if err == nil {
			for _, key := range keys {
				val, err := rdb.Get(ctx, key).Result()
				if err == nil {
					instrument := key[len("latest_price:"):]
					conn.WriteMessage(websocket.TextMessage, []byte(
						fmt.Sprintf("[LATEST] %s: ₹%s", instrument, val),
					))
				}
			}
		}

		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	})

	// POST
	r.POST("/orders", func(c *gin.Context) {
		var order Order
		if err := c.ShouldBindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Submission rejected: Quantity must be at least 1, and Price must be greater than 0."})
			return
		}

		uniqueID := fmt.Sprintf("ORD-%d", time.Now().UnixNano()%100000)
		msg := fmt.Sprintf("[%s] %s | QTY: %d | PRICE: ₹%.2f", uniqueID, order.Instrument, order.Quantity, order.Price)

		// publish
		if _, err := js.Publish("orders.new", []byte(msg)); err != nil {
			orderPublishFailures.Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish"})
			return
		}
		ordersPlaced.Inc()

		// cache latest price
		cacheLatestPrice(order.Instrument, order.Price)

		//save
		saveOrder(uniqueID, order.Instrument, order.Quantity, order.Price)

		log.Printf("order published: %s", uniqueID)
		c.JSON(http.StatusOK, gin.H{"status": "order placed", "order_id": uniqueID})
	})

	// GET
	r.GET("/orders", func(c *gin.Context) {
		records, err := getRecentOrders()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch orders"})
			return
		}
		c.JSON(http.StatusOK, records)
	})

	log.Println("server listening on :8080")
	r.Run(":8080")
}
