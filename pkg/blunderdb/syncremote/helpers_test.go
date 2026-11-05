package syncremote_test

import (
	"encoding/json"
	"io"
	"net/http"
)

func jsonDecode(r *http.Request, v any) error { return json.NewDecoder(r.Body).Decode(v) }
func jsonEncode(w io.Writer, v any) error     { return json.NewEncoder(w).Encode(v) }
