package auth

import (
	"encoding/base64"
	"crypto/rand"
	"net/http"
	"os"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)


func InitSesssion(router *gin.Engine){
	secretKey := os.Getenv("SESSION_SECRET")
	store := cookie.NewStore([]byte(secretKey))
	store.Options( sessions.Options{
		Path: "/", 
		MaxAge: 7*86400,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure: false,
	})
	
	router.Use(sessions.Sessions("session", store))
}

// Generates random state
func GenerateState() string {
	data := make([]byte, 16)
	rand.Read(data)
 	return base64.URLEncoding.EncodeToString(data)
}
