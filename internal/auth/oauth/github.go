package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

const (
	githubUserURL   = "https://api.github.com/user"
	githubEmailsURL = "https://api.github.com/user/emails"
)

type GithubProvider struct {
	cfg *oauth2.Config
}

func NewGithubProvider(clientID, clientSecret, redirectURL string) *GithubProvider {
	return &GithubProvider{
		cfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     github.Endpoint,
			Scopes:       []string{"read:user", "user:email"},
		},
	}
}

func (p *GithubProvider) AuthURL(state string) string {
	return p.cfg.AuthCodeURL(state)
}

func (p *GithubProvider) Exchange(ctx context.Context, code string) (OAuthProfile, error) {
	tok, err := p.cfg.Exchange(ctx, code)
	if err != nil {
		return OAuthProfile{}, fmt.Errorf("github exchange: %w", err)
	}
	client := p.cfg.Client(ctx, tok)

	var user struct {
		ID    int64  `json:"id"`
		Name  string `json:"name"`
		Login string `json:"login"`
		Email string `json:"email"`
	}
	if err := getJSON(ctx, client, githubUserURL, &user); err != nil {
		return OAuthProfile{}, fmt.Errorf("github user: %w", err)
	}

	displayName := user.Name
	if displayName == "" {
		displayName = user.Login
	}

	profile := OAuthProfile{
		ProviderUserID: strconv.FormatInt(user.ID, 10),
		DisplayName:    displayName,
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := getJSON(ctx, client, githubEmailsURL, &emails); err != nil {
		return OAuthProfile{}, fmt.Errorf("github emails: %w", err)
	}
	for _, e := range emails {
		if e.Primary {
			profile.Email = e.Email
			profile.EmailVerified = e.Verified
			break
		}
	}

	return profile, nil
}

func getJSON(ctx context.Context, client *http.Client, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", res.StatusCode)
	}
	return json.NewDecoder(res.Body).Decode(dst)
}
