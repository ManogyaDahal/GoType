package main 

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/ManogyaDahal/GoType/internal/routes"
	"github.com/ManogyaDahal/GoType/internal/websockets"
)

var hubManager *websockets.HubManager

func main(){
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load environment variables.", err)
	}

	//creating single hub manager
	hubManager = websockets.NewHubManager()

	router := routes.SetupRouters(hubManager)
	router.Run(":8080")
}
