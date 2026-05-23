package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// NewGoogleOAuthConfig создаёт конфиг Google OAuth 2.0.
func NewGoogleOAuthConfig(clientID, secret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: secret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

// GetGoogleLoginURL возвращает URL для редиректа на Google OAuth.
func GetGoogleLoginURL(config *oauth2.Config, state string) string {
	return config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

// ExchangeCode обменивает код на токен.
func ExchangeCode(config *oauth2.Config, code string) (*oauth2.Token, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}
	return token, nil
}

// GoogleUserInfo содержит данные пользователя из Google.
type GoogleUserInfo struct {
	ID      string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// GetGoogleUserInfo получает данные пользователя из Google по токену.
func GetGoogleUserInfo(token *oauth2.Token) (*GoogleUserInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client := oauth2.NewClient(ctx, oauth2.StaticTokenSource(token))
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, fmt.Errorf("get google user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned status %d", resp.StatusCode)
	}

	var info GoogleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode google user info: %w", err)
	}
	return &info, nil
}
