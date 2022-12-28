package app_notifications

import (
	"context"
	"log"
	"os"

	db_handler "chat.app/db"
	firebase "firebase.google.com/go"
	"firebase.google.com/go/messaging"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/api/option"
)

var app *firebase.App

func SetupFirebase() {
	godotenv.Load()
	// Raw json from:
	// https://console.firebase.google.com/project/<PROJECT_NAME>/settings/serviceaccounts/adminsdk
	sdk := os.Getenv("FIREBASE_SDK")
	opt := option.WithCredentialsJSON([]byte(sdk))

	//Firebase admin SDK initialization
	init, _ := firebase.NewApp(context.Background(), nil, opt)
	app = init
}

func Notify(to primitive.ObjectID, group primitive.ObjectID, title string, message string) {
	client, err := app.Messaging(context.TODO())
	if err != nil {
		log.Fatalf("error getting Messaging client: %v\n", err)
	}

	type User struct {
		Id    string `json:"_id" bson:"_id"`
		Token string `json:"token" bson:"token"`
	}
	var user User
	filter := bson.M{
		"_id": to,
	}
	project := bson.M{
		"token": 1,
	}
	options := options.FindOne().SetProjection(project)
	collection := db_handler.Client().Collection("users")
	err = collection.FindOne(context.TODO(), filter, options).Decode(&user)
	if err != nil {
		log.Println("Failed Notification, No User: " + err.Error())
	}

	if user.Token != "" {
		notification := &messaging.Message{
			Notification: &messaging.Notification{
				Title: title + ":",
				Body:  message,
			},
			Token: user.Token,
			Data: map[string]string{
				"tag": group.Hex(),
			},
		}
		_, err = client.Send(context.TODO(), notification)
		if err != nil {
			log.Fatalln("Failed Notification, Firebase", err)
		}
	}

}
