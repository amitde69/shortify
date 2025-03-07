package main

import (
	"context"
	"log"
	"shortify/config"
	"shortify/mongo"
	"shortify/redis"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	redislib "github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
)

type Hits struct {
	urls  map[string]int
	mutex sync.Mutex
}

func LockKeys(rdb *redislib.Client) *redsync.Mutex {
	pool := goredis.NewPool(rdb)
	rs := redsync.New(pool)
	mutexname := "flushhits:lock"
	mutex := rs.NewMutex(mutexname)
	mutex.Lock()
	return mutex
}

func FlushHits(config config.Config) {
	mongo := config.GetMongo()
	rdb := config.GetRedis()
	for {
		mutex := LockKeys(rdb)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		var cursor uint64
		var err error
		var moreKeys []string
		cursor = 1
		var keys []string
		for cursor != 0 {
			moreKeys, cursor, err = rdb.Scan(ctx, 0, "stats:*", 10).Result()
			if err != nil {
				log.Printf("failed scanning for hits_* in redis with cursor %d: %s", cursor, err)
			}
			keys = append(keys, moreKeys...)
		}
		cancel()
		for _, key := range keys {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			value, err := rdb.Get(ctx, key).Result()
			urlHits, _ := strconv.Atoi(value)
			cancel()
			if err == redislib.Nil {
				log.Printf("cant find %s in cache", key)
				continue
			}

			url := strings.Split(key, ":")[1]

			filter := bson.M{"url": url}
			update := bson.M{"$inc": bson.M{"hits": urlHits}}

			log.Printf("Found hits for %s, flushing %d hits..", url, urlHits)
			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
			_, err = mongo.Database("shortify").Collection("stats").UpdateOne(ctx, filter, update)
			if err != nil {
				log.Printf("Failed incremeting hits for %s by %d: %s", url, urlHits, err)
				continue
			}
			cancel()
			ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
			_, err = rdb.Del(ctx, key).Result()
			cancel()
			if err == redislib.Nil {
				log.Printf("cant find %s in cache", key)
			}
		}
		mutex.Unlock()
		time.Sleep(time.Second * 3)
	}

}

func main() {
	config := config.Config{}
	config.InitConfig(&config)
	config.SetMongo(mongo.InitMongo("mongodb://localhost:27017"))
	config.SetRedis(redis.InitRedis("localhost:6379"))
	FlushHits(config)
}
