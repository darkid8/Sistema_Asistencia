package http

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"workshop/internal/domain"
)

type contextKey string

const callerContextKey contextKey = "caller"

// caller is the authenticated identity of the current request.
type caller struct {
	UserID string
	Role   domain.Role
}

// authMiddleware rejects a request without a valid session token and puts the
// caller identity in the context for the handlers to read. The token is read
// from the HttpOnly session cookie first, with the Authorization header kept
// as a fallback for non-browser callers.
func authMiddleware(issuer TokenIssuer, now func() time.Time) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			token, ok := tokenFromRequest(request)
			if !ok {
				failure(writer, domain.ErrUnauthorized)
				return
			}
			claim, err := issuer.Verify(token, now())
			if err != nil {
				failure(writer, domain.ErrUnauthorized)
				return
			}
			identity := caller{UserID: claim.UserID, Role: domain.Role(claim.Role)}
			next.ServeHTTP(writer, request.WithContext(
				context.WithValue(request.Context(), callerContextKey, identity),
			))
		})
	}
}

// callerFrom reads the authenticated identity a handler runs under.
func callerFrom(ctx context.Context) (caller, error) {
	identity, ok := ctx.Value(callerContextKey).(caller)
	if !ok || identity.UserID == "" {
		return caller{}, domain.ErrUnauthorized
	}
	return identity, nil
}

// requireAdministrator refuses a caller who is not the workshop manager.
func requireAdministrator(ctx context.Context) (caller, error) {
	identity, err := callerFrom(ctx)
	if err != nil {
		return caller{}, err
	}
	if identity.Role != domain.RoleAdministrator {
		return caller{}, fmt.Errorf("%w: this operation belongs to the administrator", domain.ErrForbidden)
	}
	return identity, nil
}
