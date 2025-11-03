package main 

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/ManogyaDahal/GoType/internal/routes"
	"github.com/ManogyaDahal/GoType/internal/websockets"
)

func main(){
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load environment variables.", err)
	}

	// continously checks for the incoming request
	hub := websockets.NewHub()
	go hub.Run()

	router := routes.SetupRouters()
	router.Run(":8080")
}
