package server

import "testing"

// TestParseServeArgs_RejectsPositionalArgument: flag.FlagSet stops at the
// first non-flag, so flags after a stray positional would be silently
// dropped; it must be rejected instead.
func TestParseServeArgs_RejectsPositionalArgument(t *testing.T) {
	if _, err := parseServeArgs([]string{"--addr", ":9090", "extra"}); err == nil {
		t.Fatal("expected an error for a trailing positional argument, got nil")
	}
	if _, err := parseServeArgs([]string{"bogus", "--addr", ":9090"}); err == nil {
		t.Fatal("expected an error for a leading positional argument that isn't \"serve\", got nil")
	}
}

// TestParseServeArgs_TreatsLeadingServeAsFlags: a leading literal "serve"
// (ENTRYPOINT ["blunderdb"] + CMD ["serve"]) is the one tolerated positional,
// and flags after it are honoured.
func TestParseServeArgs_TreatsLeadingServeAsFlags(t *testing.T) {
	cfg, err := parseServeArgs([]string{"serve", "--addr", ":9090"})
	if err != nil {
		t.Fatalf("parseServeArgs: %v", err)
	}
	if cfg.addr != ":9090" {
		t.Errorf("addr = %q, want %q (a leading \"serve\" token must not swallow the flags after it)", cfg.addr, ":9090")
	}
}

// TestParseServeArgs_EnvFallbacks pins the BLUNDERDB_* variables for
// --metrics, --cors-allow-origin and --rate-limit-*, so a compose file that
// only sets environment variables cannot forget the rate limit.
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

// TestParseServeArgs_RateLimitDefaultsOn: the rate limiter is on by default,
// at a generous rate.
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

// TestParseServeArgs_PprofAddrDefaultsOffAndHonoursFlagAndEnv: pprof is off
// by default (it exposes profiling with no tenant scoping); the flag or
// BLUNDERDB_PPROF_ADDR turns it on.
func TestParseServeArgs_PprofAddrDefaultsOffAndHonoursFlagAndEnv(t *testing.T) {
	cfg, err := parseServeArgs(nil)
	if err != nil {
		t.Fatalf("parseServeArgs: %v", err)
	}
	if cfg.pprofAddr != "" {
		t.Errorf("pprofAddr default = %q, want empty (disabled)", cfg.pprofAddr)
	}

	cfg2, err := parseServeArgs([]string{"--pprof-addr", "127.0.0.1:6060"})
	if err != nil {
		t.Fatalf("parseServeArgs: %v", err)
	}
	if cfg2.pprofAddr != "127.0.0.1:6060" {
		t.Errorf("pprofAddr = %q, want %q from --pprof-addr", cfg2.pprofAddr, "127.0.0.1:6060")
	}

	t.Setenv("BLUNDERDB_PPROF_ADDR", "127.0.0.1:6061")
	cfg3, err := parseServeArgs(nil)
	if err != nil {
		t.Fatalf("parseServeArgs: %v", err)
	}
	if cfg3.pprofAddr != "127.0.0.1:6061" {
		t.Errorf("pprofAddr = %q, want %q from BLUNDERDB_PPROF_ADDR", cfg3.pprofAddr, "127.0.0.1:6061")
	}
}
