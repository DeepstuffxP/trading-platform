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

	// subscribe to orders.new
	_, err = nc.Subscribe("orders.new", func(msg *nats.Msg) {
		log.Println("received from NATS, broadcasting:", string(msg.Data))
		hub.broadcast(string(msg.Data))
	})
	if err != nil {
		log.Fatal("NATS subscribe failed:", err)
	}

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
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				break
			}
		}
	})

	// REST endpoint
	r.POST("/orders", func(c *gin.Context) {
		var order Order
		if err := c.ShouldBindJSON(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Submission rejected: Quantity must be at least 1, and Price must be greater than 0."})
			return
		}

		//id
		uniqueID := fmt.Sprintf("ORD-%d", time.Now().Unix()%100000)

		msg := fmt.Sprintf("[%s] %s | QTY: %d | PRICE: ₹%.2f", uniqueID, order.Instrument, order.Quantity, order.Price)

		if err := nc.Publish("orders.new", []byte(msg)); err != nil {
			orderPublishFailures.Inc()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish"})
			return
		}
		ordersPlaced.Inc()

		log.Printf("order published: %s", uniqueID)
		c.JSON(http.StatusOK, gin.H{"status": "order placed", "order_id": uniqueID})
	})

	log.Println("server listening on :8080")
	r.Run(":8080")
}
