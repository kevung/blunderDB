package server

import (
	"bufio"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

// TestFilterSetPinned drives /v1/filters.setPinned end to end: a pin shows on
// filters.list, an unknown id is a 404, and the pin stays in its tenant.
func TestFilterSetPinned(t *testing.T) {
	ts := newTestServer(t)

	save := postAs(t, ts, "1", "/v1/filters.save", map[string]string{"name": "fav", "command": "s E>100"})
	var saved struct {
		ID int64 `json:"id"`
	}
	if err := json.NewDecoder(save.Body).Decode(&saved); err != nil {
		t.Fatal(err)
	}
	save.Body.Close()

	pin := postAs(t, ts, "1", "/v1/filters.setPinned", map[string]any{"id": saved.ID, "pinned": true})
	pin.Body.Close()
	if pin.StatusCode != http.StatusOK {
		t.Fatalf("filters.setPinned status = %d, want 200", pin.StatusCode)
	}

	list := postAs(t, ts, "1", "/v1/filters.list", nil)
	defer list.Body.Close()
	var got []storage.Filter
	sc := bufio.NewScanner(list.Body)
	for sc.Scan() {
		var f storage.Filter
		if err := json.Unmarshal(sc.Bytes(), &f); err != nil {
			t.Fatalf("decode %q: %v", sc.Text(), err)
		}
		got = append(got, f)
	}
	if len(got) != 1 || !got[0].Pinned {
		t.Fatalf("filters.list = %+v, want the one filter, pinned", got)
	}

	unknown := postAs(t, ts, "1", "/v1/filters.setPinned", map[string]any{"id": saved.ID + 99, "pinned": true})
	unknown.Body.Close()
	if unknown.StatusCode != http.StatusNotFound {
		t.Errorf("unknown id: status = %d, want 404", unknown.StatusCode)
	}
	other := postAs(t, ts, "2", "/v1/filters.setPinned", map[string]any{"id": saved.ID, "pinned": false})
	other.Body.Close()
	if other.StatusCode != http.StatusNotFound {
		t.Errorf("another tenant's filter: status = %d, want 404", other.StatusCode)
	}
}
