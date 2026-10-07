package compute

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"testing"
	"time"
)

func TestFloatingIPDeleteOptionsDefaultsRetriesAndBulkReplacement(t *testing.T) {
	opts, err := PrepareFloatingIPDeleteOptions(context.Background())
	if err != nil || opts.Retries != 1 || opts.Timeout != 0 || opts.DirectGet {
		t.Fatal(opts, err)
	}
	opts, err = PrepareFloatingIPDeleteOptions(context.Background(), WithFloatingIPDeleteRetries(-2))
	if err != nil || opts.Retries != -2 {
		t.Fatal(opts, err)
	}
	opts, err = PrepareFloatingIPDeleteOptions(context.Background(), WithFloatingIPDeleteRetries(3), WithFloatingIPDeleteOptions(FloatingIPDeleteOpts{}))
	if err != nil || opts.Retries != 0 {
		t.Fatal(opts, err)
	}
	opts, err = PrepareFloatingIPDeleteOptions(context.Background(), WithFloatingIPDeleteTimeout(time.Minute), WithUnlimitedFloatingIPDeleteTimeout())
	if err != nil || opts.Timeout != 0 {
		t.Fatal(opts, err)
	}
}
func TestFloatingIPDeleteOptionsOwnSourceAndLocation(t *testing.T) {
	source := FloatingIPNeutron
	name := "owned"
	id := json.RawMessage(`"scope"`)
	input := FloatingIPDeleteOpts{Source: &source, Location: &resource.CloudLocation{Cloud: &name, Project: resource.CloudProject{ID: id}}, Retries: 2}
	option := WithFloatingIPDeleteOptions(input)
	source = FloatingIPNova
	name = "changed"
	id[1] = 'X'
	first, err := PrepareFloatingIPDeleteOptions(context.Background(), option)
	if err != nil || *first.Source != FloatingIPNeutron || *first.Location.Cloud != "owned" || string(first.Location.Project.ID) != `"scope"` {
		t.Fatal(first, err)
	}
	*first.Source = FloatingIPNone
	*first.Location.Cloud = "returned"
	first.Location.Project.ID[1] = 'Y'
	second, err := PrepareFloatingIPDeleteOptions(context.Background(), option)
	if err != nil || *second.Source != FloatingIPNeutron || *second.Location.Cloud != "owned" || string(second.Location.Project.ID) != `"scope"` {
		t.Fatal(second, err)
	}
	calls := 0
	_, err = PrepareFloatingIPDeleteOptions(context.Background(), func(o *FloatingIPDeleteOpts) error { calls++; o.Location = input.Location; return nil }, func(o *FloatingIPDeleteOpts) error {
		*input.Location.Cloud = "outside"
		if *o.Location.Cloud == "outside" {
			t.Fatal("callback alias")
		}
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
}
func TestFloatingIPDeleteOptionsPreflightAndCancellationCause(t *testing.T) {
	for _, option := range []FloatingIPDeleteOption{nil, WithFloatingIPDeleteSource("bad"), WithFloatingIPDeleteTimeout(0), WithFloatingIPDeleteOptions(FloatingIPDeleteOpts{Timeout: -1})} {
		if _, err := PrepareFloatingIPDeleteOptions(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("delete cancelled")
	cancel(cause)
	calls := 0
	_, err := PrepareFloatingIPDeleteOptions(ctx, func(*FloatingIPDeleteOpts) error { calls++; return nil })
	if !errors.Is(err, cause) || calls != 0 {
		t.Fatal(err, calls)
	}
	if _, err := PrepareFloatingIPDeleteOptions(nil); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
