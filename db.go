package main

import (
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

type OrderRecord struct {
	ID         int       `json:"id"`
	OrderID    string    `json:"order_id"`
	Instrument string    `json:"instrument"`
	Quantity   int       `json:"quantity"`
	Price      float64   `json:"price"`
	CreatedAt  time.Time `json:"created_at"`
}

func initDB() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=postgres password=5432 dbname=trading port=5432 sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("DB open failed:", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatal("DB connection failed:", err)
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
		id          SERIAL PRIMARY KEY,
		order_id    TEXT UNIQUE NOT NULL,
	    instrument  TEXT NOT NULL,
		quantity    INT NOT NULL,
	    price       NUMERIC(10,2) NOT NULL,
		created_at  TIMESTAMPTZ DEFAULT NOW()
		)

	`)
	if err != nil {
		log.Fatal("table creation failed:", err)
	}

	log.Println("connected to postgres")
}

func saveOrder(orderID, instrument string, quantity int, price float64) {
	_, err := db.Exec(
		`INSERT INTO orders (order_id, instrument, quantity, price) VALUES ($1, $2, $3, $4)`,
		orderID, instrument, quantity, price,
	)
	if err != nil {
		log.Println("DB insert failed:", err)
	}
}

func getRecentOrders() ([]OrderRecord, error) {
	rows, err := db.Query(
		`SELECT id, order_id, instrument, quantity, price, created_at FROM orders ORDER BY created_at DESC LIMIT 50`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []OrderRecord
	for rows.Next() {
		var r OrderRecord
		if err := rows.Scan(&r.ID, &r.OrderID, &r.Instrument, &r.Quantity, &r.Price, &r.CreatedAt); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}
