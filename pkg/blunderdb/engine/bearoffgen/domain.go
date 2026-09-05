package bearoffgen

import "fmt"

// Kind tells the two table families apart.
type Kind int

const (
	// TwoSidedKind is the exact table over both players' races: it answers a
	// cube decision, and is what ADR-0009's "never estimated" verdict rests on.
	TwoSidedKind Kind = iota
	// OneSidedKind is the distribution of rolls-to-bear-off for one player,
	// which is what the EPC reads.
	OneSidedKind
)

// Domain names one generatable table.
type Domain struct {
	Kind     Kind
	Points   int
	Checkers int
}

// String is the short label the interface shows: "TS-06-11", "OS-06".
func (d Domain) String() string {
	if d.Kind == OneSidedKind {
		return fmt.Sprintf("OS-%02d", d.Points)
	}
	return fmt.Sprintf("TS-%02d-%02d", d.Points, d.Checkers)
}

// FileName is the name the table is stored under, matching the names gnubg and
// the retired download asset used.
func (d Domain) FileName() string {
	if d.Kind == OneSidedKind {
		return fmt.Sprintf("gnubg_os%d.bd", d.Points)
	}
	return fmt.Sprintf("gnubg_ts%dx%d.bd", d.Points, d.Checkers)
}

// Size is the exact byte size of a two-sided table: a 40-byte header plus four
// int16 per pair. For a one-sided table the data section is only known after
// generation, and Size returns 0 — ask the generator.
func (d Domain) Size() int64 {
	if d.Kind != TwoSidedKind {
		return 0
	}
	n := int64(NumPositions(d.Points, d.Checkers))
	return 40 + n*n*planeCount*2
}

// RAMNeeded is what generating this domain holds in memory: the table itself
// plus the precomputed successor lists. The caller compares it to what the
// machine has before starting a run that would otherwise die hours in.
func (d Domain) RAMNeeded() int64 {
	if d.Kind != TwoSidedKind {
		return 0
	}
	n := int64(NumPositions(d.Points, d.Checkers))
	table := n * n * planeCount * 2
	// The successor lists: 21 rolls per position, a handful of int per roll.
	// Measured at 49 MiB total for TS-06-06, of which the table is 6.8 MB.
	reach := n * 21 * 8 * 6
	return table + reach
}

// KnownFingerprints maps a domain to the SHA-256 of the file gnubg produces for
// it. A generated table whose hash is in this map is *verified*: it is the
// same bytes the reference implementation writes. One that is not in the map
// is *unverified* — usable, but nobody has compared it to gnubg.
//
// TS-06-06 and OS-06 are the two tables blunderDB shipped until ADR-0027;
// TS-06-11 is the retired `bearoff-data-1` download asset. OS-07 … OS-10 are
// the wider one-sided tables the EPC reads beyond the home board, each taken
// from `makebearoff -o <p>` on gnubg 1.08 and reproduced byte for byte by this
// generator (TestOneSided_WiderDomainsIdenticalToGnubg).
var KnownFingerprints = map[Domain]string{
	{Kind: TwoSidedKind, Points: 6, Checkers: 6}:   "9eac8a2c697dae8a09f2e5653022084b9e725df6c32950cb5299b273fc64500f",
	{Kind: TwoSidedKind, Points: 6, Checkers: 11}:  "c52133cd59a7db478a71d18c8f2093ba343200fa72ede8004c32c6778c724f46",
	{Kind: OneSidedKind, Points: 6, Checkers: 15}:  "38089567e87a681682bb4ff53f30d781af215fc04debbdff3f61b6db68547a49",
	{Kind: OneSidedKind, Points: 7, Checkers: 15}:  "35ba8efc361e858c468dd28684f4c5c5edc845ae5d4dbdb2dcb289efa6809e99",
	{Kind: OneSidedKind, Points: 8, Checkers: 15}:  "28446a06018c2350e3c1b9ea18ca60a783fb76175fb1b643d547c38ec33c76b2",
	{Kind: OneSidedKind, Points: 9, Checkers: 15}:  "4d2930498332a736c7c951c1715d9663d8a0ee597636226627cdb7e292b72dc3",
	{Kind: OneSidedKind, Points: 10, Checkers: 15}: "dcfa1fb32360cbc7474344f680cf5054fae86756b1e5a19a696b4cd01254642a",
}

// Verdict is what Verify concluded about a file.
type Verdict int

const (
	// Verified: the file hashes to what gnubg produces for its domain.
	Verified Verdict = iota
	// Unverified: the file is well-formed but its domain has no recorded
	// fingerprint. Nothing is wrong with it; nobody has checked it either.
	Unverified
	// Corrupt: the file's size or hash contradicts its own header.
	Corrupt
)

func (v Verdict) String() string {
	switch v {
	case Verified:
		return "verified"
	case Unverified:
		return "unverified"
	default:
		return "corrupt"
	}
}
