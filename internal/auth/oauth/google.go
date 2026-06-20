package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const googleUserInfoURL = "https://www.googleapis.com/oauth2/v3/userinfo"

type GoogleProvider struct {
	cfg *oauth2.Config
}

func NewGoogleProvider(clientID, clientSecret, redirectURL string) *GoogleProvider {
	return &GoogleProvider{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     google.Endpoint,
			Scopes:       []string{"openid", "email", "profile"},
		},
	}
}

func (p *GoogleProvider) AuthURL(state string) string {
	return p.cfg.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (p *GoogleProvider) Exchange(ctx context.Context, code string) (OAuthProfile, error) {
	tok, err := p.cfg.Exchange(ctx, code)
	if err != nil {
		return OAuthProfile{}, fmt.Errorf("google exchange: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return OAuthProfile{}, fmt.Errorf("google userinfo request: %w", err)
	}

	res, err := p.cfg.Client(ctx, tok).Do(req)
	if err != nil {
		return OAuthProfile{}, fmt.Errorf("google userinfo: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return OAuthProfile{}, fmt.Errorf("google userinfo: unexpected status %d", res.StatusCode)
	}

	var body struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return OAuthProfile{}, fmt.Errorf("google userinfo decode: %w", err)
	}

	return OAuthProfile{
		ProviderUserID: body.Sub,
		Email:          body.Email,
		EmailVerified:  body.EmailVerified,
		DisplayName:    body.Name,
	}, nil
}
