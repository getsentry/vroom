package sentryutil

import (
	"context"

	"github.com/getsentry/sentry-go"
)

// EnsureScope returns the [*sentry.Scope] on ctx if present, or a new scope via
// [sentry.WithScope] if not.
func EnsureScope(ctx context.Context) (context.Context, *sentry.Scope) {
	if s := sentry.ScopeFromContext(ctx); s != nil {
		return ctx, s
	}
	return sentry.WithScope(ctx)
}
