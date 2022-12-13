package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	db_handler "chat.app/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func saveMessage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	type BodyStruct = struct {
		Message   string             `json:"message"`
		From      primitive.ObjectID `json:"from"`
		To        primitive.ObjectID `json:"to"`
		CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
		ExpireAt  time.Time          `json:"expireAt" bson:"expireAt"`
	}
	var data BodyStruct
	err := json.NewDecoder(r.Body).Decode(&data)
	// Messages will expire in a week
	data.ExpireAt = time.Now().Add(time.Hour * time.Duration(24*7))

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	_, err = db_handler.Client().Collection("messages").InsertOne(
		context.TODO(),
		data,
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	if clients[data.To.Hex()] != nil {
		err = clients[data.To.Hex()].WriteJSON(data)
		if err != nil {
			clients[data.To.Hex()].Close()
			delete(clients, data.To.Hex())
		}
	}

	w.WriteHeader(200)
	w.Write([]byte("{ \"success\": true }"))
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	type BodyStruct = struct {
		IndexId *primitive.ObjectID `json:"index"`
		Me      primitive.ObjectID  `json:"me"`
		You     primitive.ObjectID  `json:"you"`
	}
	var data BodyStruct
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	type Message = struct {
		Id        primitive.ObjectID `json:"_id" bson:"_id"`
		Message   string             `json:"message"`
		From      primitive.ObjectID `json:"from"`
		To        primitive.ObjectID `json:"to"`
		CreatedAt time.Time          `json:"createdAt"`
	}
	var messages []Message

	messagesFilter := bson.M{
		"$or": bson.A{
			bson.M{
				"$and": bson.A{
					bson.M{"from": data.Me},
					bson.M{"to": data.You},
				},
			},
			bson.M{
				"$and": bson.A{
					bson.M{"from": data.You},
					bson.M{"to": data.Me},
				},
			},
		},
	}

	filter := bson.M{}
	if data.IndexId != nil {
		indexFilter := bson.M{
			"_id": bson.M{
				"$gt": data.IndexId,
			},
		}
		filter["$and"] = bson.A{
			messagesFilter,
			indexFilter,
		}
	} else {
		filter = messagesFilter
	}

	collection := db_handler.Client().Collection("messages")
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if err = cursor.All(context.TODO(), &messages); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	json_data, json_error := json.Marshal(&messages)
	if json_error != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	} else {
		w.Write(json_data)
	}
}
