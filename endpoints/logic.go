package endpoints

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"shortify/redis"
	"strings"
	"time"

	redislib "github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func AlignAndHash(reqURL string) string {

	FinalURL := AlignScheme(reqURL)

	return HashIt(FinalURL)[:7]
}

func AlignScheme(reqURL string) string {
	OriginalURL := reqURL
	if strings.Contains(OriginalURL, "http") {
		OriginalURL = strings.Replace(OriginalURL, "https://", "", -1)
		OriginalURL = strings.Replace(OriginalURL, "http://", "", -1)
	}

	return "https://" + OriginalURL
}

func HashIt(URL string) string {
	h := sha256.New()
	h.Write([]byte(URL))
	bs := h.Sum(nil)
	// Convert to hex string
	hashStr := hex.EncodeToString(bs)
	return hashStr

}

func URLCleanup(url string, mdb *mongo.Client, rdb *redislib.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// delete from urls collection
	_, err := mdb.Database("shortify").Collection("urls").DeleteOne(ctx, bson.M{"short_url": url})
	if err != nil {
		log.Printf("error when deleting URL %s from mongo: %s", url, err)
		return fmt.Errorf("error when deleting URL %s from mongo: %s", url, err)
	}

	// delete from stats collection
	_, err = mdb.Database("shortify").Collection("stats").DeleteOne(ctx, bson.M{"url": url})
	if err != nil {
		return fmt.Errorf("error when deleting URL stats %s from mongo: %s", url, err)
	}

	// delete from redis cache
	err = redis.InvalidateCache(url, rdb)
	if err != nil {
		return fmt.Errorf("error when deleting URL %s from redis: %s", url, err)
	}
	return nil
}

func IncrementHits(key string, hits int, rdb *redislib.Client) error {
	mutex := redis.LockKeys(rdb)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	_, err := rdb.Incr(ctx, "stats:"+key).Result()
	cancel()
	mutex.Unlock()
	return err
}
