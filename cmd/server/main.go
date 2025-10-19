package main 

import (
	"log"

	"github.com/joho/godotenv"
	"github.com/ManogyaDahal/GoType/internal/routes"
)

func main(){
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load environment variables.", err)
	}

	router := routes.SetupRouters()
	router.Run(":8080")
}
