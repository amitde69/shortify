package main

import (
	"shortify/config"
	"shortify/endpoints"
	"shortify/mongo"
	"shortify/redis"
)

func main() {
	config := config.Config{}
	config.InitConfig(&config)
	config.SetMongo(mongo.InitMongo("mongodb://localhost:27017"))
	config.SetRedis(redis.InitRedis("localhost:6379"))
	endpoints.ExpireDocuments(config, 15)
}
