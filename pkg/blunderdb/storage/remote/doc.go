// Package remote is the third backend of the storage contract: it implements
// storage.Storage over the /v1 API of a `blunderdb serve` daemon (HTTP + JSON,
// NDJSON for the streamed lists), so a client can work on a distant tenant
// without the daemon gaining a single route.
//
// What it promises (ADR-0095):
//
//   - Each method is one /v1 call, atomic only as far as that route's handler
//     is. There is no transaction across calls: BeginTx answers ErrUnsupported
//     rather than pretend a rollback it cannot perform.
//   - A contract method with no /v1 route answers an *UnsupportedError (it
//     matches ErrUnsupported and storage.ErrInternal), never a fake success.
//     Unsupported lists them all; unsupported_gen.go is regenerated with
//     `go generate` when the contract or the routes move.
//   - The daemon's error envelope comes back as the storage sentinels
//     (not_found → storage.ErrNotFound, conflict → storage.ErrConflict,
//     invalid → storage.ErrInvalid); a transport failure or a daemon that
//     does not answer the envelope comes back as ErrUnavailable.
//   - No authentication lives here (ADR-0005): the backend sends the headers
//     it is configured with — a proxy token, say — and X-Tenant-ID, and the
//     authenticating proxy in front of the daemon decides.
package remote

//go:generate go run gen_stubs.go
