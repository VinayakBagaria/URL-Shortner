package database

import (
	"github.com/go-redis/redis"
	"log"
	"os"
)

func CreateClient(dbNo int) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("DB_ADDR"),
		Password: os.Getenv("DB_PASS"),
		DB:       dbNo,
	})
	if _, err := rdb.Ping().Result(); err != nil {
		log.Fatalln("Could not connect to the database")
	}
	return rdb
}
