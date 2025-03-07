package config

import (
	"log"
	"os"
	"strconv"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
)

type Config struct {
	mongo       *mongo.Client
	redis       *redis.Client
	EnableCache bool
}

func (c *Config) InitConfig(config *Config) {
	var err error
	config.EnableCache, err = strconv.ParseBool(os.Getenv("ENABLE_CACHE"))
	if err != nil {
		log.Println("Failed parsing ENABLE_CACHE config env var")
		config.EnableCache = false
	}
	log.Println("ENABLE_CACHE ==", config.EnableCache)
}

func (c *Config) GetMongo() *mongo.Client {
	return c.mongo
}
func (c *Config) SetMongo(mongo *mongo.Client) {
	c.mongo = mongo
}

func (c *Config) GetRedis() *redis.Client {
	return c.redis
}
func (c *Config) SetRedis(redis *redis.Client) {
	c.redis = redis
}
