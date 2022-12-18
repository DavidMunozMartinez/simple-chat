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
	w.Header().Set("Content-Type", "application/json")
	type BodyStruct = struct {
		AuthId string `json:"authId"`
	}
	var data BodyStruct
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	var user struct {
		Id   string `json:"_id" bson:"_id"`
		Name string `json:"name" bson:"name"`
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
		return
	}
	json_data, json_error := json.Marshal(&user)
	if json_error != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	w.Write([]byte(json_data))
}

func signIn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	type BodyStruct = struct {
		Id     primitive.ObjectID `json:"_id" bson:"_id"`
		Email  string             `json:"email" bson:"email"`
		AuthId string             `json:"authId" bson:"authId"`
	}
	var data BodyStruct
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	data.Id = primitive.NewObjectID()

	_, err = db_handler.Client().Collection("users").InsertOne(
		context.TODO(),
		data,
	)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	json_data, json_error := json.Marshal(&data)
	if json_error != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	} else {
		w.WriteHeader(200)
		w.Write(json_data)
	}
}

func getUserContacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

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

	type ContactsData = struct {
		Contacts         *[]primitive.ObjectID `json:"contacts" bson:"contacts"`
		ReceivedRequests *[]primitive.ObjectID `json:"receivedRequests" bson:"receivedRequests"`
		SentRequests     *[]primitive.ObjectID `json:"sentRequests" bson:"sentRequests"`
	}

	objectId, _ := primitive.ObjectIDFromHex(body.Id)
	var contactsData ContactsData
	filter := bson.M{
		"_id": objectId,
	}
	project := bson.M{
		"contacts":         1,
		"receivedRequests": 1,
		"sentRequests":     1,
	}
	options := options.FindOne().SetProjection(project)
	collection := db_handler.Client().Collection("users")
	err = collection.FindOne(context.TODO(), filter, options).Decode(&contactsData)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	type User = struct {
		Id    primitive.ObjectID `json:"_id" bson:"_id"`
		Email string             `json:"email" bson:"email"`
		Name  string             `json:"name" bson:"name"`
	}
	type ResponseStruct = struct {
		Contacts     []User               `json:"contacts"`
		Requests     []User               `json:"requests"`
		SentRequests []primitive.ObjectID `json:"sentRequests"`
	}
	var response ResponseStruct
	var userContacts []User
	var userRequests []User
	contactsFilter := bson.M{
		"_id": bson.M{
			"$in": contactsData.Contacts,
		},
	}
	requestsFilter := bson.M{
		"_id": bson.M{
			"$in": contactsData.ReceivedRequests,
		},
	}

	if contactsData.Contacts != nil {
		contacts_cursor, contacts_err := collection.Find(context.TODO(), contactsFilter)
		if contacts_err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		if err = contacts_cursor.All(context.TODO(), &userContacts); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	}

	if contactsData.ReceivedRequests != nil {
		requests_cursor, requests_err := collection.Find(context.TODO(), requestsFilter)
		if requests_err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		if err = requests_cursor.All(context.TODO(), &userRequests); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	}

	response.Contacts = userContacts
	response.Requests = userRequests
	if contactsData.SentRequests != nil {
		response.SentRequests = *contactsData.SentRequests
	}

	json_data, json_error := json.Marshal(&response)
	if json_error != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	} else {
		w.Write(json_data)
	}
}

// DEPRECATED
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
		return
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
		return
	} else {
		w.Write(json_data)
	}
}

func sendFriendRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	type BodyStruct = struct {
		From primitive.ObjectID `json:"from"` // Who sends the friend request
		To   primitive.ObjectID `json:"to"`   // Who receives the request
	}
	var body BodyStruct
	err := json.NewDecoder(r.Body).Decode(&body)
	// Can't send friend request to yourself
	if err != nil && body.From != body.To {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	// Update sender
	collection := db_handler.Client().Collection("users")
	sender := bson.M{
		"_id": body.From,
	}
	senderUpdate := bson.M{
		"$push": bson.M{
			"sentRequests": body.To,
		},
	}

	// Update receiver
	receiver := bson.M{
		"_id": body.To,
	}
	receiverUpdate := bson.M{
		"$push": bson.M{
			"receivedRequests": body.From,
		},
	}

	type User = struct {
		Type  string `json:"type" bson:"type"`
		Email string `json:"email" bson:"email"`
		Name  string `json:"name" bson:"name"`
		Id    string `json:"_id" bson:"_id"`
	}
	var user User
	// Send info about user who sends the request to the one receiving the request
	err = collection.FindOne(context.TODO(), bson.M{"_id": body.From}).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	user.Type = "request-received"
	// Notify who received the request trough WS
	if clients[body.To.Hex()] != nil {
		clients[body.To.Hex()].WriteJSON(user)
	}

	type requests = struct {
		SentRequests []primitive.ObjectID `json:"sentRequests" bson:"sentRequests"`
	}
	var results requests
	collection.FindOneAndUpdate(context.TODO(), receiver, receiverUpdate)
	collection.FindOneAndUpdate(context.TODO(), sender, senderUpdate).Decode(&results)
	// Send request data to who made the request
	json_data, json_err := json.Marshal(&results.SentRequests)
	if json_err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(200)
	w.Write(json_data)
}

func acceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	type BodyStruct = struct {
		From primitive.ObjectID `json:"from"` // Who originally sent the friend request
		To   primitive.ObjectID `json:"to"`   // Who is accepting the request
	}
	var body BodyStruct
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil && body.From != body.To {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	collection := db_handler.Client().Collection("users")

	// Update sender
	sender := bson.M{
		"_id": body.From,
	}
	senderUpdate := bson.M{
		"$pull": bson.M{
			"sentRequests": body.To,
		},
		"$push": bson.M{
			"contacts": body.To,
		},
	}

	// Update receiver
	receiver := bson.M{
		"_id": body.To,
	}
	receiverUpdate := bson.M{
		"$pull": bson.M{
			"receivedRequests": body.From,
		},
		"$push": bson.M{
			"contacts": body.From,
		},
	}

	collection.FindOneAndUpdate(context.TODO(), receiver, receiverUpdate)
	collection.FindOneAndUpdate(context.TODO(), sender, senderUpdate)
	type User = struct {
		Id    string `json:"_id" bson:"_id"`
		Email string `json:"email" bson:"email"`
		Name  string `json:"name" bson:"name"`
		Type  string `json:"type" bson:"type"`
	}
	var user User // Who sent the request
	// Return new contact info to who accepted the request
	err = collection.FindOne(context.TODO(), bson.M{"_id": body.To}).Decode(&user)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	user.Type = "request-accepted"
	// Notify original sender trough WS
	if clients[body.From.Hex()] != nil {
		err := clients[body.From.Hex()].WriteJSON(user)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	}
	w.WriteHeader(200)
	w.Write([]byte(`{"success": true}`))
}

func updateUserData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	type BodyStruct = struct {
		Id   primitive.ObjectID `json:"_id" bson:"_id"`
		Name string             `json:"name" bson:"name"`
	}
	var body BodyStruct
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	collection := db_handler.Client().Collection("users")
	filter := bson.M{
		"_id": body.Id,
	}
	setter := bson.M{
		"$set": bson.M{
			"name": body.Name,
		},
	}
	collection.FindOneAndUpdate(context.TODO(), filter, setter)

	w.WriteHeader(200)
	w.Write([]byte(`{"success": true}`))
}

func updateUserNotificationToken(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	type BodyStruct = struct {
		Id    primitive.ObjectID `json:"_id" bson:"_id"`
		Token string             `json:"token" bson:"token"`
	}
	var body BodyStruct
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	collection := db_handler.Client().Collection("users")
	filter := bson.M{
		"_id": body.Id,
	}
	setter := bson.M{
		"$set": bson.M{
			"token": body.Token,
		},
	}
	collection.FindOneAndUpdate(context.TODO(), filter, setter)
	w.WriteHeader(200)
	w.Write([]byte(`{"success": true}`))
}
