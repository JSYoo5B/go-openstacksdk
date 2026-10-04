package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestVolumeSnapshotMutationOriginalFailuresJoinSourceAndContextBeforeHTTP(t *testing.T) {
	for _, family := range []string{"create", "delete"} {
		for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
			t.Run(family+"/"+fact, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := snapshotReadContractClient(cloud)
				var calls, callbacks atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				callbackCause, cancellation := errors.New("original snapshot option"), errors.New("original snapshot cancellation")
				callback := func() error {
					callbacks.Add(1)
					snapshotReadTransportMutate(client, fact)
					cancel(cancellation)
					return callbackCause
				}
				var err error
				if family == "create" {
					result, failure := blockstorage.CreateVolumeSnapshot(ctx, client, blockstorage.CreateVolumeSnapshotRequest{VolumeID: "literal"}, func(*blockstorage.CreateVolumeSnapshotOpts) error { return callback() })
					if result != nil {
						t.Fatal("preparation error created execution proof", result)
					}
					err = failure
				} else {
					result, failure := blockstorage.DeleteVolumeSnapshot(ctx, client, blockstorage.DeleteVolumeSnapshotRequest{NameOrID: "snap"}, func(*blockstorage.DeleteVolumeSnapshotOpts) error { return callback() })
					if result != nil {
						t.Fatal("preparation error created execution proof", result)
					}
					err = failure
				}
				var physical *resource.ResponseError
				if calls.Load() != 0 || callbacks.Load() != 1 || !errors.Is(err, callbackCause) || !errors.Is(err, cancellation) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) {
					t.Fatal(err, calls.Load(), callbacks.Load())
				}
			})
		}
	}
}

func TestVolumeSnapshotMutationDirectCaptureOwnsHeadersOptionsAndEveryPublishedStage(t *testing.T) {
	cloud := testcloud.New(t)
	client := snapshotReadContractClient(cloud)
	var calls, callbacks atomic.Int32
	wait := false
	fields := map[string]json.RawMessage{"name": json.RawMessage(`"owned"`)}
	var retained *blockstorage.CreateVolumeSnapshotOpts
	options := make([]blockstorage.CreateVolumeSnapshotOption, 2)
	options[0] = func(next *blockstorage.CreateVolumeSnapshotOpts) error {
		callbacks.Add(1)
		next.Wait, next.Attributes.Fields = &wait, fields
		retained = next
		options[1] = func(*blockstorage.CreateVolumeSnapshotOpts) error {
			t.Fatal("replacement original callback ran")
			return nil
		}
		return nil
	}
	options[1] = func(next *blockstorage.CreateVolumeSnapshotOpts) error {
		callbacks.Add(1)
		wait = true
		fields["name"][0] = '!'
		retained.Wait = &wait
		if next.Wait == nil || *next.Wait || string(next.Attributes.Fields["name"]) != `"owned"` {
			t.Error("prior original rewrote prepared snapshot policy", next)
		}
		client.MoreHeaders["x-source"] = "later ordinary header"
		cloud.Provider.SetToken("live-token")
		return nil
	}
	body := `{"snapshot":{"status":false,"unknown":{"large":900719925474099312345}}}`
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		requestBody, err := io.ReadAll(r.Body)
		if err != nil || r.Method != http.MethodPost || r.URL.Path != snapshotReadContractBasic || r.Header.Get("X-Source") != "entry" || r.Header.Get("X-Auth-Token") != "live-token" || r.Header.Get("OpenStack-API-Version") != "volume 3.60" || string(requestBody) != `{"snapshot":{"force":false,"name":"owned","volume_id":"literal"}}` {
			t.Error(r.Method, r.URL, r.Header, string(requestBody), err)
		}
		w.Header().Set("X-Proof", "created")
		testcloud.JSON(w, 203, body)
	})
	result, err := blockstorage.CreateVolumeSnapshot(context.Background(), client, blockstorage.CreateVolumeSnapshotRequest{VolumeID: "literal"}, options...)
	if err != nil || result == nil || result.Created == nil || result.LastAccepted == nil || result.CreatedSnapshot == nil || result.Snapshot == nil || result.Ready != nil || result.ReadySnapshot != nil || string(result.SnapshotID) != "null" || calls.Load() != 1 || callbacks.Load() != 2 {
		t.Fatal(result, err, calls.Load(), callbacks.Load())
	}
	created, final := snapshotReadContractFields(t, result.CreatedValue), snapshotReadContractFields(t, result.Value)
	if string(created["name"]) != `"owned"` || string(final["name"]) != `"owned"` || string(final["status"]) != "false" || string(final["id"]) != "null" {
		t.Fatal(created, final)
	}
	result.Created.Body[0] = '!'
	result.Created.Header.Set("X-Proof", "changed")
	result.CreatedValue[0] = '!'
	result.CreatedSnapshot.Body["unknown"][0] = '!'
	result.CreatedSnapshot.Header.Set("X-Proof", "changed")
	if string(result.LastAccepted.Body) != body || result.LastAccepted.Header.Get("X-Proof") != "created" || result.Value[0] != '{' || string(result.Snapshot.Body["unknown"]) != `{"large":900719925474099312345}` || result.Snapshot.Header.Get("X-Proof") != "created" {
		t.Fatal("published snapshot phases share mutable fields", result)
	}
	if _, present := result.Snapshot.Body["name"]; present {
		t.Fatal("request name became actual server field", result.Snapshot.Body)
	}
}
