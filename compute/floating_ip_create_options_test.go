package compute_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestFloatingIPCreateOptionsDefaultsBulkAndIgnoredSelectors(t *testing.T) {
	p, err := compute.PrepareFloatingIPCreateOptions(context.Background())
	if err != nil || p.Wait || p.WaitTimeout != time.Minute || p.WaitOptions != nil || p.Timeout != 0 || p.Source != nil {
		t.Fatal(p, err)
	}
	p, err = compute.PrepareFloatingIPCreateOptions(context.Background(), compute.WithFloatingIPCreateWaitTimeout(-time.Second), func(c *compute.FloatingIPCreateOpts) error { c.Server = resource.ID("ignored/path"); return nil }, func(c *compute.FloatingIPCreateOpts) error { c.FixedAddress = "literal IPv6 or arbitrary"; return nil })
	if err != nil || p.WaitTimeout != -time.Second || p.Server.String() != "ignored/path" {
		t.Fatal(p, err)
	}
	p, err = compute.PrepareFloatingIPCreateOptions(context.Background(), compute.WithFloatingIPCreateTimeout(time.Minute), compute.WithFloatingIPCreateOptions(compute.FloatingIPCreateOpts{}), compute.WithFloatingIPCreateWait(true))
	if err != nil || !p.Wait || p.WaitTimeout != 0 || p.Timeout != 0 {
		t.Fatal(p, err)
	}
}
func TestFloatingIPCreateOptionsOwnSnapshotsAndApplyOnce(t *testing.T) {
	source := compute.FloatingIPNeutron
	location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"owner"`)}}
	option := compute.WithFloatingIPCreateOptions(compute.FloatingIPCreateOpts{Source: &source, Location: &location, WaitTimeout: time.Minute})
	source = compute.FloatingIPNova
	location.Project.ID[1] = 'X'
	calls := 0
	p, err := compute.PrepareFloatingIPCreateOptions(context.Background(), option, func(o *compute.FloatingIPCreateOpts) error {
		calls++
		if *o.Source != compute.FloatingIPNeutron || string(o.Location.Project.ID) != `"owner"` {
			t.Error(o)
		}
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatal(p, err, calls)
	}
	*p.Source = compute.FloatingIPNone
	p.Location.Project.ID[1] = 'Y'
	p, err = compute.PrepareFloatingIPCreateOptions(context.Background(), option)
	if err != nil || *p.Source != compute.FloatingIPNeutron || string(p.Location.Project.ID) != `"owner"` {
		t.Fatal(p, err)
	}
}
func TestFloatingIPCreateOptionsPreflightAndCancellation(t *testing.T) {
	for _, option := range []compute.FloatingIPCreateOption{nil, func(c *compute.FloatingIPCreateOpts) error {
		v := compute.FloatingIPSource("invalid")
		c.Source = &v
		return nil
	}, compute.WithFloatingIPCreateTimeout(0), compute.WithFloatingIPCreateWaitOptions(resource.WithPollInterval(0)), compute.WithFloatingIPCreateOptions(compute.FloatingIPCreateOpts{WaitOptions: []resource.WaitOption{resource.WithPollInterval(-1)}})} {
		if _, err := compute.PrepareFloatingIPCreateOptions(context.Background(), option); !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	cause := errors.New("stop preparation")
	ctx, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	calls := 0
	_, err := compute.PrepareFloatingIPCreateOptions(ctx, func(*compute.FloatingIPCreateOpts) error { calls++; return nil })
	if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal(err, calls)
	}
}
