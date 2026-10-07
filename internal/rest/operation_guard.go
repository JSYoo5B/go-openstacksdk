package rest

import "context"

type operationGuardKey struct{}

// WithOperationGuard carries a library-owned outer workflow invariant into
// another service's guarded requests. The service still owns its own guard.
func WithOperationGuard(ctx context.Context, guard func(context.Context) error) context.Context {
	return context.WithValue(ctx, operationGuardKey{}, guard)
}

func CheckOperationGuard(ctx context.Context) error {
	if guard, ok := ctx.Value(operationGuardKey{}).(func(context.Context) error); ok && guard != nil {
		return guard(ctx)
	}
	return nil
}
