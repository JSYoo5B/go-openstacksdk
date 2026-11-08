package compute_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/compute"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestComputeWaitStrategyOptionsRejectedBeforeHTTP(t *testing.T) {
	cloud := testcloud.New(t)
	calls := testcloud.NewRecorder(func(*http.Request) (*http.Response, error) {
		return testcloud.Response(500, `{}`, nil), nil
	})
	cloud.SetTransport(calls)
	service := compute.New(cloud.Client("compute", "/nova/v2.1/"), compute.Dependencies{})
	ctx := context.Background()
	invalid := resource.ExponentialWait(time.Millisecond, 0, 0)
	valid := resource.FixedWait(time.Millisecond)
	server := &compute.Server{ID: "server"}
	for name, call := range map[string]func() error{
		"backup": func() error {
			_, err := service.BackupServer(ctx, "server", compute.BackupServerRequest{Name: "b"}, compute.WithBackupServerWait(true), compute.WithBackupServerWaitStrategy(invalid))
			return err
		},
		"backup both": func() error {
			interval := time.Millisecond
			_, err := service.BackupServer(ctx, "server", compute.BackupServerRequest{Name: "b"}, func(o *compute.BackupServerOpts) error {
				o.Wait, o.PollInterval, o.WaitStrategy = true, &interval, &valid
				return nil
			})
			return err
		},
		"floating IP create": func() error {
			_, err := service.CreateFloatingIP(ctx, compute.CreateFloatingIPRequest{}, compute.WithFloatingIPCreateWaitStrategy(invalid))
			return err
		},
		"add auto IP": func() error {
			_, err := service.AddAutoIP(ctx, server, compute.WithAddAutoIPWaitStrategy(invalid))
			return err
		},
		"pool IP": func() error {
			_, err := service.AddIPFromPool(ctx, server, resource.ID("pool"), compute.WithPoolIPWaitStrategy(resource.WaitStrategy{}))
			return err
		},
		"automatic IP": func() error {
			_, err := service.EnsureServerFloatingIP(ctx, compute.AutomaticFloatingIPRequest{Server: server}, compute.WithAutomaticIPWaitStrategy(invalid))
			return err
		},
	} {
		if err := call(); !errors.Is(err, resource.ErrInvalidOption) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if calls.Len() != 0 {
		t.Fatal("HTTP before validation", calls.Calls())
	}
}

func TestFloatingIPCreateWaitStrategyOptions(t *testing.T) {
	ms := time.Millisecond
	growing := resource.ExponentialWait(ms, 2, 8*ms)
	p, err := compute.PrepareFloatingIPCreateOptions(context.Background(), compute.WithFloatingIPCreateWaitInterval(5*ms), compute.WithFloatingIPCreateWaitStrategy(growing))
	if err != nil || p.WaitInterval != 0 || p.WaitStrategy == nil || *p.WaitStrategy != growing {
		t.Fatal(p, err)
	}
	p, err = compute.PrepareFloatingIPCreateOptions(context.Background(), compute.WithFloatingIPCreateWaitStrategy(growing), compute.WithFloatingIPCreateWaitInterval(5*ms))
	if err != nil || p.WaitInterval != 5*ms || p.WaitStrategy != nil {
		t.Fatal(p, err)
	}
	if _, err := compute.PrepareFloatingIPCreateOptions(context.Background(), compute.WithFloatingIPCreateOptions(compute.FloatingIPCreateOpts{WaitInterval: ms, WaitStrategy: &growing})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
