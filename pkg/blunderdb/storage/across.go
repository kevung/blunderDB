package storage

import (
	"context"
	"fmt"
	"iter"
)

// A read across tenants (ADR-0061; domain term: Read tenants, see CONTEXT.md)
// is the union of single-tenant reads, one per tenant, each result tagged with
// the tenant it came from. Nothing here decides who may read whom: the set is
// handed in by the caller (the serve daemon takes it from a header its
// authenticating proxy writes, ADR-0005). What this file guarantees is the
// shape of the read: no tenant outside the set is ever passed to a store, the
// writing tenant is always part of the set, and every row says its origin.
//
// No store method takes several scopes. A read across tenants calls the
// existing single-scope methods once per tenant, under a context carrying that
// tenant (WithTenant), so PostgreSQL's row-level security filters each call as
// it filters any other, and both backends answer through the code the contract
// suite already holds them to.

// MaxReadTenants bounds a read set, the writing tenant included. A read across
// tenants costs one store call per tenant: the bound keeps one request's cost
// predictable, and a club larger than this is split by its caller.
const MaxReadTenants = 64

// ReadTenants is the ordered set of tenants one read spans: the writing tenant
// first, then each other tenant in the order the caller listed it, each once.
// Build it with NewReadTenants; the zero value spans no tenant.
type ReadTenants []string

// NewReadTenants builds the read set of a request whose writing tenant is
// writer and whose caller listed the tenants in listed. writer always comes
// first, whether or not listed repeats it; duplicates are dropped. Every
// listed tenant must be a canonical tenant (ParseTenant), the empty scope
// excepted for writer alone: the desktop's implicit tenant reads only itself.
// A set larger than MaxReadTenants is refused with ErrInvalid.
func NewReadTenants(writer string, listed []string) (ReadTenants, error) {
	if _, err := ParseTenant(writer); err != nil {
		return nil, err
	}
	set := ReadTenants{writer}
	seen := map[string]bool{writer: true}
	for _, scope := range listed {
		if scope == "" {
			return nil, fmt.Errorf("%w: an empty tenant in the read tenants", ErrInvalidTenant)
		}
		if _, err := ParseTenant(scope); err != nil {
			return nil, err
		}
		if seen[scope] {
			continue
		}
		seen[scope] = true
		set = append(set, scope)
	}
	if len(set) > MaxReadTenants {
		return nil, fmt.Errorf("%w: %d read tenants, at most %d", ErrInvalid, len(set), MaxReadTenants)
	}
	return set, nil
}

// Contains reports whether scope is one of the set's tenants.
func (r ReadTenants) Contains(scope string) bool {
	for _, s := range r {
		if s == scope {
			return true
		}
	}
	return false
}

// Tagged is one result of a read across tenants, with the tenant it was read
// from. A row id is unique within its tenant only: (Tenant, id) names a row.
type Tagged[T any] struct {
	Tenant string
	Item   T
}

// tenantContext scopes ctx to one tenant of a read set, for the RLS GUC. A
// hand-built set that skipped NewReadTenants is checked again here, so an
// invalid scope never reaches a store.
func tenantContext(ctx context.Context, scope string) (context.Context, error) {
	n, err := ParseTenant(scope)
	if err != nil {
		return nil, err
	}
	return WithTenant(ctx, n), nil
}

// ReadOne reads scope through read, under a context scoped to it, provided
// scope is one of tenants. A tenant outside the set is refused with
// ErrInvalid before any store is called.
func ReadOne[T any](ctx context.Context, tenants ReadTenants, scope string, read func(ctx context.Context, scope string) (T, error)) (Tagged[T], error) {
	if !tenants.Contains(scope) {
		return Tagged[T]{}, fmt.Errorf("%w: tenant %q is not one of this request's read tenants", ErrInvalid, scope)
	}
	tctx, err := tenantContext(ctx, scope)
	if err != nil {
		return Tagged[T]{}, err
	}
	item, err := read(tctx, scope)
	if err != nil {
		return Tagged[T]{}, err
	}
	return Tagged[T]{Tenant: scope, Item: item}, nil
}

// StreamOne streams scope through read, under a context scoped to it, provided
// scope is one of tenants; a tenant outside the set ends the stream with
// ErrInvalid before any store is called.
func StreamOne[T any](ctx context.Context, tenants ReadTenants, scope string, read func(ctx context.Context, scope string) iter.Seq2[T, error]) iter.Seq2[Tagged[T], error] {
	if !tenants.Contains(scope) {
		return func(yield func(Tagged[T], error) bool) {
			yield(Tagged[T]{}, fmt.Errorf("%w: tenant %q is not one of this request's read tenants", ErrInvalid, scope))
		}
	}
	return StreamAcross(ctx, ReadTenants{scope}, read)
}

// ReadAcross calls read once per tenant of tenants, in order, and tags each
// answer. The first error stops the read and is returned alone: a partial
// answer would pass for the whole set.
func ReadAcross[T any](ctx context.Context, tenants ReadTenants, read func(ctx context.Context, scope string) (T, error)) ([]Tagged[T], error) {
	out := make([]Tagged[T], 0, len(tenants))
	for _, scope := range tenants {
		tctx, err := tenantContext(ctx, scope)
		if err != nil {
			return nil, err
		}
		item, err := read(tctx, scope)
		if err != nil {
			return nil, err
		}
		out = append(out, Tagged[T]{Tenant: scope, Item: item})
	}
	return out, nil
}

// StreamAcross chains read's streams, one per tenant of tenants, in order,
// tagging every item. Bounds a caller passes to read (limit, offset) apply to
// each tenant's stream: the result is the tenants' pages end to end, not one
// page cut from their merge. An error ends the stream; it comes after the
// items of the tenants already read, so a streaming caller must treat the
// stream as failed, not as those tenants' complete answer.
func StreamAcross[T any](ctx context.Context, tenants ReadTenants, read func(ctx context.Context, scope string) iter.Seq2[T, error]) iter.Seq2[Tagged[T], error] {
	return func(yield func(Tagged[T], error) bool) {
		for _, scope := range tenants {
			tctx, err := tenantContext(ctx, scope)
			if err != nil {
				yield(Tagged[T]{}, err)
				return
			}
			for item, err := range read(tctx, scope) {
				if err != nil {
					yield(Tagged[T]{}, err)
					return
				}
				if !yield(Tagged[T]{Tenant: scope, Item: item}, nil) {
					return
				}
			}
		}
	}
}
