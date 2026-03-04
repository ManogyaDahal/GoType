package routes

import (
	"os"
	"time"

	"github.com/ManogyaDahal/GoType/internal/auth"
	"github.com/ManogyaDahal/GoType/internal/websockets"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Sets Up the routers and defines all the routes
func SetupRouters(manager *websockets.HubManager) *gin.Engine {
	if os.Getenv("ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.Default()

	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:5173"
	}

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{frontendURL},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Cookie"},
		ExposeHeaders:    []string{"Content-Length", "Set-Cookie"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Initialize session middleware
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
