package cli

import (
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
)

// formatMWC7 renders a 7-point MWC loss (ADR-0075) for the text outputs: the
// loss in percent with its 95 % interval, then its Elo reading against the
// engine. A money game has none, and says so rather than print a zero.
func formatMWC7(e domain.MWC7) string {
	if !e.Available {
		return "— (money play: no match length)"
	}
	s := fmt.Sprintf("%.1f %%", 100*e.Loss)
	if e.HasInterval {
		s += fmt.Sprintf(" [%.1f, %.1f]", 100*e.Low, 100*e.High)
	}
	at := ""
	if e.EloFloored {
		at = "≤ "
	}
	s += fmt.Sprintf("  (Elo vs engine %s%.0f", at, e.Elo)
	if e.HasInterval {
		s += fmt.Sprintf(" [%.0f, %.0f]", e.EloLow, e.EloHigh)
	}
	return s + ")"
}
