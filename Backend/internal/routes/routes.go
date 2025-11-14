package routes

import (
	"time"
	"github.com/ManogyaDahal/GoType/internal/websockets"
	"github.com/ManogyaDahal/GoType/internal/auth"
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"

)    
// Sets Up the routers and defines all the routes
func SetupRouters(manager *websockets.HubManager) *gin.Engine{
	router := gin.Default()

	// CHANGED: Updated CORS to allow credentials properly
    router.Use(cors.New(cors.Config{
        AllowOrigins:     []string{"http://localhost:5173"},
        AllowMethods:     []string{"GET", "POST", "OPTIONS"},
        AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Cookie"}, // ADDED: Cookie header
        ExposeHeaders:    []string{"Content-Length", "Set-Cookie"}, // ADDED: Set-Cookie header
        AllowCredentials: true, // This is crucial for sessions
        MaxAge: 12 * time.Hour,
    }))
	//Initialize session middleware
	auth.InitSesssion(router)

	// oauth setup
	cfg := auth.InitOauth()

	// defining routes
	router.GET("/", auth.HomeHandler)
	router.GET("/login", auth.LoginHandler(cfg))
	router.GET("/api/whoamI", auth.WhoAmI)
	router.GET("/auth/google/callback", auth.CallbackHandler(cfg))
	router.GET("/logout", auth.LogoutHandler)

	router.GET("/ws", websockets.AuthenticatedWSHandler(manager))
	router.POST("/api/create-room", websockets.CreateNewRoom(manager))

	return router
}
