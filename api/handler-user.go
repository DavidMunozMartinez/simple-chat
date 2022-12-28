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

type User = struct {
	Id     primitive.ObjectID `json:"_id" bson:"_id"`
	AuthId string             `json:"authId" bson:"authId"`
	Email  string             `json:"email" bson:"email"`
	Name   string             `json:"name" bson:"name"`
}

type ContactsData = struct {
	Contacts         *[]primitive.ObjectID `json:"contacts" bson:"contacts"`
	ReceivedRequests *[]primitive.ObjectID `json:"receivedRequests" bson:"receivedRequests"`
	SentRequests     *[]primitive.ObjectID `json:"sentRequests" bson:"sentRequests"`
}

var userRoutes = []AppRoute{
	{"/get-user-id", getUserId},
	{"/get-user-contacts", getUserContacts},
	{"/update", updateUser},
	{"/update-user", updateUserData},
	{"/update-user-token", updateUserNotificationToken},
	{"/send-friend-request", sendFriendRequest},
	{"/accept-friend-request", acceptFriendRequest},
}

var validUserProperties = []string{
	"token",
	"name",
}

func getUserId(w http.ResponseWriter, r *http.Request) {
	var data User
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	var user User
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
	var body User
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	var contactsData ContactsData
	filter := bson.M{
		"_id": body.Id,
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

	type Contact = struct {
		User
		LastMessage *Message `json:"lastMessage"`
	}
	type ResponseStruct = struct {
		Contacts         []Contact            `json:"contacts"`
		ReceivedRequests []User               `json:"receivedRequests"`
		SentRequests     []primitive.ObjectID `json:"sentRequests"`
	}
	var response ResponseStruct

	if contactsData.Contacts != nil {
		var contacts []Contact
		users, err := getArrayOfUserIds(contactsData.Contacts)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		for _, user := range users {
			var contact Contact
			lastMessage, err := getLastMessageBetweenUsers(user.Id, body.Id)
			if err == nil {
				contact.Id = user.Id
				contact.AuthId = user.AuthId
				contact.Email = user.Email
				contact.Name = user.Name
				contact.LastMessage = lastMessage
				contacts = append(contacts, contact)
			}
		}
		response.Contacts = contacts
	}

	if contactsData.ReceivedRequests != nil {
		requests, err := getArrayOfUserIds(contactsData.ReceivedRequests)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		response.ReceivedRequests = requests
	}

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

func sendFriendRequest(w http.ResponseWriter, r *http.Request) {
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

func updateUser(w http.ResponseWriter, r *http.Request) {
	type BodyStruct = struct {
		Id    primitive.ObjectID `json:"_id" bson:"_id"`
		Prop  string             `json:"prop"`
		Value string             `json:"value`
	}
	var body BodyStruct
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}
	if !contains(validUserProperties, body.Prop) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write(responseError("bad call"))
		return
	}

	collection := db_handler.Client().Collection("users")
	filter := bson.M{
		"_id": body.Id,
	}
	setter := bson.M{
		"$set": bson.M{
			body.Prop: body.Value,
		},
	}
	collection.FindOneAndUpdate(context.TODO(), filter, setter)

	w.WriteHeader(200)
	w.Write([]byte(`{"success": true}`))
}

// TODO: deprecate
func updateUserData(w http.ResponseWriter, r *http.Request) {
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

// TODO: deprecate
func updateUserNotificationToken(w http.ResponseWriter, r *http.Request) {
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

func getArrayOfUserIds(ids *[]primitive.ObjectID) ([]User, error) {
	filter := bson.M{
		"_id": bson.M{
			"$in": ids,
		},
	}
	var users []User
	collection := db_handler.Client().Collection("users")
	contacts_cursor, contacts_err := collection.Find(context.TODO(), filter)
	if contacts_err != nil {
		return nil, contacts_err
	}
	err := contacts_cursor.All(context.TODO(), &users)
	if err != nil {
		return nil, err
	}
	return users, nil
}
