package sqlite

import "github.com/kevung/blunderdb/pkg/blunderdb/storage/sqlshared"

// SharedExecer exposes a Storage's execer to the external tests that wrap it.
func SharedExecer(s *Storage) sqlshared.Execer { return s.shared() }
