package oauth

import "context"

type OAuthProfile struct {
	ProviderUserID string
	Email          string
	EmailVerified  bool
	DisplayName    string
}

type Provider interface {
	AuthURL(state string) string
	Exchange(ctx context.Context, code string) (OAuthProfile, error)
}

type Registry map[string]Provider

func (r Registry) Get(name string) (Provider, bool) {
	p, ok := r[name]
	return p, ok
}
