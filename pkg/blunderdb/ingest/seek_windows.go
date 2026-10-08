//go:build windows

package ingest

import "syscall"

// errNegativeSeek is Windows' answer to a seek before the file's start; it
// plays the part EINVAL plays elsewhere.
const errNegativeSeek = syscall.Errno(131) // ERROR_NEGATIVE_SEEK
