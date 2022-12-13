package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	api "chat.app/api"
	db_handler "chat.app/db"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
)

func main() {
	godotenv.Load()
	db_handler.MongoConnection()
	api.InitRouterFunctions()
	log.Println("http server started on :" + os.Getenv("PORT"))
	err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func queryContacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	type BodyStruct = struct {
		SearchTerm string `json:"searchTerm"`
	}
	var body BodyStruct
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	filter := bson.D{
		{Key: "$text", Value: bson.D{
			{Key: "$search", Value: body.SearchTerm},
		}},
	}

	type UsersDocument = struct {
		Email string `json:"email" bson:"email"`
		Id    string `json:"id" bson:"id"`
	}
	var results []UsersDocument
	collection := db_handler.Client().Collection("users")
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	if err = cursor.All(context.TODO(), &results); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	if len(results) > 0 {
		json_data, json_error := json.Marshal(&results)
		if json_error != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		} else {
			w.Write(json_data)
		}
	} else {
		w.Write([]byte("[]"))
	}
}
