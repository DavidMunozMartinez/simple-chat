package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	db_handler "chat.app/db"
	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
)

type Message struct {
	Timestamp int    `json:"timestamp"`
	Message   string `json:"message"`
	Id        string `json:"id"`
	To        string `json:"to"`
}

var clients = make(map[string]*websocket.Conn)
var broadcast = make(chan Message)
var origins = []string{"https://simple-chat-ui.vercel.app"}
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		var origin = r.Header.Get("Origin")
		if os.Getenv("LOCAL") == "true" {
			return true
		}

		for _, allowed := range origins {
			if origin == allowed {
				return true
			}
		}
		return false
	},
}

func InitRouterFunctions() {
	// Generic routes
	http.HandleFunc("/query-contacts", queryContacts)

	// User based routes
	http.HandleFunc("/sign-in", signIn)
	http.HandleFunc("/get-user-id", getUserId)
	http.HandleFunc("/get-user-contacts", getUserContacts)
	http.HandleFunc("/add-user-contacts", addUserContact)
	http.HandleFunc("/update-user", updateUserData)
	http.HandleFunc("/update-user-token", updateUserNotificationToken)

	http.HandleFunc("/send-friend-request", sendFriendRequest)
	http.HandleFunc("/accept-friend-request", acceptFriendRequest)

	// Message based routes
	http.HandleFunc("/save-message", saveMessage)
	http.HandleFunc("/get-messages", getMessages)

	// Websocket connections
	http.HandleFunc("/ws", handleConnections)
	go handleMessages()

	// Initialize server
	log.Println("http server started on :" + os.Getenv("PORT"))
	err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func queryContacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	type BodyStruct = struct {
		SearchTerm string `json:"searchTerm"`
	}
	var body BodyStruct
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	filter := bson.D{
		{Key: "$text", Value: bson.D{
			{Key: "$search", Value: body.SearchTerm},
		}},
	}

	type UsersDocument = struct {
		Email string `json:"email" bson:"email"`
		Id    string `json:"_id" bson:"_id"`
	}
	var results []UsersDocument
	collection := db_handler.Client().Collection("users")
	cursor, err := collection.Find(context.TODO(), filter)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if err = cursor.All(context.TODO(), &results); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
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

func handleConnections(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()
	clients[query.Get("id")] = ws
	for {
		var msg Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("error: %v", err)
			delete(clients, query.Get("id"))
			break
		}
		broadcast <- msg
	}
}

func handleMessages() {
	for {
		msg := <-broadcast
		if clients[msg.To] != nil {
			err := clients[msg.To].WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				clients[msg.To].Close()
				delete(clients, msg.To)
			}
		}
	}
}
