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
