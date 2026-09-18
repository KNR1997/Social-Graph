package httpx

import (
	"context"

	"github.com/google/uuid"
)

// ownerKey is an unexported context key, so the only way to put an owner on a
// context is WithOwner, and no client input can reach it.
type ownerKey struct{}

// WithOwner marks a request as belonging to an authenticated user.
//
// This lives in httpx rather than in rest/auth because every resource package
// needs the owner and none of them may name the identity aggregate: a REST
// package imports exactly one domain package, and both domain packages are
// called entity, so rest/person cannot refer to entity.User even if it wanted
// to. A uuid is transport-shaped, which is what makes it shareable here.
//
// The session middleware is the only production caller. Anything else setting
// an owner is claiming a request is authenticated when it is not.
func WithOwner(ctx context.Context, owner uuid.UUID) context.Context {
	return context.WithValue(ctx, ownerKey{}, owner)
}

// OwnerFrom returns the authenticated owner of the request. The second result
// is false on any request that did not pass through the session middleware.
func OwnerFrom(ctx context.Context) (uuid.UUID, bool) {
	owner, ok := ctx.Value(ownerKey{}).(uuid.UUID)

	return owner, ok
}
