package blockstorage_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func TestRestoreVolumeBackupPrepareOwnsFactoryValuesAndRemainsReusableWithoutHTTP(t *testing.T) {
	defaults, err := blockstorage.PrepareRestoreVolumeBackupOptions(bmwContext(t))
	if err != nil || !reflect.DeepEqual(defaults, blockstorage.RestoreVolumeBackupOpts{}) {
		t.Fatal(defaults, err)
	}
	volume, name, cloudName := "volume before", "name before", "cloud before"
	seed := json.RawMessage(`{"id":"literal","metadata":{"owned":[9007199254740993]}}`)
	location := resource.CloudLocation{Cloud: &cloudName, Zone: json.RawMessage(`"configured zone"`), Project: resource.CloudProject{ID: json.RawMessage(`"project before"`)}}
	value := blockstorage.RestoreVolumeBackupOpts{VolumeID: &volume, Name: &name, Seed: seed, Location: &location}
	whole := blockstorage.WithRestoreVolumeBackupOptions(value)
	seedFactory := blockstorage.WithRestoreVolumeBackupSeed(seed)
	locationFactory := blockstorage.WithRestoreVolumeBackupLocation(location)
	volume = "caller changed volume"
	name = "caller changed name"
	cloudName = "caller changed cloud"
	seed[1] = '!'
	location.Project.ID[1] = '!'
	location.Zone[1] = '!'
	for iteration := 0; iteration < 2; iteration++ {
		prepared, err := blockstorage.PrepareRestoreVolumeBackupOptions(bmwContext(t), whole)
		if err != nil || prepared.VolumeID == nil || *prepared.VolumeID != "volume before" || prepared.Name == nil || *prepared.Name != "name before" || string(prepared.Seed) != `{"id":"literal","metadata":{"owned":[9007199254740993]}}` || prepared.Location == nil || prepared.Location.Cloud == nil || *prepared.Location.Cloud != "cloud before" || string(prepared.Location.Project.ID) != `"project before"` || string(prepared.Location.Zone) != `"configured zone"` {
			t.Fatal(prepared, err)
		}
		*prepared.VolumeID = "prepared changed"
		*prepared.Name = "prepared changed"
		prepared.Seed[1] = '?'
		*prepared.Location.Cloud = "prepared changed"
		prepared.Location.Project.ID[1] = '?'
		prepared.Location.Zone[1] = '?'
		individual, err := blockstorage.PrepareRestoreVolumeBackupOptions(bmwContext(t), blockstorage.WithRestoreVolumeBackupVolumeID(""), blockstorage.WithRestoreVolumeBackupName(""), seedFactory, locationFactory)
		if err != nil || individual.VolumeID == nil || *individual.VolumeID != "" || individual.Name == nil || *individual.Name != "" || string(individual.Seed) != `{"id":"literal","metadata":{"owned":[9007199254740993]}}` || individual.Location == nil || *individual.Location.Cloud != "cloud before" || string(individual.Location.Project.ID) != `"project before"` {
			t.Fatal("individual factory ownership or explicit empty changed", individual, err)
		}
		individual.Seed[1] = '?'
		individual.Location.Project.ID[1] = '?'
	}
	for _, raw := range []json.RawMessage{nil, json.RawMessage{}} {
		prepared, err := blockstorage.PrepareRestoreVolumeBackupOptions(bmwContext(t), blockstorage.WithRestoreVolumeBackupSeed(raw))
		if err != nil || (prepared.Seed == nil) != (raw == nil) {
			t.Fatal("omitted seed and explicit empty seed collapsed", prepared, err)
		}
	}
}

func TestRestoreVolumeBackupPrepareCapturesOriginalSliceAndIsolatesRetainedCallbacks(t *testing.T) {
	var calls [3]int
	var retained *blockstorage.RestoreVolumeBackupOpts
	var options []blockstorage.RestoreVolumeBackupOption
	options = []blockstorage.RestoreVolumeBackupOption{
		func(o *blockstorage.RestoreVolumeBackupOpts) error {
			calls[0]++
			retained = o
			name, cloud := "original name", "original cloud"
			o.Name = &name
			o.Seed = json.RawMessage(`{"name":"cached","metadata":{"n":[1]}}`)
			o.Location = &resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"owned"`)}}
			options[1] = func(*blockstorage.RestoreVolumeBackupOpts) error {
				t.Error("caller changed a later original callback slot")
				return nil
			}
			return nil
		},
		func(o *blockstorage.RestoreVolumeBackupOpts) error {
			calls[1]++
			*retained.Name = "retained changed"
			retained.Seed[1] = '!'
			*retained.Location.Cloud = "retained changed"
			retained.Location.Project.ID[1] = '!'
			if *o.Name != "original name" || string(o.Seed) != `{"name":"cached","metadata":{"n":[1]}}` || *o.Location.Cloud != "original cloud" || string(o.Location.Project.ID) != `"owned"` {
				t.Error("later callback borrowed retained earlier policy", o)
			}
			volume := "destination"
			o.VolumeID = &volume
			return nil
		},
		func(o *blockstorage.RestoreVolumeBackupOpts) error {
			calls[2]++
			if o.VolumeID == nil || *o.VolumeID != "destination" {
				t.Error(o)
			}
			return nil
		},
	}
	prepared, err := blockstorage.PrepareRestoreVolumeBackupOptions(bmwContext(t), options...)
	if err != nil || calls != [3]int{1, 1, 1} || prepared.Name == nil || *prepared.Name != "original name" || prepared.VolumeID == nil || *prepared.VolumeID != "destination" || string(prepared.Seed) != `{"name":"cached","metadata":{"n":[1]}}` || *prepared.Location.Cloud != "original cloud" || string(prepared.Location.Project.ID) != `"owned"` {
		t.Fatal(prepared, err, calls)
	}
	*retained.Name = "changed after return"
	retained.Seed[0] = '!'
	if *prepared.Name != "original name" || !bytes.Equal(prepared.Seed, []byte(`{"name":"cached","metadata":{"n":[1]}}`)) {
		t.Fatal("returned policy borrows a retained callback", prepared)
	}
}

func TestRestoreVolumeBackupPrepareStopsOnNilContextCancellationNilOptionAndJoinedCallbackCauses(t *testing.T) {
	for _, kind := range []string{"nil context", "already canceled", "nil option", "callback error", "callback cancel", "callback error and cancel"} {
		t.Run(kind, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(bmwContext(t))
			defer cancel(nil)
			callbackCause, cancelCause := errors.New("restore option callback"), errors.New("restore option custom cancellation")
			var first, later int
			var selected context.Context = ctx
			options := []blockstorage.RestoreVolumeBackupOption{func(o *blockstorage.RestoreVolumeBackupOpts) error {
				first++
				name := "changed before failure"
				o.Name = &name
				if kind == "callback cancel" || kind == "callback error and cancel" {
					cancel(cancelCause)
				}
				if kind == "callback error" || kind == "callback error and cancel" {
					return callbackCause
				}
				return nil
			}, func(*blockstorage.RestoreVolumeBackupOpts) error { later++; return nil }}
			switch kind {
			case "nil context":
				selected = nil
			case "already canceled":
				cancel(cancelCause)
			case "nil option":
				options[0] = nil
			}
			prepared, err := blockstorage.PrepareRestoreVolumeBackupOptions(selected, options...)
			if err == nil || !reflect.DeepEqual(prepared, blockstorage.RestoreVolumeBackupOpts{}) || later != 0 {
				t.Fatal(prepared, err, first, later)
			}
			if kind == "nil context" || kind == "nil option" {
				if !errors.Is(err, resource.ErrInvalidOption) || first != 0 {
					t.Fatal(err, first)
				}
			}
			if kind == "already canceled" || kind == "callback cancel" || kind == "callback error and cancel" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause) {
					t.Fatal("cancellation identity lost", err)
				}
			}
			if kind == "callback error" || kind == "callback error and cancel" {
				if !errors.Is(err, callbackCause) {
					t.Fatal("callback identity lost", err)
				}
			}
			wantFirst := 1
			if kind == "nil context" || kind == "nil option" || kind == "already canceled" {
				wantFirst = 0
			}
			if first != wantFirst {
				t.Fatal(first, wantFirst)
			}
		})
	}
}

func TestRestoreVolumeBackupCapturesSourceBeforeOriginalsAndRejectsDriftBeforeRestoration(t *testing.T) {
	for _, kind := range []string{"provider", "endpoint", "resource base", "type", "microversion", "ordinary header"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			provider, endpoint, base, role, version := client.ProviderClient, client.Endpoint, client.ResourceBase, client.Type, client.Microversion
			var first, later, calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				bmwRequest(t, r, "POST", bmwCollection+"/literal/restore", "3.60", "test-token")
				testcloud.JSON(w, 202, `{}`)
			})
			result, err := blockstorage.RestoreVolumeBackup(bmwContext(t), client, blockstorage.RestoreVolumeBackupRequest{BackupID: "literal"},
				func(*blockstorage.RestoreVolumeBackupOpts) error {
					first.Add(1)
					if kind == "ordinary header" {
						client.MoreHeaders["x-source"] = "changed during original"
					} else {
						snapshotReadTransportMutate(client, kind)
					}
					return nil
				},
				func(o *blockstorage.RestoreVolumeBackupOpts) error {
					later.Add(1)
					client.ProviderClient = provider
					client.Endpoint = endpoint
					client.ResourceBase = base
					client.Type = role
					client.Microversion = version
					name := "destination"
					o.Name = &name
					return nil
				})
			if first.Load() != 1 {
				t.Fatal(first.Load())
			}
			if kind == "ordinary header" {
				if err != nil || result == nil || result.Value == nil || calls.Load() != 1 || later.Load() != 1 || client.MoreHeaders["x-source"] != "changed during original" {
					t.Fatal(result, err, calls.Load(), later.Load())
				}
				return
			}
			var physical *resource.ResponseError
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) || calls.Load() != 0 || later.Load() != 0 {
				t.Fatal("later original restored a changed captured source", result, err, calls.Load(), later.Load())
			}
			bmwOperation(t, err, "RestoreVolumeBackup")
		})
	}
}

func TestRestoreVolumeBackupRejectsInvalidTargetSeedAndLocalDescriptorsBeforeHTTP(t *testing.T) {
	for _, kind := range []string{"no target", "both empty", "invalid text", "unsafe backup ID", "seed null", "seed list", "seed empty bytes", "seed invalid UTF8", "seed different ID", "seed null ID", "seed numeric ID", "seed invalid descriptor", "invalid location", "nil client", "nil context", "wrong role"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := bmwClient(cloud)
			ctx := bmwContext(t)
			id := "literal"
			name := "destination"
			policy := blockstorage.RestoreVolumeBackupOpts{Name: &name}
			wantInvalid := true
			wantUnsupported := false
			wantOriginal := int32(1)
			switch kind {
			case "no target":
				policy = blockstorage.RestoreVolumeBackupOpts{}
			case "both empty":
				empty := ""
				policy = blockstorage.RestoreVolumeBackupOpts{Name: &empty, VolumeID: &empty}
			case "invalid text":
				name = string([]byte{0xff})
			case "unsafe backup ID":
				id = "a/b"
			case "seed null":
				policy.Seed = json.RawMessage(`null`)
				wantInvalid = false
			case "seed list":
				policy.Seed = json.RawMessage(`[]`)
				wantInvalid = false
			case "seed empty bytes":
				policy.Seed = json.RawMessage{}
				wantInvalid = false
			case "seed invalid UTF8":
				policy.Seed = json.RawMessage{'{', '"', 'x', '"', ':', '"', 0xff, '"', '}'}
			case "seed different ID":
				policy.Seed = json.RawMessage(`{"id":"other"}`)
			case "seed null ID":
				policy.Seed = json.RawMessage(`{"id":null}`)
			case "seed numeric ID":
				policy.Seed = json.RawMessage(`{"id":1}`)
			case "seed invalid descriptor":
				policy.Seed = json.RawMessage(`{"size":"²"}`)
				wantInvalid = false
			case "invalid location":
				policy.Location = &resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`not-json`)}}
			case "nil client":
				client = nil
				wantOriginal = 0
			case "nil context":
				ctx = nil
				wantOriginal = 0
			case "wrong role":
				client.Type = "compute"
				wantUnsupported = true
				wantInvalid = false
				wantOriginal = 0
			}
			var calls, original atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("local restore failure reached HTTP", r.URL)
				w.WriteHeader(500)
			})
			result, err := blockstorage.RestoreVolumeBackup(ctx, client, blockstorage.RestoreVolumeBackupRequest{BackupID: id}, func(o *blockstorage.RestoreVolumeBackupOpts) error { original.Add(1); *o = policy; return nil })
			var physical *resource.ResponseError
			if result != nil || err == nil || errors.As(err, &physical) || calls.Load() != 0 || original.Load() != wantOriginal || (wantInvalid && !errors.Is(err, resource.ErrInvalidOption)) || (wantUnsupported && !errors.Is(err, resource.ErrUnsupported)) {
				t.Fatal(result, err, calls.Load(), original.Load())
			}
			bmwOperation(t, err, "RestoreVolumeBackup")
		})
	}
}
