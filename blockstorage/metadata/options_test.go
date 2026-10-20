package metadata

import (
	"errors"
	"sync"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestHeaderOptionsSnapshotCanonicalLastWinsAndIndependentReuse(t *testing.T) {
	input := map[string]string{"x-request-id": "original", "X-Trace": "trace"}
	bulk := WithHeaders(input)
	input["x-request-id"] = "mutated"
	delete(input, "X-Trace")
	var wait sync.WaitGroup
	for i := 0; i < 12; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			cfg, err := request.Apply(Opts{}, bulk, WithHeader("X-Request-ID", "last"))
			if err != nil || cfg.Headers["X-Request-Id"] != "last" || cfg.Headers["X-Trace"] != "trace" || len(cfg.Headers) != 2 {
				t.Errorf("headers=%v err=%v", cfg.Headers, err)
			}
			cfg.Headers["X-Trace"] = "owned iteration"
		}()
	}
	wait.Wait()
	first, err := request.Apply(Opts{}, bulk)
	if err != nil || first.Headers["X-Request-Id"] != "original" || first.Headers["X-Trace"] != "trace" {
		t.Fatalf("headers=%v err=%v", first.Headers, err)
	}
	first.Headers["X-Trace"] = "changed"
	second, err := request.Apply(Opts{}, bulk)
	if err != nil || second.Headers["X-Trace"] != "trace" {
		t.Fatalf("reused headers=%v err=%v", second.Headers, err)
	}
}

func TestHeaderOptionsRejectBulkConflictsAndReinitializeNilConfigMap(t *testing.T) {
	for _, headers := range []map[string]string{
		{"X-Trace": "first", "x-trace": "second"},
		{"bad:name": "value"}, {"X-Trace": "line\r\nbreak"}, {"": "value"},
	} {
		if _, err := request.Apply(Opts{}, WithHeaders(headers)); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("headers=%v err=%v", headers, err)
		}
	}
	clear := func(cfg *request.Config[Opts]) error { cfg.Headers = nil; return nil }
	cfg, err := request.Apply(Opts{}, clear, WithHeaders(map[string]string{"X-Trace": "equal", "x-trace": "equal"}), WithHeader("X-Other", "last"))
	if err != nil || cfg.Headers["X-Trace"] != "equal" || cfg.Headers["X-Other"] != "last" || len(cfg.Headers) != 2 {
		t.Fatalf("headers=%v err=%v", cfg.Headers, err)
	}
}
