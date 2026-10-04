package blockstorage_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func TestVolumeTypeAccessOriginalOptionsOwnLocationAndSliceAcrossBothPhysicalStages(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			cloudName, projectName := "owned-cloud", "owned-project"
			location := resource.CloudLocation{Cloud: &cloudName, Zone: json.RawMessage(`"owned-zone"`), Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
			factory := blockstorage.WithVolumeTypeReadOptions(blockstorage.VolumeTypeReadOpts{Location: &location})
			cloudName, projectName = "outside-mutated", "outside-mutated"
			location.Project.ID[1] = '!'
			var options []blockstorage.VolumeTypeReadOption
			var retained *blockstorage.VolumeTypeReadOpts
			var order []int
			var calls atomic.Int32
			options = []blockstorage.VolumeTypeReadOption{factory, func(next *blockstorage.VolumeTypeReadOpts) error {
				retained = next
				order = append(order, 1)
				options[2] = func(*blockstorage.VolumeTypeReadOpts) error { return errors.New("replaced original callback") }
				client.MoreHeaders["x-source"] = "callback changed"
				cloud.Provider.SetToken("callback-live")
				return nil
			}, func(next *blockstorage.VolumeTypeReadOpts) error {
				order = append(order, 2)
				*retained.Location.Cloud = "retained changed"
				retained.Location.Project.ID[1] = '!'
				if *next.Location.Cloud != "owned-cloud" || string(next.Location.Project.ID) != `"scope"` {
					t.Fatal("retained original option aliases next callback", next)
				}
				return nil
			}}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					volumeTypeAccessContractLookup(t, r, "callback-live")
					*retained.Location.Project.Name = "HTTP changed"
					retained.Location.Zone[1] = '!'
					client.MoreHeaders["x-source"] = "between stages"
					cloud.Provider.SetToken("phase-live")
					testcloud.JSON(w, 200, `{"volume_type":{"id":"returned-한글","project_id":"wire-foreign","availability_zone":"wire-zone"}}`)
				case 2:
					volumeTypeAccessContractSecond(t, r, operation, "project", "phase-live")
					if operation == "GetVolumeTypeAccess" {
						testcloud.JSON(w, 200, `{"volume_type_access":[{"project_id":"foreign-access"}]}`)
					} else {
						w.WriteHeader(202)
					}
				default:
					t.Error("option/lookup replay", r.URL)
					w.WriteHeader(500)
				}
			})
			got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", "project", options...)
			if got.err != nil || got.nilResult || got.resolved == nil || got.resolved.Type == nil || got.phase == nil || calls.Load() != 2 || !reflect.DeepEqual(order, []int{1, 2}) {
				t.Fatal(got, calls.Load(), order)
			}
			computed := volumeReadContractFields(t, volumeReadContractFields(t, got.resolved.Value)["location"])
			project := volumeReadContractFields(t, computed["project"])
			if string(computed["cloud"]) != `"owned-cloud"` || string(computed["zone"]) != `"owned-zone"` || string(project["id"]) != `"scope"` || string(project["name"]) != `"owned-project"` {
				t.Fatal("resolved location changed after capture", string(got.resolved.Value))
			}
			if operation == "GetVolumeTypeAccess" {
				if len(got.accesses) != 1 || len(got.accesses[0].Body) != 1 || string(got.accesses[0].Body["project_id"]) != `"foreign-access"` {
					t.Fatal("access rows inherited Type location policy", got)
				}
			}
		})
	}
}

func TestVolumeTypeAccessOriginalSourceViolationIsTerminalBeforeRestoringOption(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
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
				got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", "project", func(*blockstorage.VolumeTypeReadOpts) error { mutate(); return nil }, func(*blockstorage.VolumeTypeReadOpts) error { restores.Add(1); *client = original; return nil })
				if !got.nilResult || !errors.Is(got.err, resource.ErrInvalidOption) || calls.Load() != 0 || restores.Load() != 0 {
					t.Fatal(got, calls.Load(), restores.Load())
				}
				volumeTypeContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestVolumeTypeAccessPreflightContextRoleAndOptionsStopBeforeLookup(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, kind := range []string{"nil context", "canceled", "nil client", "wrong role", "nil option", "callback cancel", "callback error"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				ctx := context.Background()
				cause := errors.New("original caller cause")
				callbacks := 0
				var calls atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
				original := func(*blockstorage.VolumeTypeReadOpts) error { callbacks++; return nil }
				options := []blockstorage.VolumeTypeReadOption{original}
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
					options = append(options, nil)
					wantCallbacks = 1
				case "callback cancel":
					next, cancel := context.WithCancelCause(ctx)
					ctx = next
					options = []blockstorage.VolumeTypeReadOption{func(*blockstorage.VolumeTypeReadOpts) error { callbacks++; cancel(cause); return nil }, original}
					want = context.Canceled
					wantCallbacks = 1
				case "callback error":
					options = []blockstorage.VolumeTypeReadOption{func(*blockstorage.VolumeTypeReadOpts) error { callbacks++; return cause }, original}
					want = cause
					wantCallbacks = 1
				}
				got := volumeTypeAccessContractCall(ctx, client, operation, "target", "project", options...)
				if !got.nilResult || !errors.Is(got.err, want) || callbacks != wantCallbacks || calls.Load() != 0 {
					t.Fatal(got, callbacks, calls.Load())
				}
				if want == context.Canceled && !errors.Is(got.err, cause) {
					t.Fatal(got.err)
				}
				volumeTypeContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestVolumeTypeAccessMalformedCallerLocationKeepsLookupProofWithoutPhysicalError(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls atomic.Int32
			body := `{"volume_type":{"id":"returned-한글"}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				volumeTypeAccessContractLookup(t, r, "test-token")
				if calls.Add(1) != 1 {
					t.Error("bad resolved location reached second stage", r.URL)
				}
				w.Header().Set("X-Proof", "lookup-location")
				testcloud.JSON(w, 200, body)
			})
			location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`invalid-local`)}}
			got := volumeTypeAccessContractCall(context.Background(), client, operation, "target", "project", blockstorage.WithVolumeTypeReadLocation(location))
			var physical *resource.ResponseError
			if got.nilResult || got.resolved == nil || got.resolved.Value != nil || got.resolved.Type != nil || got.resolved.Observed == nil || string(got.resolved.Observed.Body) != body || got.resolved.Observed.Header.Get("X-Proof") != "lookup-location" || got.phase != nil || calls.Load() != 1 || !errors.Is(got.err, resource.ErrInvalidOption) || errors.As(got.err, &physical) {
				t.Fatal(got, calls.Load())
			}
		})
	}
}
