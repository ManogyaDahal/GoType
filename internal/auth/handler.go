package auth

import (
	"context"
	"net/http"
	"log"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

//Handler the Home route
func HomeHandler() gin.HandlerFunc {
	return func(c *gin.Context){
		c.JSON(http.StatusOK, gin.H{"message":"Welcome to goType"})
	}
}

// handles the /login route
func LoginHandler( cfg *oauth2.Config) gin.HandlerFunc {
	return func(c *gin.Context){
		//Loading and setting session state
		session := sessions.Default(c)
		state := GenerateState()
		log.Printf("Generated state: %s\n",state)
		session.Set("oauth_state", state)
		session.Save()

		// retrieving and redirecting the url recieved for concent page
		url := cfg.AuthCodeURL(state)		
		c.Redirect(http.StatusFound, url)
	}
}

//Callback Handler handles
func CallbackHandler( cfg *oauth2.Config) gin.HandlerFunc {
	return func(c *gin.Context){
		session := sessions.Default(c)
		stateInSession := session.Get("oauth_state")
		stateInUrl := c.Query("state")

		log.Printf("stateInsession: %s\n",stateInSession)
		log.Printf("stateInUrl: %s\n", stateInUrl)
		// validating the state
		if stateInSession != stateInUrl {
			c.JSON(http.StatusBadRequest, 
				   gin.H{ "error": "Invalid Oauth state"})
			return 
		}

		code := c.Query("code")
		if code == ""{
			c.JSON(http.StatusBadRequest, gin.H{"error":"got empty code"})
			return 
		}

		tok, err := cfg.Exchange(context.Background(), code)
		if err != nil {
			c.JSON(http.StatusInternalServerError, 
			gin.H{"error": "Error while exchanging tokens"})
			return 
		}
		
		//storing the access token in session
		session.Set("access_token", tok.AccessToken)
		if err := session.Save(); err != nil {
			log.Printf("[SESSION] Failed to save: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "session save failed"})
			return
		}
		log.Println("[CALLBACK] Redirecting to home...")
		c.Redirect(http.StatusFound, "/")
	}
}

// handles the /logout route
func LogoutHandler(cfg *oauth2.Config) gin.HandlerFunc {
	return func(c *gin.Context){
		session := sessions.Default(c)
		session.Clear()
		session.Save()
		c.Redirect(http.StatusOK, "/")
	}
}
