package rest

import (
	"context"
	"errors"
	"sync"
)

type operationGuardKey struct{}
type operationSourcesKey struct{}
type operationSources struct {
	mu     sync.Mutex
	guards []func(context.Context) error
}

// WithOperationSources retains lazily bound, source-only dependency invariants
// through the entire compound workflow. Registered guards must not invoke the
// outer operation guard or another service's full workflow check.
func WithOperationSources(ctx context.Context) context.Context {
	sources := &operationSources{}
	ctx = context.WithValue(ctx, operationSourcesKey{}, sources)
	return WithOperationGuard(ctx, sources.check)
}

func RegisterOperationSource(ctx context.Context, guard func(context.Context) error) {
	if sources, ok := ctx.Value(operationSourcesKey{}).(*operationSources); ok && guard != nil {
		sources.mu.Lock()
		sources.guards = append(sources.guards, guard)
		sources.mu.Unlock()
	}
}

func (sources *operationSources) check(ctx context.Context) error {
	sources.mu.Lock()
	guards := append([]func(context.Context) error(nil), sources.guards...)
	sources.mu.Unlock()
	var causes []error
	for _, guard := range guards {
		causes = append(causes, guard(ctx))
	}
	return errors.Join(causes...)
}

func OperationGuard(ctx context.Context) func(context.Context) error {
	guard, _ := ctx.Value(operationGuardKey{}).(func(context.Context) error)
	return guard
}

func HasOperationGuard(ctx context.Context) bool {
	guard, _ := ctx.Value(operationGuardKey{}).(func(context.Context) error)
	return guard != nil
}

// WithOperationGuard carries a library-owned outer workflow invariant into
// another service's guarded requests. The service still owns its own guard.
func WithOperationGuard(ctx context.Context, guard func(context.Context) error) context.Context {
	previous, _ := ctx.Value(operationGuardKey{}).(func(context.Context) error)
	if previous != nil && guard != nil {
		next := guard
		guard = func(ctx context.Context) error { return errors.Join(previous(ctx), next(ctx)) }
	} else if guard == nil {
		guard = previous
	}
	return context.WithValue(ctx, operationGuardKey{}, guard)
}

func CheckOperationGuard(ctx context.Context) error {
	if guard, ok := ctx.Value(operationGuardKey{}).(func(context.Context) error); ok && guard != nil {
		return guard(ctx)
	}
	return nil
}
