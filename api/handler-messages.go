package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	app_notifications "chat.app/app-notifications"
	db_handler "chat.app/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Message = struct {
	Id        primitive.ObjectID `json:"_id" bson:"_id"`
	Message   string             `json:"message"`
	From      primitive.ObjectID `json:"from"`
	To        primitive.ObjectID `json:"to"`
	CreatedAt time.Time          `json:"createdAt"`
}

var messageRoutes = []AppRoute{
	{"/save-message", saveMessage},
	{"/get-messages", getMessages},
}

func saveMessage(w http.ResponseWriter, r *http.Request) {
	type BodyStruct = struct {
		Id        primitive.ObjectID `json:"_id" bson:"_id"`
		Message   string             `json:"message"`
		Title     string             `json:"title"`
		From      primitive.ObjectID `json:"from"`
		To        primitive.ObjectID `json:"to"`
		CreatedAt time.Time          `json:"createdAt" bson:"createdAt"`
		ExpireAt  time.Time          `json:"expireAt" bson:"expireAt"`
	}
	var data BodyStruct
	err := json.NewDecoder(r.Body).Decode(&data)
	// Messages will expire in a week
	data.ExpireAt = time.Now().Add(time.Hour * time.Duration(24*7))
	data.Id = primitive.NewObjectID()

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseError("Bad request"))
		return
	}

	_, err = db_handler.Client().Collection("messages").InsertOne(
		context.TODO(),
		data,
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseError("Unable to save"))
		return
	}

	// Messaging priority
	// 1.- respond OK to sender
	// 2.- Send WS event to receiver
	// 3.- Send Push Notification to receiver
	json_data, json_error := json.Marshal(&data)
	if json_error != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseError("Bad data"))
		return
	}

	if clients[data.To.Hex()] != nil {
		err = clients[data.To.Hex()].WriteJSON(data)
		if err != nil {
			clients[data.To.Hex()].Close()
			delete(clients, data.To.Hex())
		}
	}
	app_notifications.Notify(data.To, data.From, data.Title, data.Message)

	w.WriteHeader(200)
	w.Write(json_data)
}

func getMessages(w http.ResponseWriter, r *http.Request) {
	type BodyStruct = struct {
		RetrieveBeforeIndex bool                `json:"retrieveBeforeIndex"`
		IndexId             *primitive.ObjectID `json:"index"`
		Me                  primitive.ObjectID  `json:"me"`
		You                 primitive.ObjectID  `json:"you"`
	}
	var data BodyStruct
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseError("Bad request"))
		return
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
		condition := "$gt"
		if data.RetrieveBeforeIndex {
			condition = "$lt"
		}
		indexFilter := bson.M{
			"_id": bson.M{
				condition: data.IndexId,
			},
		}

		filter["$and"] = bson.A{
			messagesFilter,
			indexFilter,
		}
	} else {
		filter = messagesFilter
	}

	opts := options.Find().SetLimit(5)
	opts.SetSort(bson.M{"_id": -1})
	collection := db_handler.Client().Collection("messages")
	cursor, err := collection.Find(context.TODO(), filter, opts)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseError("Unable to get"))
		return
	}

	if err = cursor.All(context.TODO(), &messages); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseError("Unable to get"))
		return
	}

	json_data, json_error := json.Marshal(&messages)
	if json_error != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseError("Bad data"))
		return
	} else {
		w.Write(json_data)
	}
}

func getLastMessageBetweenUsers(id1 primitive.ObjectID, id2 primitive.ObjectID) (*Message, error) {
	filter := bson.M{
		"$or": bson.A{
			bson.M{
				"$and": bson.A{
					bson.M{"from": id1},
					bson.M{"to": id2},
				},
			},
			bson.M{
				"$and": bson.A{
					bson.M{"from": id2},
					bson.M{"to": id1},
				},
			},
		},
	}
	var message Message
	collection := db_handler.Client().Collection("messages")
	opts := options.FindOne().SetSort(bson.M{"createdAt": -1})
	err := collection.FindOne(context.TODO(), filter, opts).Decode(&message)
	if err != nil {
		message.Id = primitive.NewObjectID()
		message.CreatedAt = time.Date(1995, 0, 0, 0, 0, 0, 0, time.Local)
		message.From = primitive.NewObjectID()
		message.To = primitive.NewObjectID()
		message.Message = ""

		return &message, nil
	}
	return &message, nil
}
