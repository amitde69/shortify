package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

func InitRedis(address string) *redis.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatal(err)
	}
	return rdb
}

func LoadToCache(key string, value any, redis *redis.Client) error {
	ctx := context.Background()
	jsonvalue, err := json.Marshal(value)
	if err != nil {
		fmt.Println(err)
	}
	strvalue := string(jsonvalue)
	err = redis.Set(ctx, key, strvalue, time.Second*10).Err()
	if err != nil {
		log.Printf("failed setting key %s in cache: %s", key, err)
		return err
	}
	return nil
}

func RetrieveFromCache(key string, rdb *redis.Client) (string, error) {
	ctx := context.Background()
	value, err := rdb.Get(ctx, key).Result()
	if err != redis.Nil {
		return value, nil
	}
	log.Printf("cant find %s in cache", key)
	return "", errors.New("cant find key in cache")

}

func InvalidateCache(key string, rdb *redis.Client) error {
	ctx := context.Background()
	_, err := rdb.Del(ctx, key).Result()
	if err != redis.Nil {
		return nil
	}
	log.Printf("Failed deleting %s from cache", key)
	return fmt.Errorf("cant find key %s in cache", key)

}
