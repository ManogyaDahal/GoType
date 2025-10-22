package auth

import (
	"os"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var Email string =  "https://www.googleapis.com/auth/userinfo.email" 
var Profile string = "https://www.googleapis.com/auth/userinfo.profile"

// returns the Initialized oauth configuration
func InitOauth() *oauth2.Config {
	return &oauth2.Config{ 
		ClientID: os.Getenv("GOOGLE_CLIENT_ID"),	
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Endpoint: google.Endpoint,
		RedirectURL: "http://localhost:8080/auth/google/callback" ,
		Scopes: []string{ Email, Profile},
	}
}
