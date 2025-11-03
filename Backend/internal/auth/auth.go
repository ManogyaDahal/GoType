package auth

import (
	"os"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

//defines scopes
var ScopeEmail string =  "https://www.googleapis.com/auth/userinfo.email" 
var ScopeProfile string = "https://www.googleapis.com/auth/userinfo.profile"
//From where it fetches data 
var UserInfo string  = "https://www.googleapis.com/oauth2/v2/userinfo" 

// returns the Initialized oauth configuration
func InitOauth() *oauth2.Config {
	return &oauth2.Config{ 
		ClientID: os.Getenv("GOOGLE_CLIENT_ID"),	
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		Endpoint: google.Endpoint,
		RedirectURL: "http://localhost:8080/auth/google/callback" ,
		Scopes: []string{ ScopeEmail, ScopeProfile},
	}
}
