package issuance

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// IssueRecord is one line of the Issue register: a copy this database produced, and enough
// about it to recognise that copy if it ever comes back.
type IssueRecord struct {
	Distribution string `json:"distribution"`
	Recipient    string `json:"recipient,omitempty"`
	Number       int    `json:"number,omitempty"`
	Total        int    `json:"total,omitempty"`
	CopyID       string `json:"copyId"`
	// Signature is the copy's own Watermark signature — the value that identifies a found
	// file against this register without ambiguity.
	Signature string `json:"signature"`
	// Salt is the Distribution's salt, kept here so every later copy of the same course
	// reuses it and their Holder fingerprints stay comparable.
	Salt     string `json:"salt,omitempty"`
	IssuedAt string `json:"issuedAt"`
	FileName string `json:"fileName,omitempty"`
	Contents string `json:"contents,omitempty"`
	// Password is the Distribution's transport password, kept here because the Issuer
	// will need it months later and has nowhere else to put it. It is the single strongest
	// reason the Issue register must never travel inside an Issued copy.
	Password string `json:"password,omitempty"`
}

// IssueRegister is the Issuer's own list of the copies they produced.
type IssueRegister struct {
	Records []IssueRecord `json:"records"`
}

// Add appends a record, stamping the time when the caller left it blank.
func (reg *IssueRegister) Add(rec IssueRecord, now time.Time) {
	if rec.IssuedAt == "" {
		rec.IssuedAt = now.UTC().Format(time.RFC3339)
	}
	reg.Records = append(reg.Records, rec)
}

// Find returns the record matching a Watermark signature, which is how a copy found in the
// wild is looked up.
func (reg IssueRegister) Find(signature string) (IssueRecord, bool) {
	for _, r := range reg.Records {
		if r.Signature == signature {
			return r, true
		}
	}
	return IssueRecord{}, false
}

// Distributions lists the distinct Distribution names in the register, most recent first,
// so the export screen can offer what the Issuer used last time instead of an empty field.
func (reg IssueRegister) Distributions() []string {
	seen := map[string]bool{}
	var out []string
	for i := len(reg.Records) - 1; i >= 0; i-- {
		name := reg.Records[i].Distribution
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	return out
}

// SaltFor returns the salt already in use for a Distribution, so a course issued in several
// batches keeps one salt and its copies stay comparable with each other. An unknown
// Distribution returns "", which tells Seal to mint a fresh one.
func (reg IssueRegister) SaltFor(distribution string) string {
	for i := len(reg.Records) - 1; i >= 0; i-- {
		if reg.Records[i].Distribution == distribution && reg.Records[i].Salt != "" {
			return reg.Records[i].Salt
		}
	}
	return ""
}

// NextNumber returns the number to give the next copy of a Distribution, so a second batch
// continues the first rather than restarting at one.
func (reg IssueRegister) NextNumber(distribution string) int {
	highest := 0
	for _, r := range reg.Records {
		if r.Distribution == distribution && r.Number > highest {
			highest = r.Number
		}
	}
	return highest + 1
}

// EncodeRegister renders the Issue register for storage. An empty register encodes to "".
func EncodeRegister(reg IssueRegister) (string, error) {
	if len(reg.Records) == 0 {
		return "", nil
	}
	b, err := json.Marshal(reg)
	return string(b), err
}

// DecodeRegister parses a stored Issue register.
func DecodeRegister(s string) (IssueRegister, error) {
	if strings.TrimSpace(s) == "" {
		return IssueRegister{}, nil
	}
	var reg IssueRegister
	if err := json.Unmarshal([]byte(s), &reg); err != nil {
		return IssueRegister{}, fmt.Errorf("unreadable issue register: %w", err)
	}
	return reg, nil
}
