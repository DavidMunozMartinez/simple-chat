package main

import (
	"log"
	"net/http"
	"os"

	api "chat.app/api"
	app_notifications "chat.app/app-notifications"
	db_handler "chat.app/db"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	app_notifications.SetupFirebase()

	db_handler.MongoConnection()
	api.InitRouterFunctions()

	log.Println("http server started on :" + os.Getenv("PORT"))
	err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
