package blockstorage_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestDeleteVolumeOptionsEffectiveDefaultsReplacementAndLastWins(t *testing.T) {
	defaults, err := blockstorage.PrepareDeleteVolumeOptions(context.Background())
	if err != nil || defaults.Wait == nil || !*defaults.Wait || defaults.Force || !preparedWait(t, defaults.WaitOptions).Unlimited || preparedWait(t, defaults.WaitOptions).Strategy.Delay(1) != 2*time.Second || preparedWait(t, defaults.WaitOptions).ProgressCallback != nil {
		t.Fatalf("defaults=%+v error=%v", defaults, err)
	}
	value, err := blockstorage.PrepareDeleteVolumeOptions(context.Background(), func(c *blockstorage.DeleteVolumeOpts) error { c.Force = true; return nil }, blockstorage.WithDeleteVolumeWait(false), blockstorage.WithDeleteVolumeOptions(blockstorage.DeleteVolumeOpts{}))
	if err != nil || value.Wait == nil || !*value.Wait || value.Force {
		t.Fatalf("complete replacement=%+v error=%v", value, err)
	}
	value, err = blockstorage.PrepareDeleteVolumeOptions(context.Background(), blockstorage.WithDeleteVolumeWait(true), blockstorage.WithDeleteVolumeWait(false), func(c *blockstorage.DeleteVolumeOpts) error { c.Force = true; return nil }, func(c *blockstorage.DeleteVolumeOpts) error { c.Force = false; return nil })
	if err != nil || *value.Wait || value.Force {
		t.Fatalf("last options=%+v error=%v", value, err)
	}
	timeout, interval := time.Hour, time.Millisecond
	value, err = blockstorage.PrepareDeleteVolumeOptions(context.Background(),
		blockstorage.WithDeleteVolumeWaitOptions(resource.WithTimeout(timeout), resource.WithPollInterval(interval), resource.WithProgressCallback(func(int) error { t.Fatal("replaced callback ran"); return nil })),
		blockstorage.WithDeleteVolumeOptions(blockstorage.DeleteVolumeOpts{}),
		blockstorage.WithDeleteVolumeWait(false), func(c *blockstorage.DeleteVolumeOpts) error { c.Force = true; return nil },
		blockstorage.WithDeleteVolumeWaitOptions())
	if err != nil || value.Wait == nil || *value.Wait || !value.Force || !preparedWait(t, value.WaitOptions).Unlimited || preparedWait(t, value.WaitOptions).Strategy.Delay(1) != 2*time.Second || preparedWait(t, value.WaitOptions).ProgressCallback != nil {
		t.Fatalf("empty wait subtree replacement=%+v error=%v", value, err)
	}
}

func TestDeleteVolumeOptionsOwnFactoryPointersAndConcurrentReusablePolicies(t *testing.T) {
	disabled := false
	waitOptions := []resource.WaitOption{resource.WithUnlimitedWait(), resource.WithPollInterval(time.Millisecond)}
	option := blockstorage.WithDeleteVolumeOptions(blockstorage.DeleteVolumeOpts{Wait: &disabled, Force: true, WaitOptions: waitOptions})
	policyOption := blockstorage.WithDeleteVolumeWaitOptions(waitOptions...)
	disabled = true
	waitOptions[0], waitOptions[1] = resource.WithTimeout(time.Hour), resource.WithPollInterval(time.Hour)
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := blockstorage.PrepareDeleteVolumeOptions(context.Background(), option, policyOption)
			if err != nil || value.Wait == nil || *value.Wait || !value.Force || !preparedWait(t, value.WaitOptions).Unlimited || preparedWait(t, value.WaitOptions).Strategy.Delay(1) != time.Millisecond {
				t.Errorf("owned policy=%+v error=%v", value, err)
				return
			}
			*value.Wait = true
			value.WaitOptions[0] = resource.WithTimeout(time.Hour)
		}()
	}
	wait.Wait()
}

func TestDeleteVolumeOptionsCallbacksRunOnceAndRetainedHandlesCannotRewritePreparedState(t *testing.T) {
	var retained *blockstorage.DeleteVolumeOpts
	calls := 0
	disabled, timeout := false, time.Millisecond
	value, err := blockstorage.PrepareDeleteVolumeOptions(context.Background(), func(v *blockstorage.DeleteVolumeOpts) error {
		calls++
		v.Wait = &disabled
		v.Force = true
		v.WaitOptions = []resource.WaitOption{resource.WithTimeout(timeout)}
		retained = v
		return nil
	}, func(v *blockstorage.DeleteVolumeOpts) error {
		calls++
		*retained.Wait = true
		retained.WaitOptions[0] = resource.WithTimeout(time.Hour)
		retained.Force = false
		if *v.Wait || !v.Force || preparedWait(t, v.WaitOptions).Timeout != time.Millisecond {
			t.Error("retained callback changed next owned config", v)
		}
		return nil
	})
	if err != nil || calls != 2 || *value.Wait || !value.Force || preparedWait(t, value.WaitOptions).Timeout != time.Millisecond {
		t.Fatalf("prepared=%+v error=%v calls=%d", value, err, calls)
	}
	*retained.Wait = true
	retained.WaitOptions[0] = resource.WithTimeout(time.Hour)
	if *value.Wait || preparedWait(t, value.WaitOptions).Timeout != time.Millisecond {
		t.Fatal("returned policy aliases callback handle", value)
	}
	cause := errors.New("delete option failed")
	calls = 0
	_, err = blockstorage.PrepareDeleteVolumeOptions(context.Background(), func(*blockstorage.DeleteVolumeOpts) error { return cause }, func(*blockstorage.DeleteVolumeOpts) error { calls++; return nil })
	if !errors.Is(err, cause) || calls != 0 {
		t.Fatal(err, calls)
	}
}

func TestDeleteVolumeOptionsValidateInactivePolicyAndPreserveCancellationCause(t *testing.T) {
	negative, zero := -time.Second, time.Duration(0)
	for _, option := range []blockstorage.DeleteVolumeOption{nil, blockstorage.WithDeleteVolumeWaitOptions(resource.WithTimeout(negative)), blockstorage.WithDeleteVolumeWaitOptions(resource.WithPollInterval(zero)), blockstorage.WithDeleteVolumeWaitOptions(resource.WithPollInterval(negative))} {
		_, err := blockstorage.PrepareDeleteVolumeOptions(context.Background(), blockstorage.WithDeleteVolumeWait(false), option)
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatalf("inactive policy accepted: %v", err)
		}
	}
	calls := 0
	_, err := blockstorage.PrepareDeleteVolumeOptions(nil, func(*blockstorage.DeleteVolumeOpts) error { calls++; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal(err, calls)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("canceled delete preparation")
	_, err = blockstorage.PrepareDeleteVolumeOptions(ctx, func(*blockstorage.DeleteVolumeOpts) error { calls++; cancel(cause); return nil }, func(*blockstorage.DeleteVolumeOpts) error { calls++; return nil })
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || calls != 1 {
		t.Fatal(err, calls)
	}
}
