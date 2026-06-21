package validators

import (
	"net/url"
	"strings"

	"shrt/internal/apierr"
)

func ValidateURL(raw string) error {
	if raw == "" {
		return apierr.NewValidation("url is required")
	}
	if len(raw) > 2048 {
		return apierr.NewValidation("url is too long")
	}

	u, err := url.ParseRequestURI(raw)
	if err != nil {
		return apierr.NewValidation("url is invalid")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return apierr.NewValidation("url must start with http or https")
	}

	if u.Host == "" {
		return apierr.NewValidation("url is invalid")
	}

	host := u.Hostname()
	if !strings.Contains(host, ".") && host != "localhost" {
		return apierr.NewValidation("url is invalid")
	}

	return nil
}
