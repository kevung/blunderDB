//go:build !windows

package ingest

import "syscall"

// errNegativeSeek is the errno a seek before the file's start returns.
const errNegativeSeek = syscall.EINVAL
