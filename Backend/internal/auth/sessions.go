package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func InitSesssion(router *gin.Engine) {
	secretKey := os.Getenv("SESSION_SECRET")
	store := cookie.NewStore([]byte(secretKey))

	env := os.Getenv("ENV")
	if env == "production" {
		store.Options(sessions.Options{
			Path:     "/",
			MaxAge:   7 * 86400,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   true,
		})
	} else {
		store.Options(sessions.Options{
			Path:     "/",
			MaxAge:   7 * 86400,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   false,
		})
	}

	router.Use(sessions.Sessions("session", store))
}

// signState produces an HMAC-SHA256 signature for the given message.
func signState(message string) string {
	secret := os.Getenv("SESSION_SECRET")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}

// GenerateState creates a self-validating HMAC-signed state token.
// Format: nonce.timestamp.signature
// This eliminates the need to store state in the session cookie,
// so the OAuth callback doesn't depend on cookies surviving the redirect chain.
func GenerateState() string {
	nonce := make([]byte, 16)
	rand.Read(nonce)
	nonceStr := base64.URLEncoding.EncodeToString(nonce)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())

	payload := nonceStr + "." + timestamp
	signature := signState(payload)

	return payload + "." + signature
}

// ValidateState verifies that the HMAC-signed state token is valid
// and was created within the last 10 minutes.
func ValidateState(state string) bool {
	parts := strings.SplitN(state, ".", 3)
	if len(parts) != 3 {
		return false
	}

	payload := parts[0] + "." + parts[1]
	signature := parts[2]

	// Verify HMAC signature
	expected := signState(payload)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return false
	}

	// Verify timestamp is within 10 minutes
	var ts int64
	fmt.Sscanf(parts[1], "%d", &ts)
	elapsed := time.Since(time.Unix(ts, 0))
	if elapsed > 10*time.Minute || elapsed < -1*time.Minute {
		return false
	}

	return true
}

// GenerateWSToken creates a short-lived HMAC-signed token for WebSocket auth.
// Format: urlencoded(name).timestamp.signature
// The frontend fetches this from /api/whoamI (through the Vercel proxy where
// the session is valid) and passes it as ?token= in the WebSocket URL, which
// connects directly to Render — bypassing the proxy entirely.
func GenerateWSToken(name string) string {
	encodedName := url.QueryEscape(name)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	payload := encodedName + "." + timestamp
	signature := signState(payload)
	return payload + "." + signature
}

// ValidateWSToken verifies the HMAC-signed WebSocket token and returns the
// embedded user name. Tokens are valid for 2 hours to accommodate long sessions.
func ValidateWSToken(token string) (string, bool) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return "", false
	}

	payload := parts[0] + "." + parts[1]
	signature := parts[2]

	// Verify HMAC signature — prevents forgery
	expected := signState(payload)
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return "", false
	}

	// Verify token is within 2 hours
	var ts int64
	fmt.Sscanf(parts[1], "%d", &ts)
	elapsed := time.Since(time.Unix(ts, 0))
	if elapsed > 2*time.Hour || elapsed < -1*time.Minute {
		return "", false
	}

	name, err := url.QueryUnescape(parts[0])
	if err != nil {
		return "", false
	}
	return name, true
}
