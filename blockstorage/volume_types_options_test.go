package blockstorage_test

import (
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
	"github.com/gophercloud/gophercloud/v2"
	identity "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

func TestVolumeTypeOriginalOptionsOwnFactoriesRetainedStateHeadersAndCallbackSlice(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			cloudName, projectName := "owned-cloud", "owned-project"
			location := resource.CloudLocation{Cloud: &cloudName, Zone: json.RawMessage(`"owned-zone"`), Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
			filters := json.RawMessage(`{"extra_specs":{"nested":{"enabled":true}}}`)
			readFactory := blockstorage.WithVolumeTypeReadOptions(blockstorage.VolumeTypeReadOpts{Location: &location})
			searchFactory := blockstorage.WithVolumeTypeSearchOptions(blockstorage.VolumeTypeSearchOpts{Location: &location, Filters: &filters})
			cloudName, projectName = "caller-mutated", "caller-mutated"
			location.Project.ID[1] = '!'
			location.Zone[1] = '!'
			filters[0] = '!'
			var order []string
			var retainedRead *blockstorage.VolumeTypeReadOpts
			var retainedSearch *blockstorage.VolumeTypeSearchOpts
			var reads []blockstorage.VolumeTypeReadOption
			var searches []blockstorage.VolumeTypeSearchOption
			reads = []blockstorage.VolumeTypeReadOption{readFactory, func(next *blockstorage.VolumeTypeReadOpts) error {
				order = append(order, "first")
				retainedRead = next
				reads[2] = func(*blockstorage.VolumeTypeReadOpts) error { return errors.New("replaced original callback") }
				client.MoreHeaders["x-source"] = "after"
				cloud.Provider.SetToken("live")
				return nil
			}, func(next *blockstorage.VolumeTypeReadOpts) error {
				order = append(order, "second")
				*retainedRead.Location.Cloud = "retained"
				retainedRead.Location.Project.ID[1] = '!'
				if *next.Location.Cloud != "owned-cloud" || string(next.Location.Project.ID) != `"scope"` {
					t.Fatal("later original option borrowed retained state", next)
				}
				return nil
			}}
			searches = []blockstorage.VolumeTypeSearchOption{searchFactory, func(next *blockstorage.VolumeTypeSearchOpts) error {
				order = append(order, "first")
				retainedSearch = next
				searches[2] = func(*blockstorage.VolumeTypeSearchOpts) error { return errors.New("replaced original callback") }
				client.MoreHeaders["x-source"] = "after"
				cloud.Provider.SetToken("live")
				return nil
			}, func(next *blockstorage.VolumeTypeSearchOpts) error {
				order = append(order, "second")
				*retainedSearch.Location.Cloud = "retained"
				(*retainedSearch.Filters)[0] = '!'
				if *next.Location.Cloud != "owned-cloud" || string(*next.Filters) != `{"extra_specs":{"nested":{"enabled":true}}}` {
					t.Fatal("later original option borrowed retained state", next)
				}
				return nil
			}}
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeTypeContractPath, "live")
				if r.URL.RawQuery != "" {
					t.Error("filtered/public list query leaked", r.URL)
				}
				calls.Add(1)
				if operation == "ListVolumeTypes" {
					*retainedRead.Location.Project.Name = "HTTP retained"
				} else {
					*retainedSearch.Location.Project.Name = "HTTP retained"
					retainedSearch.Location.Zone[1] = '!'
					(*retainedSearch.Filters)[0] = '!'
				}
				testcloud.JSON(w, 200, volumeTypeContractPage(`[{"id":"target","extra_specs":{"nested":{"enabled":1}},"project_id":"wire-project","availability_zone":"wire-zone"}]`, ""))
			})
			got := volumeTypeContractCall(context.Background(), client, operation, reads, searches...)
			if got.err != nil || got.nilResult || calls.Load() != 1 || !reflect.DeepEqual(order, []string{"first", "second"}) {
				t.Fatal(got, calls.Load(), order)
			}
			view := got.value
			if operation != "GetVolumeType" {
				rows := volumeReadContractRows(t, view)
				if len(rows) != 1 {
					t.Fatal(string(view))
				}
				raw, _ := json.Marshal(rows[0])
				view = raw
			}
			computed := volumeReadContractFields(t, volumeReadContractFields(t, view)["location"])
			project := volumeReadContractFields(t, computed["project"])
			if string(computed["cloud"]) != `"owned-cloud"` || string(computed["zone"]) != `"owned-zone"` || string(project["id"]) != `"scope"` || string(project["name"]) != `"owned-project"` {
				t.Fatal("owned option snapshot changed", string(view))
			}
		})
	}
}

func TestVolumeTypeOriginalSourceMutationStopsBeforeRestoringOptionAndHTTP(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
			t.Run(operation+"/"+fact, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				original := *client
				var calls, restores atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
				mutate := func() {
					switch fact {
					case "provider":
						client.ProviderClient = &gophercloud.ProviderClient{}
					case "endpoint":
						client.Endpoint = cloud.Server.URL + "/changed/"
					case "resource base":
						client.ResourceBase = cloud.Server.URL + "/changed/"
					case "type":
						client.Type = "compute"
					case "microversion":
						client.Microversion = "3.61"
					}
				}
				reads := []blockstorage.VolumeTypeReadOption{func(*blockstorage.VolumeTypeReadOpts) error { mutate(); return nil }, func(*blockstorage.VolumeTypeReadOpts) error { restores.Add(1); *client = original; return nil }}
				searches := []blockstorage.VolumeTypeSearchOption{func(*blockstorage.VolumeTypeSearchOpts) error { mutate(); return nil }, func(*blockstorage.VolumeTypeSearchOpts) error { restores.Add(1); *client = original; return nil }}
				got := volumeTypeContractCall(context.Background(), client, operation, reads, searches...)
				if !got.nilResult || !errors.Is(got.err, resource.ErrInvalidOption) || calls.Load() != 0 || restores.Load() != 0 {
					t.Fatal(got, calls.Load(), restores.Load())
				}
				volumeTypeContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestVolumeTypePreflightContextRoleAndOriginalCallbacksFailBeforeHTTP(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		for _, kind := range []string{"nil context", "canceled", "nil client", "wrong role", "nil option", "callback cancel", "callback error"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				ctx := context.Background()
				cause := errors.New("original caller cause")
				callbacks := 0
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
				read := func(*blockstorage.VolumeTypeReadOpts) error { callbacks++; return nil }
				search := func(*blockstorage.VolumeTypeSearchOpts) error { callbacks++; return nil }
				reads := []blockstorage.VolumeTypeReadOption{read}
				searches := []blockstorage.VolumeTypeSearchOption{search}
				want := resource.ErrInvalidOption
				wantCallbacks := 0
				switch kind {
				case "nil context":
					ctx = nil
				case "canceled":
					next, cancel := context.WithCancelCause(ctx)
					cancel(cause)
					ctx = next
					want = context.Canceled
				case "nil client":
					client = nil
				case "wrong role":
					client.Type = "volumev2"
					want = resource.ErrUnsupported
				case "nil option":
					reads = append(reads, nil)
					searches = append(searches, nil)
					wantCallbacks = 1
				case "callback cancel":
					next, cancel := context.WithCancelCause(ctx)
					ctx = next
					reads = []blockstorage.VolumeTypeReadOption{func(*blockstorage.VolumeTypeReadOpts) error { callbacks++; cancel(cause); return nil }, read}
					searches = []blockstorage.VolumeTypeSearchOption{func(*blockstorage.VolumeTypeSearchOpts) error { callbacks++; cancel(cause); return nil }, search}
					want = context.Canceled
					wantCallbacks = 1
				case "callback error":
					reads = []blockstorage.VolumeTypeReadOption{func(*blockstorage.VolumeTypeReadOpts) error { callbacks++; return cause }, read}
					searches = []blockstorage.VolumeTypeSearchOption{func(*blockstorage.VolumeTypeSearchOpts) error { callbacks++; return cause }, search}
					want = cause
					wantCallbacks = 1
				}
				got := volumeTypeContractCall(ctx, client, operation, reads, searches...)
				if !got.nilResult || !errors.Is(got.err, want) || callbacks != wantCallbacks || calls.Load() != 0 {
					t.Fatal(got, callbacks, calls.Load())
				}
				if want == context.Canceled && !errors.Is(got.err, cause) {
					t.Fatal("custom cancellation cause lost", got.err)
				}
				volumeTypeContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestVolumeTypeInvalidOwnedLocationIsLazyAndNeverFabricatesResponseError(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		for _, known := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: " raw-only", true: " known-null"}[known], func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				var calls atomic.Int32
				row := `{"location":false,"metadata":{},"project_id":"unknown","availability_zone":"unknown"}`
				if known {
					row = `{"id":null,"location":false}`
				}
				body := volumeTypeContractPage("["+row+"]", "")
				path := volumeTypeContractPath
				if operation == "GetVolumeType" {
					body = `{"volume_type":` + row + `}`
					path = volumeTypeContractMember
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					volumeReadContractWire(t, r, path, "test-token")
					calls.Add(1)
					w.Header().Set("X-Proof", "admitted-location-origin")
					testcloud.JSON(w, 200, body)
				})
				location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`invalid-owned`)}}
				got := volumeTypeContractCall(context.Background(), client, operation, []blockstorage.VolumeTypeReadOption{blockstorage.WithVolumeTypeReadLocation(location)}, blockstorage.WithVolumeTypeSearchLocation(location))
				if got.nilResult || calls.Load() != 1 {
					t.Fatal(got, calls.Load())
				}
				if !known && operation != "GetVolumeType" {
					rows := volumeReadContractRows(t, got.value)
					if got.err != nil || len(got.types) != 1 || string(rows[0]["location"]) != "false" {
						t.Fatal(got)
					}
					return
				}
				var accepted *resource.ResponseError
				if got.value != nil || got.types != nil || got.typ != nil || !errors.Is(got.err, resource.ErrInvalidOption) || errors.As(got.err, &accepted) {
					t.Fatal("caller policy borrowed HTTP error proof", got)
				}
				proof := got.observed
				if operation != "GetVolumeType" {
					if len(got.pages) != 1 {
						t.Fatal(got)
					}
					proof = got.pages[0]
				}
				if proof == nil || string(proof.Body) != body || proof.Header.Get("X-Proof") != "admitted-location-origin" {
					t.Fatal("actual admitted origin lost", got)
				}
			})
		}
	}
}

func TestVolumeTypeRecordedProjectScopeSnapshotsAfterOptionsAndFailsOnlyWhenConsumed(t *testing.T) {
	for _, kind := range []string{"snapshot after option", "invalid scope raw-only", "invalid scope known-null"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			cause := errors.New("native recorded project extraction")
			auth := &identity.CreateResult{}
			auth.Header = http.Header{"X-Subject-Token": {"recorded"}}
			auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "before"}}}
			if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
				t.Fatal(err)
			}
			var callbacks, calls atomic.Int32
			row := `{"id":null,"location":false}`
			if kind == "invalid scope raw-only" {
				row = `{"location":false}`
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeReadContractWire(t, r, volumeTypeContractPath, "recorded")
				calls.Add(1)
				auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "HTTP-mutated"}}}
				w.Header().Set("X-Proof", "recorded-origin")
				testcloud.JSON(w, 200, volumeTypeContractPage("["+row+"]", ""))
			})
			result, err := blockstorage.ListVolumeTypes(context.Background(), client, func(*blockstorage.VolumeTypeReadOpts) error {
				callbacks.Add(1)
				if kind == "snapshot after option" {
					auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "after-option"}}}
				} else {
					auth.Err = cause
				}
				return nil
			})
			if result == nil || callbacks.Load() != 1 || calls.Load() != 1 || len(result.Pages) != 1 {
				t.Fatal(result, err, calls.Load())
			}
			if kind == "invalid scope known-null" {
				var physical *resource.ResponseError
				if !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) || result.Value != nil || result.Types != nil {
					t.Fatal(result, err)
				}
				return
			}
			if err != nil || len(result.Types) != 1 {
				t.Fatal(result, err)
			}
			rows := volumeReadContractRows(t, result.Value)
			if kind == "invalid scope raw-only" {
				if string(rows[0]["location"]) != "false" {
					t.Fatal(string(result.Value))
				}
			} else {
				project := volumeReadContractFields(t, volumeReadContractFields(t, rows[0]["location"])["project"])
				if string(project["id"]) != `"after-option"` {
					t.Fatal("scope consumed before options or after HTTP", string(result.Value))
				}
			}
		})
	}
}
