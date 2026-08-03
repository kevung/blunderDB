package issuance

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/adrg/xdg"
)

// appDir is blunderDB's own folder inside the XDG config home — the same one config.yaml
// lives in, so an Issuer's identity sits next to the rest of their settings and is backed
// up by whatever already backs those up.
const appDir = "blunderDB"

// ConfigDir returns the directory holding the Issuer identity. A writable config directory
// is an *essential* host capability (ADR-0004): without it there is nowhere to keep a
// durable identity, so callers surface the failure rather than silently signing with a
// throwaway key that would make every emission unattributable to the next one.
func ConfigDir() string {
	return filepath.Join(xdg.ConfigHome, appDir)
}

// DefaultIssuerName is the label a freshly created identity starts with. It is only a
// display name — the Issuer can change it, and copies already sealed keep the name they
// were sealed with.
func DefaultIssuerName() string {
	if name := firstNonEmpty(os.Getenv("USER"), os.Getenv("USERNAME")); name != "" {
		return name
	}
	return "blunderDB"
}

var nonFilename = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// FileSlug turns a Distribution or recipient name into something safe to put in a filename
// on every platform, without mangling it beyond recognition: accents survive, separators do
// not.
func FileSlug(s string) string {
	slug := nonFilename.ReplaceAllString(strings.TrimSpace(s), "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "exemplaire"
	}
	if len(slug) > 60 {
		slug = strings.Trim(slug[:60], "-")
	}
	return strings.ToLower(slug)
}

// CopyFileName is the name given to one Issued copy: the Distribution, then the recipient
// when there is one, so a folder of twenty-four files sorts and reads sensibly.
func CopyFileName(distribution, recipient string, encrypted bool) string {
	name := FileSlug(distribution)
	if strings.TrimSpace(recipient) != "" {
		name += "_" + FileSlug(recipient)
	}
	if encrypted {
		return name + ContainerExtension
	}
	return name + ".db"
}
