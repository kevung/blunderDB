package ingest

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/kevung/blunderdb/pkg/blunderdb/storage"
)

func TestWrapInvalid_ParserRefusalIsInvalid(t *testing.T) {
	if err := wrapInvalid(errors.New("unexpected token")); !errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("a parser refusal must be ErrInvalid, got %v", err)
	}
}

func TestWrapInvalid_ReadFailureStaysAFault(t *testing.T) {
	readErr := &fs.PathError{Op: "open", Path: "/tmp/upload-123/match.xg", Err: fs.ErrPermission}
	if err := wrapInvalid(readErr); errors.Is(err, storage.ErrInvalid) {
		t.Fatalf("a failure to read the file is not the input's fault, got %v", err)
	}
}
