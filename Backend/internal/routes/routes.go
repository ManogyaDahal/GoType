package routes

import (
	"time"
	"github.com/ManogyaDahal/GoType/internal/websockets"
	"github.com/ManogyaDahal/GoType/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"

)    
// Sets Up the routers and defines all the routes
func SetupRouters() *gin.Engine{
	router := gin.Default()
    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:5173"},
        AllowMethods:     []string{"GET", "POST", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
        ExposeHeaders:    []string{"Content-Length"},
        AllowCredentials: true,
        MaxAge: 12 * time.Hour,
    }))

	//Initialize session middleware
	auth.InitSesssion(router)

	// oauth setup
	cfg := auth.InitOauth()

	//hub manager
	manager := websockets.NewHubManager()

	// defining routes
	router.GET("/", auth.HomeHandler)
	router.GET("/login", auth.LoginHandler(cfg))
	router.GET("/api/whoamI", auth.WhoAmI)
	router.GET("/auth/google/callback", auth.CallbackHandler(cfg))
	router.GET("/logout", auth.LogoutHandler)

	router.GET("/ws", websockets.AuthenticatedWSHandler(manager))

	return router
}
