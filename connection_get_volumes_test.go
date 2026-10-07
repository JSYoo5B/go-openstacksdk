package gophercloudsdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionGetVolumesUsesOnlyCachedCinderAndOriginalOptionsOnce(t *testing.T) {
	cloud := testcloud.New(t)
	var locates, lists, callbacks atomic.Int32
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		return "", errors.New("unexpected service locator")
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/cinder/v3/p/"))
	if err != nil {
		t.Fatal(err)
	}
	cinder, err := conn.BlockStorageV3(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /cinder/v3/p/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live-token" || r.Header.Get("X-Entry") != "after-option" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"volumes":[{"id":false,"attachments":[{"server_id":" /server?x# "},{"server_id":" /server?x# "}]}]}`)
	})
	for range 2 {
		input := blockstorage.GetVolumesRequest{ServerID: " /server?x# "}
		result, err := conn.GetVolumes(context.Background(), input, func(*blockstorage.GetVolumesOpts) error {
			callbacks.Add(1)
			current, err := conn.BlockStorageV3(context.Background())
			if err != nil || current != cinder {
				t.Error(current, err)
				return errors.New("cached Cinder changed")
			}
			input.ServerID = "changed outside copied request"
			cinder.RawClient().MoreHeaders = map[string]string{"X-Entry": "after-option"}
			cloud.Provider.SetToken("live-token")
			return nil
		})
		if err != nil || result == nil || len(result.Pages) != 1 || len(result.Volumes) != 2 || result.Volumes[0] != result.Volumes[1] || string(result.Volumes[0].Body["id"]) != "false" {
			t.Fatal(result, err)
		}
	}
	if callbacks.Load() != 2 || lists.Load() != 2 || locates.Load() != 0 {
		t.Fatal(callbacks.Load(), lists.Load(), locates.Load())
	}
}

func TestConnectionGetVolumesPreparesOptionsBeforeGetterAndDelaysUnusedServerFields(t *testing.T) {
	cloud := testcloud.New(t)
	var locates, lists, callbacks atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if callbacks.Load() != 2 || opts.Type == "compute" {
			t.Error("service selection preceded originals or selected Nova", opts)
		}
		return cloud.Server.URL + "/cinder/v3/p/", nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET /cinder/v3/p/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		lists.Add(1)
		testcloud.JSON(w, 200, `{"volumes":[{"attachments":[]}]}`)
	})
	rejected := errors.New("option rejected")
	result, err := conn.GetVolumes(context.Background(), blockstorage.GetVolumesRequest{}, func(*blockstorage.GetVolumesOpts) error { callbacks.Add(1); return rejected })
	if result != nil || !errors.Is(err, rejected) || locates.Load() != 0 || lists.Load() != 0 {
		t.Fatal(result, err, locates.Load())
	}
	for range 2 {
		result, err = conn.GetVolumes(context.Background(), blockstorage.GetVolumesRequest{}, blockstorage.WithGetVolumesServerFields(map[string]json.RawMessage{"id": json.RawMessage(`not JSON`)}), func(*blockstorage.GetVolumesOpts) error { callbacks.Add(1); return nil })
		if err != nil || result == nil || result.Volumes == nil || len(result.Volumes) != 0 || len(result.Pages) != 1 {
			t.Fatal("unused server fields were consumed before/after empty attachment list", result, err)
		}
	}
	if callbacks.Load() != 3 || locates.Load() != 1 || lists.Load() != 2 {
		t.Fatal(callbacks.Load(), locates.Load(), lists.Load())
	}
}

func TestConnectionGetVolumesInvalidEntryNilOptionsAndOptionCancellationSelectNoService(t *testing.T) {
	cloud := testcloud.New(t)
	var locates atomic.Int32
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		return "", errors.New("unexpected locator")
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cause := errors.New("caller canceled before get volumes")
	canceled, cancel := context.WithCancelCause(context.Background())
	cancel(cause)
	for _, name := range []string{"nil context", "nil Connection", "canceled", "nil option", "callback cancellation"} {
		t.Run(name, func(t *testing.T) {
			ctx, current := context.Background(), conn
			callbacks := 0
			options := []blockstorage.GetVolumesOption{func(*blockstorage.GetVolumesOpts) error { callbacks++; return nil }}
			want := resource.ErrInvalidOption
			expectedCallbacks := 0
			switch name {
			case "nil context":
				ctx = nil
			case "nil Connection":
				current = nil
			case "canceled":
				ctx = canceled
				want = context.Canceled
			case "nil option":
				options = append(options, nil)
				expectedCallbacks = 1
			case "callback cancellation":
				var cancel context.CancelCauseFunc
				ctx, cancel = context.WithCancelCause(ctx)
				options = []blockstorage.GetVolumesOption{func(*blockstorage.GetVolumesOpts) error { callbacks++; cancel(cause); return nil }, func(*blockstorage.GetVolumesOpts) error { callbacks++; return nil }}
				want = context.Canceled
				expectedCallbacks = 1
			}
			result, err := current.GetVolumes(ctx, blockstorage.GetVolumesRequest{ServerID: "literal no Ref validation"}, options...)
			if result != nil || !errors.Is(err, want) || callbacks != expectedCallbacks || locates.Load() != 0 {
				t.Fatal(result, err, callbacks, locates.Load())
			}
			if (name == "canceled" || name == "callback cancellation") && !errors.Is(err, cause) {
				t.Fatal(err)
			}
		})
	}
}

func TestConnectionGetVolumesGetterCancellationKeepsOriginalCause(t *testing.T) {
	cloud := testcloud.New(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	cause := errors.New("Cinder initialization canceled")
	var locates, callbacks atomic.Int32
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) { locates.Add(1); cancel(cause); return "", cause }
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	result, err := conn.GetVolumes(ctx, blockstorage.GetVolumesRequest{}, func(*blockstorage.GetVolumesOpts) error { callbacks.Add(1); return nil })
	var operation *resource.OperationError
	if result != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.As(err, &operation) || operation.Operation != "GetVolumes" || locates.Load() != 1 || callbacks.Load() != 1 {
		t.Fatal(result, err, locates.Load(), callbacks.Load())
	}
}
