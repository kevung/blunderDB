package structure

import (
	"context"
	"errors"
	"testing"
)

// countedPages counts the loads of each page, and cancels after the
// cancelAt-th load when cancel is set.
type countedPages struct {
	memPages
	loads    map[int]int
	n        int
	cancelAt int
	cancel   context.CancelFunc
}

func (c *countedPages) Page(i int) (Page, error) {
	c.loads[i]++
	if c.n++; c.cancel != nil && c.n == c.cancelAt {
		c.cancel()
	}
	return c.memPages.Page(i)
}

// TestReadLoadsEachPageOnce: the pages the shape is learned on are not
// loaded, and so not decoded, a second time.
func TestReadLoadsEachPageOnce(t *testing.T) {
	src := &countedPages{memPages: inkBook(t, 0), loads: map[int]int{}}
	if _, err := Read(context.Background(), src); err != nil {
		t.Fatal(err)
	}
	for i := range src.memPages {
		if src.loads[i] != 1 {
			t.Errorf("page %d loaded %d times", i+1, src.loads[i])
		}
	}
}

// TestReadCancelled: a document is stopped between its pages.
func TestReadCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	src := &countedPages{memPages: inkBook(t, 0), loads: map[int]int{}, cancelAt: 2, cancel: cancel}
	if _, err := Read(ctx, src); !errors.Is(err, context.Canceled) {
		t.Errorf("a cancelled read: %v", err)
	}
	if src.n != 2 {
		t.Errorf("%d pages loaded after the cancellation at the 2nd", src.n)
	}
}
