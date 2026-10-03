package sqlshared

// SetSearchChunk lets an external test force a scan across several chunks
// on a small library; the returned func restores the size.
func SetSearchChunk(n int) (restore func()) {
	old := searchChunk
	searchChunk = n
	return func() { searchChunk = old }
}
