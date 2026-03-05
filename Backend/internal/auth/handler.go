package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/ManogyaDahal/GoType/internal/logger"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

// frontendURL returns the configured frontend origin, defaulting to localhost for dev.
func frontendURL() string {
	u := os.Getenv("FRONTEND_URL")
	if u == "" {
		return "http://localhost:5173"
	}
	return u
}

// Handler the Home route
func HomeHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Welcome to goType"})
}

// handles the /login route
func LoginHandler(cfg *oauth2.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate a self-validating HMAC-signed state token.
		// No need to store it in the session — the callback will
		// verify the signature instead.
		state := GenerateState()

		// retrieving and redirecting the url recieved for concent page
		url := cfg.AuthCodeURL(state)
		c.Redirect(http.StatusFound, url)
	}
}

// Callback Handler handles
func CallbackHandler(cfg *oauth2.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		stateInUrl := c.Query("state")

		// Validate the HMAC-signed state token.
		// This does NOT require reading from the session cookie,
		// so it works even when cookies aren't forwarded through a proxy.
		if !ValidateState(stateInUrl) {
			c.JSON(http.StatusBadRequest,
				gin.H{"error": "Invalid Oauth state"})
			return
		}

		code := c.Query("code")
		if code == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "got empty code"})
			return
		}

		tok, err := cfg.Exchange(context.Background(), code)
		if err != nil {
			c.JSON(http.StatusInternalServerError,
				gin.H{"error": "Error while exchanging tokens"})
			return
		}

		client := cfg.Client(context.Background(), tok)
		resp, err := client.Get(UserInfo)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user info"})
			return
		}
		defer resp.Body.Close()

		var userInfo struct {
			ID            string `json:"id"`
			Email         string `json:"email"`
			VerifiedEmail bool   `json:"verified_email"`
			Name          string `json:"name"`
			GivenName     string `json:"given_name"`
			FamilyName    string `json:"family_name"`
			Picture       string `json:"picture"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid user info"})
			return
		}

		// setting values in session
		session := sessions.Default(c)
		session.Set("Name", userInfo.Name)
		session.Set("Email", userInfo.Email)
		session.Set("VerifiedEmail", userInfo.VerifiedEmail)
		session.Set("Picture", userInfo.Picture)

		if err := session.Save(); err != nil {
			logger.Logger.Error("[SESSION] Failed to save session",
				"error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "session save failed"})
			return
		}
		c.Redirect(http.StatusFound, frontendURL()+"/")
	}
}

// handles the /logout route
func LogoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Options(sessions.Options{
		MaxAge: -1,
		Path:   "/",
	})
	if err := session.Save(); err != nil {
		logger.Logger.Error("[SESSION] Failed to clear session",
			"error", err)
	}
	c.Redirect(http.StatusFound, frontendURL()+"/")
}

// WhoAmI returns the current user and a short-lived WebSocket auth token.
// The ws_token lets the frontend authenticate WebSocket connections that go
// directly to Render, bypassing the Vercel proxy where the session lives.
func WhoAmI(c *gin.Context) {
	session := sessions.Default(c)
	name := session.Get("Name")

	if name == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not logged In"})
		return
	}

	nameStr := name.(string)
	wsToken := GenerateWSToken(nameStr)

	c.JSON(http.StatusOK, gin.H{
		"name":     nameStr,
		"ws_token": wsToken,
	})
}
