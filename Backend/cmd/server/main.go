package main

import (
	"log"
	"os"

	"github.com/ManogyaDahal/GoType/internal/logger"
	"github.com/ManogyaDahal/GoType/internal/routes"
	"github.com/ManogyaDahal/GoType/internal/websockets"
	"github.com/joho/godotenv"
)

var hubManager *websockets.HubManager

func main(){
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load environment variables.", err)
	}
	
	//Initializing the logger
	logger.InitLogger(os.Getenv("ENV"))
	logger.Logger.Info("Server starting", "port", 8080, "env", os.Getenv("ENV"))

	//creating single hub manager
	hubManager = websockets.NewHubManager()

	router := routes.SetupRouters(hubManager)
	router.Run(":8080")
}
