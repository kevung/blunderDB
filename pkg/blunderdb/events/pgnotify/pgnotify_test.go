package pgnotify

import (
	"context"
	"testing"
	"time"

	"github.com/kevung/blunderdb/pkg/blunderdb/events"
)

// TestTransport_StartFailsWithoutDatabase: a daemon that cannot listen does not start.
func TestTransport_StartFailsWithoutDatabase(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := Start(ctx, "postgres://nobody@127.0.0.1:1/none?sslmode=disable&connect_timeout=1", events.NewBus(0), Options{}); err == nil {
		t.Fatal("Start succeeded without a database")
	}
}
