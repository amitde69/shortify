package main

import (
	"log"
	"shortify/endpoints"
	"shortify/mongo"
	"shortify/redis"

	"shortify/config"

	"github.com/gin-gonic/gin"
)

func main() {
	config := config.Config{}
	hitsChan := make(chan string)
	config.InitConfig(&config)
	config.SetMongo(mongo.InitMongo("mongodb://localhost:27017"))
	config.SetRedis(redis.InitRedis("localhost:6379"))
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.GET("/:url", endpoints.Direct(config, hitsChan))
	router.POST("/api/shorten", endpoints.ShortenURL(config))
	router.DELETE("/api/urls", endpoints.DeleteURL(config))
	router.GET("/api/urls", endpoints.ListURLs(config))
	router.GET("/api/stats/:url", endpoints.ListStatsByURL(config))
	router.GET("/api/stats", endpoints.ListStats(config))

	go endpoints.FlushHits(config, hitsChan)
	// go endpoints.ExpireDocuments(config, 15)

	log.Println("shortify is running on 0.0.0.0:5001...")
	router.Run("0.0.0.0:5001")
}
