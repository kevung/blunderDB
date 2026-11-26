// Package parallel splits a loop of independent iterations across the
// processors, for the image decoders of the document import.
package parallel

import (
	"runtime"
	"sync"
)

// For runs fn over [0, n) cut into at most GOMAXPROCS ranges of at
// least grain items, each on its own goroutine, and returns once all have
// run. A panic in one is raised again on the caller's goroutine, where the
// decoder's recover turns it into an error as it would without the split.
func For(n, grain int, fn func(lo, hi int)) {
	parts := min(runtime.GOMAXPROCS(0), n/max(grain, 1))
	if parts <= 1 {
		fn(0, n)
		return
	}
	var wg sync.WaitGroup
	var mu sync.Mutex
	var fault any
	for p := range parts {
		lo, hi := n*p/parts, n*(p+1)/parts
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if e := recover(); e != nil {
					mu.Lock()
					if fault == nil {
						fault = e
					}
					mu.Unlock()
				}
			}()
			fn(lo, hi)
		}()
	}
	wg.Wait()
	if fault != nil {
		panic(fault)
	}
}
