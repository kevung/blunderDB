package issuance

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/adrg/xdg"
)

// appDir is blunderDB's own folder inside the XDG config home — the same one config.yaml
// lives in, so a producer's identity sits next to the rest of their settings and is backed
// up by whatever already backs those up.
const appDir = "blunderDB"

// ConfigDir returns the directory holding the Issuer identity. A writable config directory
// is an *essential* host capability (ADR-0004): without it there is nowhere to keep a
// durable identity, and every export would be signed by a throwaway key that no later export
// could be matched against.
func ConfigDir() string {
	return filepath.Join(xdg.ConfigHome, appDir)
}

// DefaultIssuerName is the label a freshly created identity starts with. It is only a display
// name — the producer can change it, and watermarks already sealed keep the name they carry.
func DefaultIssuerName() string {
	if name := firstNonEmpty(os.Getenv("USER"), os.Getenv("USERNAME")); name != "" {
		return name
	}
	return "blunderDB"
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

var nonFilename = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// FileSlug turns a person's name into something safe to put in a filename on every platform,
// without mangling it beyond recognition: letters and digits survive, separators do not.
func FileSlug(s string) string {
	slug := nonFilename.ReplaceAllString(strings.TrimSpace(s), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "identite"
	}
	if len(slug) > 60 {
		slug = strings.Trim(slug[:60], "-")
	}
	return strings.ToLower(slug)
}
