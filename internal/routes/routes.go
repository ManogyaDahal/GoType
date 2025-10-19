package routes

import (
	"github.com/ManogyaDahal/GoType/internal/auth"
	"github.com/gin-gonic/gin"
)

// Sets Up the routers and defines all the routes
func SetupRouters() *gin.Engine{
	router := gin.Default()

	//Initialize session middleware
	auth.InitSesssion(router)

	// oauth setup
	cfg := auth.InitOauth()

	// defining routes
	router.GET("/", auth.HomeHandler())
	router.GET("/login", auth.LoginHandler(cfg))
	router.GET("/auth/google/callback", auth.CallbackHandler(cfg))
	router.GET("/logout", auth.LogoutHandler(cfg))

	return router
}
