package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

func InitMongo(uri string) *mongo.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	client, _ := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	_ = client.Ping(ctx, readpref.Primary())
	return client
}
