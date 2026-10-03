package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kevung/blunderdb/pkg/blunderdb/domain"
	"github.com/kevung/blunderdb/pkg/blunderdb/engine/rollout"
	"github.com/kevung/blunderdb/pkg/blunderdb/rollouts"
	"github.com/kevung/blunderdb/pkg/blunderdb/searchquery"
)

// runAnalyzeRollout is `analyze --rollout`: Database.RolloutPositions over the
// positions query selects, the loop the GUI and serve run too.
func (cli *CLI) runAnalyzeRollout(spec, query string, jobs int, text bool) error {
	s, err := rollout.ParseSpec(spec)
	if err != nil {
		return err
	}
	s.Workers = max(jobs, 0)
	filters, err := parseRolloutQuery(query)
	if err != nil {
		return err
	}

	var sum rollouts.Summary
	runErr := withInterruptibleContext(nil, func(ctx context.Context) error {
		todo, err := cli.db.PositionsToRollout(ctx, filters, s)
		if err != nil {
			return err
		}
		if len(todo) == 0 {
			if text {
				fmt.Println("Nothing to do: every selected position already carries a rollout with these settings.")
			}
			return nil
		}
		if text {
			fmt.Printf("Rolling out %d position(s): %s\n", len(todo), s.DepthLabel())
		}
		last := -1
		sum, err = cli.db.RolloutPositions(ctx, todo, s, func(p rollouts.Progress) {
			if !text || p.Done == last {
				return
			}
			last = p.Done
			fmt.Printf("  %d/%d\n", p.Done, p.Total)
		})
		return err
	})
	if errors.Is(runErr, context.Canceled) {
		sum.Cancelled = true
	}
	if runErr != nil && !sum.Cancelled {
		return fmt.Errorf("analyze --rollout failed: %w", runErr)
	}
	if !text {
		return printJSON(analyzeRolloutResult{Summary: sum, Settings: s, Signature: s.Signature()})
	}
	if sum.Total == 0 && !sum.Cancelled {
		return nil
	}
	if sum.Cancelled {
		fmt.Println("Cancelled.")
	} else {
		fmt.Println("Done.")
	}
	fmt.Printf("rolled out: %d, refused: %d, failed: %d\n", sum.RolledOut, sum.Refused, sum.Failed)
	return nil
}

// analyzeRolloutResult is the --format json shape of `analyze --rollout`.
type analyzeRolloutResult struct {
	rollouts.Summary
	Settings  rollout.Settings `json:"settings"`
	Signature string           `json:"signature"`
}

// parseRolloutQuery reads query as `search --query` does, refusing a token
// nothing claims: a sweep over the whole library by a typo is worse than a
// refusal.
func parseRolloutQuery(query string) (domain.SearchFilters, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return domain.SearchFilters{}, nil
	}
	filters, diags := searchquery.Parse(q)
	var unknown []string
	for _, d := range diags {
		if d.Kind == searchquery.DiagUnknown {
			unknown = append(unknown, d.Token)
		}
	}
	if len(unknown) > 0 {
		return domain.SearchFilters{}, fmt.Errorf("unknown token(s) in --query: %s (see search --query-help)", strings.Join(unknown, ", "))
	}
	return filters, nil
}
