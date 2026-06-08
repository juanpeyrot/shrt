package validators

import (
	"regexp"

	"shrt/internal/apierr"
)

var slugRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,16}$`)

func ValidateSlug(slug string) error {
  if !slugRegex.MatchString(slug) {
    return apierr.NewValidation("short_code must be 3-16 characters and can only contain letters, numbers, underscores, and hyphens")
  }
  return nil
}