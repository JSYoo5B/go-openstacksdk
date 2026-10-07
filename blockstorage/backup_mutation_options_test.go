package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/resource"
)

func TestBackupMutationOptionsPreserveOmissionAndExplicitEmptyFalseZero(t *testing.T) {
	ctx := context.Background()
	empty, err := blockstorage.PrepareCreateVolumeBackupOptions(ctx)
	if err != nil || empty.Name != nil || empty.Description != nil || empty.SnapshotID != nil || empty.Force != nil || empty.Incremental != nil || empty.Wait != nil || empty.WaitPolicy.Timeout != nil || empty.Location != nil {
		t.Fatal(empty, err)
	}
	zero := time.Duration(0)
	policy, err := blockstorage.PrepareCreateVolumeBackupOptions(ctx,
		blockstorage.WithCreateVolumeBackupName(""), blockstorage.WithCreateVolumeBackupDescription(""),
		blockstorage.WithCreateVolumeBackupSnapshotID(""), blockstorage.WithCreateVolumeBackupForce(false),
		blockstorage.WithCreateVolumeBackupIncremental(false), blockstorage.WithCreateVolumeBackupWait(false),
		blockstorage.WithCreateVolumeBackupWaitPolicy(blockstorage.BackupMutationWaitOpts{Timeout: &zero, PollInterval: &zero}))
	if err != nil || policy.Name == nil || *policy.Name != "" || policy.Description == nil || *policy.Description != "" || policy.SnapshotID == nil || *policy.SnapshotID != "" || policy.Force == nil || *policy.Force || policy.Incremental == nil || *policy.Incremental || policy.Wait == nil || *policy.Wait || policy.WaitPolicy.Timeout == nil || *policy.WaitPolicy.Timeout != 0 || policy.WaitPolicy.PollInterval == nil {
		t.Fatal(policy, err)
	}
	del, err := blockstorage.PrepareDeleteVolumeBackupOptions(ctx, blockstorage.WithDeleteVolumeBackupForce(false), blockstorage.WithDeleteVolumeBackupWait(false), blockstorage.WithDeleteVolumeBackupWaitPolicy(policy.WaitPolicy))
	if err != nil || del.Force == nil || *del.Force || del.Wait == nil || *del.Wait || del.WaitPolicy.Timeout == nil || *del.WaitPolicy.Timeout != 0 {
		t.Fatal(del, err)
	}
}

func TestBackupMutationOptionsOwnFactoryInputsAndEveryPreparedTree(t *testing.T) {
	name, description, snapshot, flag := "before", "description", "snapshot", true
	timeout, interval := 3*time.Second, time.Millisecond
	location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"project"`), Name: &name}}
	create := blockstorage.WithCreateVolumeBackupOptions(blockstorage.CreateVolumeBackupOpts{Name: &name, Description: &description, SnapshotID: &snapshot, Force: &flag, Incremental: &flag, Wait: &flag, WaitPolicy: blockstorage.BackupMutationWaitOpts{Timeout: &timeout, PollInterval: &interval}, Location: &location})
	deleteOption := blockstorage.WithDeleteVolumeBackupOptions(blockstorage.DeleteVolumeBackupOpts{Force: &flag, Wait: &flag, WaitPolicy: blockstorage.BackupMutationWaitOpts{Timeout: &timeout, PollInterval: &interval}, Location: &location})
	name, description, snapshot, flag, timeout = "mutated", "changed", "changed", false, 0
	location.Project.ID[1] = '!'
	first, err := blockstorage.PrepareCreateVolumeBackupOptions(context.Background(), create)
	second, secondErr := blockstorage.PrepareCreateVolumeBackupOptions(context.Background(), create)
	del, delErr := blockstorage.PrepareDeleteVolumeBackupOptions(context.Background(), deleteOption)
	if err != nil || secondErr != nil || delErr != nil || *first.Name != "before" || *first.Description != "description" || *first.SnapshotID != "snapshot" || !*first.Force || !*first.Incremental || !*first.Wait || *first.WaitPolicy.Timeout != 3*time.Second || string(first.Location.Project.ID) != `"project"` || *first.Location.Project.Name != "before" {
		t.Fatal(first, err, secondErr, delErr)
	}
	*first.Name, *first.Force, *first.WaitPolicy.Timeout = "first", false, 0
	first.Location.Project.ID[1] = '!'
	*first.Location.Project.Name = "first"
	if *second.Name != "before" || !*second.Force || *second.WaitPolicy.Timeout != 3*time.Second || string(second.Location.Project.ID) != `"project"` || *second.Location.Project.Name != "before" || !*del.Force || !*del.Wait || *del.WaitPolicy.Timeout != 3*time.Second || string(del.Location.Project.ID) != `"project"` || *del.Location.Project.Name != "before" {
		t.Fatal("prepared policies share caller or sibling storage", second, del)
	}
}

func TestBackupMutationOptionsRunOriginalsOnceAndFreezeMembershipAndWrites(t *testing.T) {
	ctx := context.Background()
	var retained *blockstorage.CreateVolumeBackupOpts
	var calls int
	options := make([]blockstorage.CreateVolumeBackupOption, 2)
	options[0] = func(target *blockstorage.CreateVolumeBackupOpts) error {
		calls++
		retained = target
		name := "owned"
		target.Name = &name
		options[1] = func(*blockstorage.CreateVolumeBackupOpts) error { t.Error("replaced option ran"); return nil }
		return nil
	}
	options[1] = func(target *blockstorage.CreateVolumeBackupOpts) error {
		calls++
		*retained.Name = "late retained change"
		if *target.Name != "owned" {
			t.Error("next original borrowed previous callback storage")
		}
		return nil
	}
	policy, err := blockstorage.PrepareCreateVolumeBackupOptions(ctx, options...)
	if err != nil || calls != 2 || policy.Name == nil || *policy.Name != "owned" {
		t.Fatal(policy, err, calls)
	}
	*retained.Name = "after"
	if *policy.Name != "owned" {
		t.Fatal("prepared output borrowed callback pointer")
	}
	var deleteCalls int
	del, err := blockstorage.PrepareDeleteVolumeBackupOptions(ctx, func(target *blockstorage.DeleteVolumeBackupOpts) error {
		deleteCalls++
		flag := true
		target.Force = &flag
		return nil
	})
	if err != nil || deleteCalls != 1 || del.Force == nil || !*del.Force {
		t.Fatal(del, err, deleteCalls)
	}
}

func TestBackupMutationOptionsJoinCancellationAndOriginalErrors(t *testing.T) {
	cause, original := errors.New("caller cancellation"), errors.New("option failed")
	ctx, cancel := context.WithCancelCause(context.Background())
	policy, err := blockstorage.PrepareCreateVolumeBackupOptions(ctx, func(target *blockstorage.CreateVolumeBackupOpts) error {
		target.Name = new(string)
		cancel(cause)
		return original
	})
	if !errors.Is(err, original) || !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || policy.Name != nil {
		t.Fatal(policy, err)
	}
	for _, execute := range []func() error{
		func() error { _, e := blockstorage.PrepareCreateVolumeBackupOptions(nil); return e },
		func() error {
			_, e := blockstorage.PrepareCreateVolumeBackupOptions(context.Background(), nil)
			return e
		},
		func() error {
			_, e := blockstorage.PrepareDeleteVolumeBackupOptions(context.Background(), nil)
			return e
		},
	} {
		if e := execute(); !errors.Is(e, resource.ErrInvalidOption) {
			t.Fatal(e)
		}
	}
	var calls int
	_, err = blockstorage.PrepareDeleteVolumeBackupOptions(ctx, func(*blockstorage.DeleteVolumeBackupOpts) error { calls++; return nil })
	if calls != 0 || !errors.Is(err, cause) {
		t.Fatal(calls, err)
	}
}

func TestBackupMutationOptionsReuseFactoriesConcurrentlyWithOwnedLocations(t *testing.T) {
	name := "shared"
	option := blockstorage.WithCreateVolumeBackupName(name)
	location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"original"`)}}
	createLocation := blockstorage.WithCreateVolumeBackupLocation(location)
	deleteLocation := blockstorage.WithDeleteVolumeBackupLocation(location)
	var wg sync.WaitGroup
	for range 24 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			create, err := blockstorage.PrepareCreateVolumeBackupOptions(context.Background(), option, createLocation)
			del, delErr := blockstorage.PrepareDeleteVolumeBackupOptions(context.Background(), deleteLocation)
			if err != nil || delErr != nil || *create.Name != name || string(create.Location.Project.ID) != `"original"` || string(del.Location.Project.ID) != `"original"` {
				t.Error(create, del, err, delErr)
				return
			}
			*create.Name = "local"
			create.Location.Project.ID[1] = '!'
			del.Location.Project.ID[1] = '!'
		}()
	}
	wg.Wait()
	if string(location.Project.ID) != `"original"` {
		t.Fatal("factory borrowed original location")
	}
}
