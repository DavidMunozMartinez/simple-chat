package api

import (
	"context"
	"encoding/json"
	"net/http"

	db_handler "chat.app/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func getUserId(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	type BodyStruct = struct {
		AuthId string `json:"authId"`
	}
	var data BodyStruct
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	var user struct {
		Id string `json:"_id" bson:"_id"`
	}
	filter := bson.M{
		"authId": data.AuthId,
	}
	err = db_handler.Client().Collection("users").FindOne(
		context.TODO(),
		filter,
	).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	} else {
		json_data, json_error := json.Marshal(&user)
		if json_error != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
		}
		w.Write([]byte(json_data))
	}
}

func signIn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	type BodyStruct = struct {
		Email  string `json:"email"`
		AuthId string `json:"authId"`
	}
	var data BodyStruct
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	_, err = db_handler.Client().Collection("users").InsertOne(
		context.TODO(),
		data,
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}
	w.WriteHeader(200)
	w.Write([]byte("success"))
}

func getUserContacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	type BodyStruct = struct {
		Id string `json:"_id"`
	}
	var body BodyStruct
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	type UserContacts = struct {
		Contacts *[]primitive.ObjectID `json:"contacts" bson:"contacts"`
	}

	objectId, _ := primitive.ObjectIDFromHex(body.Id)
	var user UserContacts
	filter := bson.M{
		"_id": objectId,
	}
	project := bson.M{
		"contacts": 1,
	}
	options := options.FindOne().SetProjection(project)
	collection := db_handler.Client().Collection("users")
	err = collection.FindOne(context.TODO(), filter, options).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	type User = struct {
		Email string `json:"email" bson:"email"`
		Id    string `json:"_id" bson:"_id"`
	}
	var users []User
	userFilter := bson.M{
		"_id": bson.M{
			"$in": user.Contacts,
		},
	}
	cursor, err := collection.Find(context.TODO(), userFilter)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if err = cursor.All(context.TODO(), &users); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	json_data, json_error := json.Marshal(&users)
	if json_error != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	} else {
		w.Write(json_data)
	}
}

func addUserContact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	type BodyStruct = struct {
		Id        primitive.ObjectID `json:"_id"`
		ContactId primitive.ObjectID `json:"contactId"`
	}
	var body BodyStruct
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil && body.Id != body.ContactId {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	collection := db_handler.Client().Collection("users")
	filter := bson.M{
		"_id": body.Id,
	}
	update := bson.M{
		"$push": bson.M{
			"contacts": body.ContactId,
		},
	}

	type Contacts = struct {
		Contacts bson.A `json:"contacts" bson:"contacts"`
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	var result Contacts
	collection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&result)
	json_data, json_error := json.Marshal(&result.Contacts)
	if json_error != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	} else {
		w.Write(json_data)
	}
}
