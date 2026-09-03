// Command openapi-gen (re)generates blunderDB's OpenAPI contract
// (openapi.yaml, repo root) and its Sphinx annex
// (doc/source/api_reference.rst) from internal/server's route table — see
// internal/server/openapigen's package doc comment for why this parses Go
// source rather than reflecting on a running server.
//
// Usage (from the repo root):
//
//	go run ./cmd/openapi-gen                 # writes both files
//	go run ./cmd/openapi-gen -check           # exits 1 if either is stale
//	go run ./cmd/openapi-gen -openapi <path> -rst <path>
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kevung/blunderdb/internal/server/openapigen"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	fs := flag.NewFlagSet("openapi-gen", flag.ContinueOnError)
	serverDir := fs.String("server-dir", "internal/server", "directory holding the handlers_*.go source to parse (repo-root-relative or absolute)")
	openapiPath := fs.String("openapi", "openapi.yaml", "output path for the OpenAPI document")
	rstPath := fs.String("rst", "doc/source/api_reference.rst", "output path for the Sphinx annex")
	check := fs.Bool("check", false, "don't write; exit 1 if either file would change (CI/test use)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	model, err := openapigen.Parse(*serverDir)
	if err != nil {
		return fmt.Errorf("openapi-gen: %w", err)
	}
	openapiDoc := openapigen.GenerateOpenAPI(model)
	rstDoc := openapigen.GenerateAPIReferenceRST(model)

	if *check {
		staleOpenAPI, err := isStale(*openapiPath, openapiDoc)
		if err != nil {
			return err
		}
		staleRST, err := isStale(*rstPath, rstDoc)
		if err != nil {
			return err
		}
		if staleOpenAPI || staleRST {
			return fmt.Errorf("stale: run `go run ./cmd/openapi-gen` and commit the result (openapi=%v rst=%v)", staleOpenAPI, staleRST)
		}
		return nil
	}

	if err := os.WriteFile(*openapiPath, []byte(openapiDoc), 0o644); err != nil {
		return fmt.Errorf("openapi-gen: write %s: %w", *openapiPath, err)
	}
	if err := os.WriteFile(*rstPath, []byte(rstDoc), 0o644); err != nil {
		return fmt.Errorf("openapi-gen: write %s: %w", *rstPath, err)
	}
	fmt.Printf("wrote %s (%d routes) and %s\n", *openapiPath, len(model.Routes), *rstPath)
	return nil
}

func isStale(path, want string) (bool, error) {
	got, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, err
	}
	return string(got) != want, nil
}
