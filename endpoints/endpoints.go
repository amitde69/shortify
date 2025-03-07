package endpoints

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"shortify/config"
	"shortify/redis"
	"time"

	"github.com/gin-gonic/gin"
	redislib "github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type shortenReq struct {
	URL string `json:"url"`
}

type shortenRes struct {
	URL string `json:"url"`
}

//	type URL struct {
//		Id       primitive.ObjectID `bson:"_id"`
//		URL      string             `bson:"url"`
//		ShortURL string             `bson:"short_url"`
//	}

type URL struct {
	URL       string    `bson:"url"`
	ShortURL  string    `bson:"short_url"`
	CreatedAt time.Time `bson:"created_at"`
}

type Stats struct {
	URL  string `bson:"url"`
	Hits int    `bson:"hits"`
}

var URLs = []URL{}

func Direct(config config.Config, hitsChan chan string) gin.HandlerFunc {
	return func(c *gin.Context) {
		url := c.Param("url")
		mongo := config.GetMongo()
		var data URL
		var err error
		var rdb *redislib.Client

		hitsChan <- url

		if config.EnableCache {
			rdb = config.GetRedis()
			data, err := fetchFromRedis(url, rdb)
			if err == nil {
				c.Redirect(http.StatusTemporaryRedirect, data.URL)
				return
			}
		}
		data, err = fetchFromMongo(url, mongo, c)
		if err != nil {
			c.IndentedJSON(http.StatusNotFound, gin.H{"message": "short url not found"})
			return
		}

		if config.EnableCache {
			err = redis.LoadToCache(data.ShortURL, data, rdb)
			if err != nil {
				log.Printf("error when loading %s to cache: %s", data.URL, err)
			}
		}

		c.Redirect(http.StatusTemporaryRedirect, data.URL)
	}
}

func ListURLs(config config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		mongo := config.GetMongo()
		cur, err := mongo.Database("shortify").Collection("urls").Find(c, bson.D{{}})
		if err != nil {
			log.Print("error when finding all urls in mongo is: ", err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			return
		}
		defer cur.Close(c)

		var data []URL
		if err := cur.All(c, &data); err != nil {
			log.Print("error when iterating on all retrived documents is: ", err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			return
		}
		c.IndentedJSON(http.StatusOK, data)
	}
}

func ShortenURL(config config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req shortenReq
		mongo := config.GetMongo()
		if err := c.BindJSON(&req); err != nil {
			return
		}

		shortened := AlignAndHash(req.URL)
		OriginalURL := req.URL

		existURL := URL{}

		cur := mongo.Database("shortify").Collection("urls").FindOne(c, bson.M{"short_url": shortened})
		err := cur.Decode(&existURL)
		// doesnt exist in mongodb
		if err != nil {
			OriginalURL = AlignScheme(OriginalURL)
			now := time.Now()
			newURL := URL{
				URL:       OriginalURL,
				ShortURL:  shortened,
				CreatedAt: now,
			}

			newStats := Stats{
				URL:  shortened,
				Hits: 0,
			}

			mongo.Database("shortify").Collection("stats").InsertOne(c, newStats)
			mongo.Database("shortify").Collection("urls").InsertOne(c, newURL)
			// URLs = append(URLs, newURL)
			c.IndentedJSON(http.StatusOK, shortenRes{URL: shortened})
		} else {
			c.IndentedJSON(http.StatusOK, shortenRes{URL: existURL.ShortURL})
		}
	}
}

func ListStats(config config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		mongo := config.GetMongo()
		cur, err := mongo.Database("shortify").Collection("stats").Find(c, bson.D{{}})
		if err != nil {
			log.Print("error when finding all stats in mongo is: ", err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			return
		}
		defer cur.Close(c)

		var data []Stats
		if err := cur.All(c, &data); err != nil {
			log.Print("error when iterating on all retrived documents is: ", err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			return
		}
		c.IndentedJSON(http.StatusOK, data)
	}
}

func DeleteURL(config config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req shortenReq
		if err := c.BindJSON(&req); err != nil {
			return
		}
		url := req.URL

		mongo := config.GetMongo()
		rdb := config.GetRedis()
		err := URLCleanup(url, mongo, rdb)
		if err != nil {
			log.Printf("Failed during URL cleanup process while deleting url %s: %s", url, err)
		}

		c.IndentedJSON(http.StatusOK, gin.H{"message": fmt.Sprintf("successfully deleted %s", url)})
	}
}

func ListStatsByURL(config config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		url := c.Param("url")
		var data Stats
		mongo := config.GetMongo()
		cur := mongo.Database("shortify").Collection("stats").FindOne(c, bson.M{"url": url})
		err := cur.Decode(&data)
		if err != nil {
			log.Print("error when finding all stats in mongo is: ", err)
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
			return
		}
		c.IndentedJSON(http.StatusOK, data)
	}
}

func fetchFromMongo(url string, mongo *mongo.Client, c *gin.Context) (URL, error) {
	var data URL
	var err error
	col := mongo.Database("shortify").Collection("urls")
	// for i := 0; i < 10; i++ {
	cur := col.FindOne(c, bson.M{"short_url": url})
	err = cur.Decode(&data)
	// }
	if err != nil {
		log.Printf("error when decoding url in mongo is: %v", err)
		return data, err
	}
	return data, nil
}

func fetchFromRedis(url string, rdb *redislib.Client) (URL, error) {
	var data URL
	var err error
	var val string
	for i := 0; i < 10; i++ {
		val, err = redis.RetrieveFromCache(url, rdb)
	}
	if err == nil {
		json.Unmarshal([]byte(val), &data)
		return data, nil
	} else {
		return data, err
	}
}
