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
	valid := resource.WithPollInterval(time.Millisecond)
	server := &compute.Server{ID: "server"}
	for name, call := range map[string]func() error{
		"backup": func() error {
			_, err := service.BackupServer(ctx, "server", compute.BackupServerRequest{Name: "b"}, compute.WithBackupServerWait(true), compute.WithBackupServerWaitOptions(resource.WithWaitStrategy(invalid)))
			return err
		},
		"backup failure states": func() error {
			_, err := service.BackupServer(ctx, "server", compute.BackupServerRequest{Name: "b"}, func(o *compute.BackupServerOpts) error {
				o.Wait, o.WaitOptions = true, []resource.WaitOption{valid, resource.WithFailureStates("error")}
				return nil
			})
			return err
		},
		"floating IP create status attribute": func() error {
			_, err := service.CreateFloatingIP(ctx, compute.CreateFloatingIPRequest{}, compute.WithFloatingIPCreateWaitOptions(resource.WithStatusAttribute("status")))
			return err
		},
		"add auto IP progress": func() error {
			_, err := service.AddAutoIP(ctx, server, compute.WithAddAutoIPWaitOptions(resource.WithProgress(make(chan int))))
			return err
		},
		"pool IP failure states": func() error {
			_, err := service.AddIPFromPool(ctx, server, resource.ID("pool"), compute.WithPoolIPWaitOptions(resource.WithFailureStates()))
			return err
		},
		"automatic IP progress": func() error {
			_, err := service.EnsureServerFloatingIP(ctx, compute.AutomaticFloatingIPRequest{Server: server}, compute.WithAutomaticIPWait(resource.WithProgressCallback(func(int) error { return nil })))
			return err
		},
		"floating IP create": func() error {
			_, err := service.CreateFloatingIP(ctx, compute.CreateFloatingIPRequest{}, compute.WithFloatingIPCreateWaitOptions(resource.WithWaitStrategy(invalid)))
			return err
		},
		"add auto IP": func() error {
			_, err := service.AddAutoIP(ctx, server, compute.WithAddAutoIPWaitOptions(resource.WithWaitStrategy(invalid)))
			return err
		},
		"pool IP": func() error {
			_, err := service.AddIPFromPool(ctx, server, resource.ID("pool"), compute.WithPoolIPWaitOptions(resource.WithWaitStrategy(resource.WaitStrategy{})))
			return err
		},
		"automatic IP": func() error {
			_, err := service.EnsureServerFloatingIP(ctx, compute.AutomaticFloatingIPRequest{Server: server}, compute.WithAutomaticIPWait(resource.WithWaitStrategy(invalid)))
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
	growing := resource.WithWaitStrategy(resource.ExponentialWait(ms, 2, 8*ms))
	variadic := []resource.WaitOption{resource.WithPollInterval(5 * ms)}
	appended := compute.WithFloatingIPCreateWaitOptions(variadic...)
	variadic[0] = nil
	p, err := compute.PrepareFloatingIPCreateOptions(context.Background(), appended, compute.WithFloatingIPCreateWaitOptions(growing))
	if err != nil || len(p.WaitOptions) != 2 || p.WaitOptions[0] == nil {
		t.Fatal(p, err)
	}
	field := []resource.WaitOption{resource.WithTimeout(time.Second)}
	option := compute.WithFloatingIPCreateOptions(compute.FloatingIPCreateOpts{WaitOptions: field})
	field[0] = resource.WithTimeout(0)
	p, err = compute.PrepareFloatingIPCreateOptions(context.Background(), option)
	if err != nil || len(p.WaitOptions) != 1 {
		t.Fatal(p, err)
	}
	p.WaitOptions[0] = nil
	if p, err = compute.PrepareFloatingIPCreateOptions(context.Background(), option); err != nil || len(p.WaitOptions) != 1 || p.WaitOptions[0] == nil {
		t.Fatal(p, err)
	}
	if _, err := compute.PrepareFloatingIPCreateOptions(context.Background(), compute.WithFloatingIPCreateOptions(compute.FloatingIPCreateOpts{WaitOptions: []resource.WaitOption{nil}})); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
