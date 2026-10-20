package rest

import (
	"context"
	"errors"
	"testing"
)

func TestOperationGuardsComposeAndPreserveOuterCauses(t *testing.T) {
	outer, inner := errors.New("outer"), errors.New("inner")
	calls := 0
	ctx := WithOperationGuard(context.Background(), func(context.Context) error { calls++; return outer })
	ctx = WithOperationGuard(ctx, func(context.Context) error { calls++; return inner })
	ctx = WithOperationGuard(ctx, nil)
	if err := CheckOperationGuard(ctx); !errors.Is(err, outer) || !errors.Is(err, inner) || calls != 2 {
		t.Fatal(err, calls)
	}
}

func TestNestedOperationSourcesRetainAncestorsAndIsolateChildBindings(t *testing.T) {
	outer, inner := errors.New("outer binding"), errors.New("inner binding")
	parent := WithOperationSources(context.Background())
	RegisterOperationSource(parent, func(context.Context) error { return outer })
	child := WithOperationSources(parent)
	RegisterOperationSource(child, func(context.Context) error { return inner })
	if err := CheckOperationGuard(child); !errors.Is(err, outer) || !errors.Is(err, inner) {
		t.Fatal(err)
	}
	if err := CheckOperationGuard(parent); !errors.Is(err, outer) || errors.Is(err, inner) {
		t.Fatal(err)
	}
}
