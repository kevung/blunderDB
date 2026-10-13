package database

import (
	"context"
	"fmt"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// MainPlayerPlan is what designating a Main player swaps and leaves.
type MainPlayerPlan = storage.MainPlayerPlan

// MainPlayer returns the library's Main player under their canonical name,
// "" when none is designated.
func (d *Database) MainPlayer() (string, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return "", fmt.Errorf("no database is currently open")
	}
	return storage.MainPlayer(context.Background(), d.store, "")
}

// PlanMainPlayer returns what designating name would swap — the matches
// where they sit at seat 2 — and the ones whose swap is refused. Nothing is
// written.
func (d *Database) PlanMainPlayer(name string) (MainPlayerPlan, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return MainPlayerPlan{}, fmt.Errorf("no database is currently open")
	}
	return storage.PlanMainPlayer(context.Background(), d.store, "", name)
}

// SetMainPlayer designates name as the library's Main player and swaps every
// match where they sit at seat 2, in one transaction; every later import
// seats them first. It returns the plan it applied.
func (d *Database) SetMainPlayer(name string) (MainPlayerPlan, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return MainPlayerPlan{}, fmt.Errorf("no database is currently open")
	}
	return storage.DesignateMainPlayer(context.Background(), d.store, "", name)
}

// ClearMainPlayer forgets the Main player. No match is swapped back.
func (d *Database) ClearMainPlayer() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.db == nil {
		return fmt.Errorf("no database is currently open")
	}
	return storage.ClearMainPlayer(context.Background(), d.store, "")
}

// PlayerMatchIDs returns the ids of the matches name plays, at either seat
// and under any of their spellings: what a player's match count opens.
func (d *Database) PlayerMatchIDs(name string) ([]int64, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.db == nil {
		return nil, fmt.Errorf("no database is currently open")
	}
	return storage.PlayerMatchIDs(context.Background(), d.store, "", name)
}
