package blockstorage_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestDeleteVolumeOptionsEffectiveDefaultsReplacementAndLastWins(t *testing.T) {
	defaults, err := blockstorage.PrepareDeleteVolumeOptions(context.Background())
	if err != nil || defaults.Wait == nil || !*defaults.Wait || defaults.Force || defaults.WaitPolicy.Timeout == nil || *defaults.WaitPolicy.Timeout != 0 || defaults.WaitPolicy.PollInterval == nil || *defaults.WaitPolicy.PollInterval != 2*time.Second || defaults.WaitPolicy.ProgressCallback != nil {
		t.Fatalf("defaults=%+v error=%v", defaults, err)
	}
	value, err := blockstorage.PrepareDeleteVolumeOptions(context.Background(), blockstorage.WithDeleteVolumeForce(true), blockstorage.WithDeleteVolumeWait(false), blockstorage.WithDeleteVolumeOptions(blockstorage.DeleteVolumeOpts{}))
	if err != nil || value.Wait == nil || !*value.Wait || value.Force {
		t.Fatalf("complete replacement=%+v error=%v", value, err)
	}
	value, err = blockstorage.PrepareDeleteVolumeOptions(context.Background(), blockstorage.WithDeleteVolumeWait(true), blockstorage.WithDeleteVolumeWait(false), blockstorage.WithDeleteVolumeForce(true), blockstorage.WithDeleteVolumeForce(false))
	if err != nil || *value.Wait || value.Force {
		t.Fatalf("last options=%+v error=%v", value, err)
	}
	timeout, interval := time.Hour, time.Millisecond
	value, err = blockstorage.PrepareDeleteVolumeOptions(context.Background(),
		blockstorage.WithDeleteVolumeWait(false), blockstorage.WithDeleteVolumeForce(true),
		blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{Timeout: &timeout, PollInterval: &interval, ProgressCallback: func(int) error { t.Fatal("replaced callback ran"); return nil }}),
		blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{}))
	if err != nil || value.Wait == nil || *value.Wait || !value.Force || value.WaitPolicy.Timeout == nil || *value.WaitPolicy.Timeout != 0 || value.WaitPolicy.PollInterval == nil || *value.WaitPolicy.PollInterval != 2*time.Second || value.WaitPolicy.ProgressCallback != nil {
		t.Fatalf("empty wait subtree replacement=%+v error=%v", value, err)
	}
}

func TestDeleteVolumeOptionsOwnFactoryPointersAndConcurrentReusablePolicies(t *testing.T) {
	disabled, timeout, interval := false, time.Duration(0), time.Millisecond
	option := blockstorage.WithDeleteVolumeOptions(blockstorage.DeleteVolumeOpts{Wait: &disabled, Force: true, WaitPolicy: blockstorage.DeleteVolumeWaitOpts{Timeout: &timeout, PollInterval: &interval}})
	policyOption := blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{Timeout: &timeout, PollInterval: &interval})
	disabled, timeout, interval = true, time.Hour, time.Hour
	var wait sync.WaitGroup
	for range 8 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			value, err := blockstorage.PrepareDeleteVolumeOptions(context.Background(), option, policyOption)
			if err != nil || value.Wait == nil || *value.Wait || !value.Force || *value.WaitPolicy.Timeout != 0 || *value.WaitPolicy.PollInterval != time.Millisecond {
				t.Errorf("owned policy=%+v error=%v", value, err)
				return
			}
			*value.Wait = true
			*value.WaitPolicy.Timeout = time.Hour
			*value.WaitPolicy.PollInterval = time.Hour
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
		v.WaitPolicy.Timeout = &timeout
		retained = v
		return nil
	}, func(v *blockstorage.DeleteVolumeOpts) error {
		calls++
		*retained.Wait = true
		*retained.WaitPolicy.Timeout = time.Hour
		retained.Force = false
		if *v.Wait || !v.Force || *v.WaitPolicy.Timeout != time.Millisecond {
			t.Error("retained callback changed next owned config", v)
		}
		return nil
	})
	if err != nil || calls != 2 || *value.Wait || !value.Force || *value.WaitPolicy.Timeout != time.Millisecond {
		t.Fatalf("prepared=%+v error=%v calls=%d", value, err, calls)
	}
	*retained.Wait = true
	*retained.WaitPolicy.Timeout = time.Hour
	if *value.Wait || *value.WaitPolicy.Timeout != time.Millisecond {
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
	for _, option := range []blockstorage.DeleteVolumeOption{nil, blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{Timeout: &negative}), blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{PollInterval: &zero}), blockstorage.WithDeleteVolumeWaitPolicy(blockstorage.DeleteVolumeWaitOpts{PollInterval: &negative})} {
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
