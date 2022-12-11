package db_handler

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	db *mongo.Client
)

func MongoConnection() {
	var mongouri = "mongodb+srv://admin:" + os.Getenv("MONGO_PASSWORD") + "@cluster0.ceyoj.gcp.mongodb.net/?retryWrites=true&w=majority"
	fmt.Println(mongouri)
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongouri))
	if err != nil {
		panic(err)
	}

	db = client
}

func Client() *mongo.Database {
	return db.Database("simple-chat")
}
