package main

import (
	"context"
	"encoding/json"
	"net/http"

	db_handler "chat.app/src"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func signIn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	type BodyStruct = struct {
		Email string `json:"email"`
		Id    string `json:"id"`
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
		Id string `json:"id"`
	}
	var body BodyStruct
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	type Contacts = struct {
		Contacts bson.A `json:"contacts" bson:"contacts"`
	}

	var contacts Contacts
	filter := bson.M{
		"id": body.Id,
	}
	project := bson.M{
		"contacts": 1,
	}
	options := options.FindOne().SetProjection(project)
	collection := db_handler.Client().Collection("users")
	err = collection.FindOne(context.TODO(), filter, options).Decode(&contacts)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	type Users = struct {
		Email string `json:"email" bson:"email"`
		Id    string `json:"id" bson:"id"`
	}
	var users []Users
	userFilter := bson.M{
		"_id": bson.M{
			"$in": contacts.Contacts,
		},
	}
	cursor, err := collection.Find(context.TODO(), userFilter)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	if err = cursor.All(context.TODO(), &users); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	json_data, json_error := json.Marshal(&users)
	if json_error != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	} else {
		w.Write(json_data)
	}
}

func addUserContact(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	type BodyStruct = struct {
		Id        string `json:"id"`
		ContactId string `json:"contactId"`
	}
	var body BodyStruct
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
	}

	collection := db_handler.Client().Collection("users")
	filter := bson.D{
		{"id", body.Id},
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
