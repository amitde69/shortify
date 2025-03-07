package main

import (
	"context"
	"log"
	"shortify/config"
	"shortify/endpoints"
	"shortify/mongo"
	"shortify/redis"
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

type URL struct {
	URL       string    `bson:"url"`
	ShortURL  string    `bson:"short_url"`
	CreatedAt time.Time `bson:"created_at"`
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

			err := endpoints.URLCleanup(url.ShortURL, mongo, rdb)
			if err != nil {
				log.Printf("Failed during URL cleanup process while deleting url %s: %s", url, err)
			}
		}

		time.Sleep(time.Second * 3)
	}

}

func main() {
	config := config.Config{}
	config.InitConfig(&config)
	config.SetMongo(mongo.InitMongo("mongodb://localhost:27017"))
	config.SetRedis(redis.InitRedis("localhost:6379"))
	ExpireDocuments(config, 15)
}
