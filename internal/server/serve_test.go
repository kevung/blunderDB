package server

import "testing"

// TestParseServeArgs_RejectsPositionalArgument guards #230: flag.FlagSet
// stops parsing at the first non-flag argument, so a flag placed after a
// stray positional one used to be silently dropped instead of rejected —
// `docker run image serve --addr :9090` (the ENTRYPOINT is already the bare
// binary, so "serve" itself is the first argument) started on the default
// :8080 without a word of complaint.
func TestParseServeArgs_RejectsPositionalArgument(t *testing.T) {
	if _, err := parseServeArgs([]string{"--addr", ":9090", "extra"}); err == nil {
		t.Fatal("expected an error for a trailing positional argument, got nil")
	}
	if _, err := parseServeArgs([]string{"bogus", "--addr", ":9090"}); err == nil {
		t.Fatal("expected an error for a leading positional argument that isn't \"serve\", got nil")
	}
}

// TestParseServeArgs_TreatsLeadingServeAsFlags is the flip side: the one
// tolerated positional token is a literal leading "serve" (what an ENTRYPOINT
// ["blunderdb"] + CMD ["serve"] passes through), and flags after it are still
// honoured rather than discarded.
func TestParseServeArgs_TreatsLeadingServeAsFlags(t *testing.T) {
	cfg, err := parseServeArgs([]string{"serve", "--addr", ":9090"})
	if err != nil {
		t.Fatalf("parseServeArgs: %v", err)
	}
	if cfg.addr != ":9090" {
		t.Errorf("addr = %q, want %q (a leading \"serve\" token must not swallow the flags after it)", cfg.addr, ":9090")
	}
}

// TestParseServeArgs_EnvFallbacks pins the four BLUNDERDB_* environment
// variables that --metrics, --cors-allow-origin, --rate-limit-rps and
// --rate-limit-burst were missing (#230) — every other serve flag already
// had one, and the asymmetry invites forgetting the rate limit in a compose
// file that only sets environment variables.
func TestParseServeArgs_EnvFallbacks(t *testing.T) {
	t.Setenv("BLUNDERDB_METRICS", "false")
	t.Setenv("BLUNDERDB_CORS_ALLOW_ORIGIN", "https://example.test")
	t.Setenv("BLUNDERDB_RATE_LIMIT_RPS", "12.5")
	t.Setenv("BLUNDERDB_RATE_LIMIT_BURST", "7")

	cfg, err := parseServeArgs(nil)
	if err != nil {
		t.Fatalf("parseServeArgs: %v", err)
	}
	if cfg.enableMetrics {
		t.Error("enableMetrics = true, want false from BLUNDERDB_METRICS=false")
	}
	if cfg.corsOrigin != "https://example.test" {
		t.Errorf("corsOrigin = %q, want %q from BLUNDERDB_CORS_ALLOW_ORIGIN", cfg.corsOrigin, "https://example.test")
	}
	if cfg.rateLimitRPS != 12.5 {
		t.Errorf("rateLimitRPS = %v, want 12.5 from BLUNDERDB_RATE_LIMIT_RPS", cfg.rateLimitRPS)
	}
	if cfg.rateLimitBurst != 7 {
		t.Errorf("rateLimitBurst = %v, want 7 from BLUNDERDB_RATE_LIMIT_BURST", cfg.rateLimitBurst)
	}

	// An explicit flag still overrides the environment.
	cfg2, err := parseServeArgs([]string{"--rate-limit-rps", "99"})
	if err != nil {
		t.Fatalf("parseServeArgs: %v", err)
	}
	if cfg2.rateLimitRPS != 99 {
		t.Errorf("rateLimitRPS = %v, want 99 (explicit flag overrides env)", cfg2.rateLimitRPS)
	}
}

// TestParseServeArgs_RateLimitDefaultsOn guards the third fix in #230: the
// rate limiter used to default to disabled (opt-in); it is now on by
// default at a generous rate, so a bare compose file that only points at a
// database no longer ships with no throttling at all.
func TestParseServeArgs_RateLimitDefaultsOn(t *testing.T) {
	cfg, err := parseServeArgs(nil)
	if err != nil {
		t.Fatalf("parseServeArgs: %v", err)
	}
	if cfg.rateLimitRPS != defaultRateLimitRPS {
		t.Errorf("rateLimitRPS default = %v, want %v", cfg.rateLimitRPS, defaultRateLimitRPS)
	}
	if cfg.rateLimitBurst != defaultRateLimitBurst {
		t.Errorf("rateLimitBurst default = %v, want %v", cfg.rateLimitBurst, defaultRateLimitBurst)
	}
}
