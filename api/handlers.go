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

type WSMessage struct {
	Timestamp int    `json:"timestamp"`
	Message   string `json:"message"`
	Id        string `json:"id"`
	To        string `json:"to"`
}

var clients = make(map[string]*websocket.Conn)
var broadcast = make(chan WSMessage)
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

type AppRoute struct {
	Path     string
	Callback func(w http.ResponseWriter, r *http.Request)
}

var generalRoutes = []AppRoute{
	{"/query-contacts", queryContacts},
	{"/sign-in", signIn},
}

var routeGroups = [][]AppRoute{
	generalRoutes,
	userRoutes,
	messageRoutes,
}

func InitRouterFunctions() {
	// Initialize route groups
	for i := 0; i < len(routeGroups); i++ {
		group := routeGroups[i]
		for j := 0; j < len(group); j++ {
			route := group[j]
			http.HandleFunc(route.Path, func(w http.ResponseWriter, r *http.Request) {
				if validateCall(w, r) {
					route.Callback(w, r)
				} else {
					w.WriteHeader(http.StatusBadRequest)
					w.Write([]byte((`"error": "bad call"`)))
				}
			})
		}
	}

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

func validateCall(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Content-Type", "application/json")
	if os.Getenv("LOCAL") == "true" {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		return true
	}
	var origin = r.Header.Get("Origin")
	for _, allowed := range origins {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		if origin == allowed {
			return true
		}
	}
	return false
}

func contains(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

func responseError(message string) []byte {
	return []byte(`{ "error": ` + `"` + message + `"`)
}

func queryContacts(w http.ResponseWriter, r *http.Request) {
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

	filter := bson.M{
		"$text": bson.M{
			"$search":             body.SearchTerm,
			"$caseSensitive":      false,
			"$diacriticSensitive": false,
		},
	}

	var results []User
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
		var msg WSMessage
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
