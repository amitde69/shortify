package endpoints

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"shortify/config"
	"shortify/redis"
	"strings"
	"sync"
	"time"

	redislib "github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Hits struct {
	urls  map[string]int
	mutex sync.Mutex
}

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

func FlushHits(config config.Config, c chan string) {
	mongo := config.GetMongo()
	hits := Hits{
		urls: make(map[string]int),
	}

	go func() {
		for url := range c {
			hits.mutex.Lock()
			hits.urls[url]++
			hits.mutex.Unlock()
		}
	}()

	for {
		hits.mutex.Lock()
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		// defer cancel()
		for url, urlHits := range hits.urls {

			filter := bson.M{"url": url}
			update := bson.M{"$inc": bson.M{"hits": urlHits}}

			// log.Printf("################# flushing to mongo %d hits", urlHits)
			_, err := mongo.Database("shortify").Collection("stats").UpdateOne(ctx, filter, update)
			if err != nil {
				log.Printf("Failed incremeting hits for %s by %d: %s", url, urlHits, err)
			}
			delete(hits.urls, url)
		}
		hits.mutex.Unlock()
		time.Sleep(time.Second * 3)
	}

}

func ExpireDocuments(config config.Config, minutes int) {
	mongo := config.GetMongo()
	rdb := config.GetRedis()

	for {
		ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		olderThan := time.Now().Add(time.Duration(-1*minutes) * time.Minute).UTC()
		filter := bson.M{"created_at": bson.M{"$lt": olderThan}}
		cur, err := mongo.Database("shortify").Collection("urls").Find(ctx, filter)
		if err != nil {
			log.Print("error when finding expired urls in mongo: ", err)
			continue
		}

		var expiredUrls []URL
		if err := cur.All(ctx, &expiredUrls); err != nil {
			log.Print("error when iterating on retrived expired urls ", err)
			continue
		}
		cur.Close(ctx)
		for _, url := range expiredUrls {
			log.Printf("found expired url %+v", url.ShortURL)

			err := URLCleanup(url.ShortURL, mongo, rdb)
			if err != nil {
				log.Printf("Failed during URL cleanup process while deleting url %s: %s", url, err)
			}
		}

		time.Sleep(time.Second * 3)
	}

}
