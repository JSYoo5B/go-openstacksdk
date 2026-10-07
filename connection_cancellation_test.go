package gophercloudsdk_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
)

func TestCachedManualServicesRespectCanceledContext(t *testing.T) {
	conn := connection(t, testcloud.New(t))
	services := map[string]func(context.Context) error{
		"Compute":      func(ctx context.Context) error { _, err := conn.Compute(ctx); return err },
		"Network":      func(ctx context.Context) error { _, err := conn.Network(ctx); return err },
		"Image":        func(ctx context.Context) error { _, err := conn.Image(ctx); return err },
		"BlockStorage": func(ctx context.Context) error { _, err := conn.BlockStorage(ctx); return err },
	}
	for name, load := range services {
		t.Run(name, func(t *testing.T) {
			if err := load(context.Background()); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := load(ctx); !errors.Is(err, context.Canceled) {
				t.Fatalf("err=%v", err)
			}
		})
	}
}
