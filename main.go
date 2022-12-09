package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
)

var clients = make(map[string]*websocket.Conn)
var broadcast = make(chan Message)
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		fmt.Printf("r.Host: %v\n", r.Host)
		if os.Getenv("LOCAL") == "true" {
			return true
		}
		fmt.Printf("r.URL.Host: %v\n", r.URL.Host)
		return r.Host == "simple-chat-ui.vercel.app"
	},
}

type Message struct {
	Timestamp int    `json:"timestamp"`
	Message   string `json:"message"`
	Id        string `json:"id"`
	To        string `json:"to"`
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

func main() {
	godotenv.Load()
	fs := http.FileServer(http.Dir("public"))
	http.Handle("/", fs)
	http.HandleFunc("/ws", handleConnections)
	go handleMessages()
	log.Println("http server started on :" + os.Getenv("PORT"))
	err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
