package issuance

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strings"
	"time"
)

// Holder is one machine that has opened an Issued copy — never one opening. A student who
// works seriously produces hundreds of openings and one Holder; what matters for a dispute
// is how many *distinct* machines a copy reached, and a per-opening log would bury exactly
// that number.
type Holder struct {
	// Fingerprint is one-way and salted per Distribution. It identifies nothing on its
	// own: it can be compared with the fingerprint another copy of the same Distribution
	// shows, and it is meaningless anywhere else.
	Fingerprint string `json:"fingerprint"`
	FirstSeen   string `json:"firstSeen"`
	LastSeen    string `json:"lastSeen"`
	Openings    int    `json:"openings"`
	// Link chains this entry to the one before it. Only the immutable fields take part,
	// so the counters above can keep moving without disturbing the chain.
	Link string `json:"link"`
}

// Registry is the Holder registry of one Issued copy.
type Registry struct {
	Holders []Holder `json:"holders"`
}

// MachineFingerprint derives this machine's pseudonymous fingerprint for a Distribution.
// The salt comes from the Watermark, so the same machine looks different in two different
// Distributions and no global machine identifier is ever constituted.
func MachineFingerprint(salt string) string {
	return hex.EncodeToString(hashBytes([]byte(salt + "\x00" + machineTraits())))[:16]
}

// machineTraits gathers what the host exposes about "which machine is this". Neither value
// is stored — only the hash is — and a host that reveals neither degrades to a constant
// rather than failing: a registry that cannot distinguish machines is far better than an
// open that errors out.
func machineTraits() string {
	var username string
	if u, err := user.Current(); err == nil {
		username = u.Username
	}
	if username == "" {
		username = firstNonEmpty(os.Getenv("USER"), os.Getenv("USERNAME"))
	}
	host, err := os.Hostname()
	if err != nil {
		host = ""
	}
	return username + "@" + host
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// Record notes that fingerprint has just opened the copy: a new Holder the first time, an
// updated last-seen and count afterwards. genesis anchors the chain — pass the Watermark's
// signature, which the Issuer sealed and no Holder can rewrite.
func (r *Registry) Record(genesis, fingerprint string, now time.Time) {
	stamp := now.UTC().Format(time.RFC3339)
	for i := range r.Holders {
		if r.Holders[i].Fingerprint == fingerprint {
			r.Holders[i].LastSeen = stamp
			r.Holders[i].Openings++
			return
		}
	}
	h := Holder{Fingerprint: fingerprint, FirstSeen: stamp, LastSeen: stamp, Openings: 1}
	h.Link = chainLink(r.previousLink(genesis), h)
	r.Holders = append(r.Holders, h)
}

func (r Registry) previousLink(genesis string) string {
	if len(r.Holders) == 0 {
		return genesis
	}
	return r.Holders[len(r.Holders)-1].Link
}

func chainLink(prev string, h Holder) string {
	return hex.EncodeToString(hashBytes([]byte(prev + "\x00" + h.Fingerprint + "\x00" + h.FirstSeen)))[:16]
}

// ChainIntact reports whether the registry still reads as an unbroken sequence from
// genesis. It catches an entry removed or reordered from the middle; it cannot catch the
// whole document being deleted, which nothing on a plain SQLite file could.
func (r Registry) ChainIntact(genesis string) bool {
	prev := genesis
	for _, h := range r.Holders {
		if h.Link != chainLink(prev, h) {
			return false
		}
		prev = h.Link
	}
	return true
}

// EncodeRegistry renders a Registry for storage. An empty Registry encodes to "" so a copy
// that has never been opened carries no row.
func EncodeRegistry(r Registry) (string, error) {
	if len(r.Holders) == 0 {
		return "", nil
	}
	b, err := json.Marshal(r)
	return string(b), err
}

// DecodeRegistry parses a stored Registry.
func DecodeRegistry(s string) (Registry, error) {
	if strings.TrimSpace(s) == "" {
		return Registry{}, nil
	}
	var r Registry
	if err := json.Unmarshal([]byte(s), &r); err != nil {
		return Registry{}, fmt.Errorf("unreadable holder registry: %w", err)
	}
	return r, nil
}
