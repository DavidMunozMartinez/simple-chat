package main

import (
	"context"
	"log"
	"net/http"
	"os"

	db_handler "chat.app/src"
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
			db_handler.Client().Collection("messages").InsertOne(
				context.TODO(),
				bson.D{
					{"from", msg.Id},
					{"to", msg.To},
					{"message", msg.Message},
				},
			)

			err := clients[msg.To].WriteJSON(msg)
			if err != nil {
				log.Printf("error: %v", err)
				clients[msg.To].Close()
				delete(clients, msg.To)
			}
		}
	}
}
