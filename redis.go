package main

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client
var ctx = context.Background()

func initRedis() {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatal("Redis URL parse failed!!", err)
	}

	rdb = redis.NewClient(opts)

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal("Redis connection failed:", err)
	}
	log.Println("connected to Redis!")
}

func cacheLatestPrice(instrument string, price float64) {
	key := "latest_price:" + instrument
	if err := rdb.Set(ctx, key, price, 0).Err(); err != nil {
		log.Println("REdis set failed:", err)
	}
}
