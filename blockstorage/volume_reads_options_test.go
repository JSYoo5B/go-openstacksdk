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

type volumeReadContractOutcome struct {
	nilResult bool
	value     json.RawMessage
	volume    *resource.RawResource
	volumes   []*resource.RawResource
	exists    *bool
	observed  *blockstorage.GetVolumesPage
	pages     []*blockstorage.GetVolumesPage
	err       error
}

func volumeReadContractCall(ctx context.Context, client *gophercloud.ServiceClient, operation string, options ...blockstorage.VolumeReadOption) volumeReadContractOutcome {
	switch operation {
	case "ListVolumes":
		result, err := blockstorage.ListVolumes(ctx, client, options...)
		if result == nil {
			return volumeReadContractOutcome{nilResult: true, err: err}
		}
		return volumeReadContractOutcome{value: result.Value, volumes: result.Volumes, pages: result.Pages, err: err}
	case "GetVolumeByID":
		result, err := blockstorage.GetVolumeByID(ctx, client, blockstorage.GetVolumeByIDRequest{ID: "target"}, options...)
		if result == nil {
			return volumeReadContractOutcome{nilResult: true, err: err}
		}
		return volumeReadContractOutcome{value: result.Value, volume: result.Volume, observed: result.Observed, pages: result.Pages, err: err}
	case "VolumeExists":
		result, err := blockstorage.VolumeExists(ctx, client, blockstorage.VolumeExistsRequest{NameOrID: "target"}, options...)
		if result == nil {
			return volumeReadContractOutcome{nilResult: true, err: err}
		}
		return volumeReadContractOutcome{value: result.Value, volume: result.Volume, exists: result.Exists, observed: result.Observed, pages: result.Pages, err: err}
	default:
		panic("unknown public volume read operation")
	}
}

func TestPrepareVolumeReadOptionsOwnsFactoriesCallbacksAndOriginalSlice(t *testing.T) {
	defaults, err := blockstorage.PrepareVolumeReadOptions(context.Background())
	if err != nil || defaults.Location != nil {
		t.Fatal(defaults, err)
	}
	for _, kind := range []string{"complete options", "location factory"} {
		t.Run(kind, func(t *testing.T) {
			cloudName, projectName := "cloud-entry", "project-entry"
			location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"project-entry"`), Name: &projectName}}
			var factory blockstorage.VolumeReadOption
			if kind == "complete options" {
				factory = blockstorage.WithVolumeReadOptions(blockstorage.VolumeReadOpts{Location: &location})
			} else {
				factory = blockstorage.WithVolumeReadLocation(location)
			}
			cloudName, projectName = "caller-mutated", "caller-mutated"
			location.Project.ID[1] = '!'
			var retained *blockstorage.VolumeReadOpts
			var order []string
			var options []blockstorage.VolumeReadOption
			options = []blockstorage.VolumeReadOption{factory, func(next *blockstorage.VolumeReadOpts) error {
				order = append(order, "first")
				retained = next
				options[2] = func(*blockstorage.VolumeReadOpts) error { return errors.New("replacement callback must not run") }
				return nil
			}, func(next *blockstorage.VolumeReadOpts) error {
				order = append(order, "second")
				*retained.Location.Cloud = "retained-mutated"
				retained.Location.Project.ID[1] = '!'
				if *next.Location.Cloud != "cloud-entry" || string(next.Location.Project.ID) != `"project-entry"` || *next.Location.Project.Name != "project-entry" {
					t.Fatal("retained callback state aliased next option", next)
				}
				return nil
			}}
			prepared, err := blockstorage.PrepareVolumeReadOptions(context.Background(), options...)
			if err != nil || prepared.Location == nil || *prepared.Location.Cloud != "cloud-entry" || string(prepared.Location.Project.ID) != `"project-entry"` || !reflect.DeepEqual(order, []string{"first", "second"}) {
				t.Fatal(prepared, err, order)
			}
			*prepared.Location.Cloud = "returned-mutated"
			prepared.Location.Project.ID[1] = '!'
			again, err := blockstorage.PrepareVolumeReadOptions(context.Background(), factory)
			if err != nil || *again.Location.Cloud != "cloud-entry" || string(again.Location.Project.ID) != `"project-entry"` {
				t.Fatal("factory reused caller/result memory", again, err)
			}
		})
	}
}

func TestVolumeReadsApplyOriginalOptionsOnceWithCapturedHeadersAndOwnedLocation(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			cloudName, projectName := "cloud-entry", "project-entry"
			location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
			factory := blockstorage.WithVolumeReadLocation(location)
			cloudName, projectName = "outside-mutated", "outside-mutated"
			location.Project.ID[1] = '!'
			var callbacks, calls atomic.Int32
			var retained *blockstorage.VolumeReadOpts
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				path := volumeReadContractBase + "volumes/target"
				if operation == "ListVolumes" {
					path = volumeReadContractPath
				}
				volumeReadContractWire(t, r, path, "callback-live-token")
				*retained.Location.Cloud = "during-request-mutated"
				retained.Location.Project.ID[1] = '!'
				if operation == "ListVolumes" {
					testcloud.JSON(w, 200, `{"volumes":[{"id":"wire","availability_zone":"az"}]}`)
				} else {
					testcloud.JSON(w, 200, `{"volume":{"id":"wire","availability_zone":"az"}}`)
				}
			})
			got := volumeReadContractCall(context.Background(), client, operation, factory, func(next *blockstorage.VolumeReadOpts) error {
				callbacks.Add(1)
				retained = next
				client.MoreHeaders["x-source"] = "callback-mutated"
				cloud.Provider.SetToken("callback-live-token")
				return nil
			})
			if got.err != nil || got.nilResult || callbacks.Load() != 1 || calls.Load() != 1 {
				t.Fatal(got, callbacks.Load(), calls.Load())
			}
			view := got.value
			if operation == "ListVolumes" {
				var rows []json.RawMessage
				if err := json.Unmarshal(view, &rows); err != nil || len(rows) != 1 {
					t.Fatal(string(view), err)
				}
				view = rows[0]
			}
			fields := volumeReadContractFields(t, view)
			computed := volumeReadContractFields(t, fields["location"])
			project := volumeReadContractFields(t, computed["project"])
			if string(computed["cloud"]) != `"cloud-entry"` || string(computed["zone"]) != `"az"` || string(project["id"]) != `"scope"` || string(project["name"]) != `"project-entry"` {
				t.Fatal("location snapshot changed after factory/callback", string(view))
			}
			if operation == "VolumeExists" && (got.exists == nil || !*got.exists) {
				t.Fatal(got)
			}
		})
	}
}

func TestVolumeReadsOriginalSourceMutationIsTerminalBeforeRestoringOption(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
			t.Run(operation+"/"+fact, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				original := *client
				var calls, restore atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
				got := volumeReadContractCall(context.Background(), client, operation, func(*blockstorage.VolumeReadOpts) error {
					switch fact {
					case "provider":
						client.ProviderClient = &gophercloud.ProviderClient{}
					case "endpoint":
						client.Endpoint = cloud.Server.URL + "/changed/"
					case "resource base":
						client.ResourceBase = cloud.Server.URL + "/changed/"
					case "type":
						client.Type = "volume"
					case "microversion":
						client.Microversion = "3.61"
					}
					return nil
				}, func(*blockstorage.VolumeReadOpts) error { restore.Add(1); *client = original; return nil })
				if !got.nilResult || !errors.Is(got.err, resource.ErrInvalidOption) || calls.Load() != 0 || restore.Load() != 0 {
					t.Fatal("source captured after options or restored after terminal boundary", got, calls.Load(), restore.Load())
				}
				volumeReadContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestVolumeReadPreflightStopsOptionsAndHTTPOnInvalidSourceOrContext(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		for _, kind := range []string{"nil context", "canceled context", "nil client", "wrong service", "nil option", "callback cancellation"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := volumeReadContractClient(cloud)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("volume-read preflight cancellation")
				var calls, originals, later atomic.Int32
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
				first := blockstorage.VolumeReadOption(func(*blockstorage.VolumeReadOpts) error {
					originals.Add(1)
					if kind == "callback cancellation" {
						cancel(cause)
					}
					return nil
				})
				switch kind {
				case "nil context":
					ctx = nil
				case "canceled context":
					cancel(cause)
				case "nil client":
					client = nil
				case "wrong service":
					client.Type = "compute"
				case "nil option":
					first = nil
				}
				got := volumeReadContractCall(ctx, client, operation, first, func(*blockstorage.VolumeReadOpts) error { later.Add(1); return nil })
				if !got.nilResult || got.err == nil || calls.Load() != 0 || later.Load() != 0 {
					t.Fatal(got, calls.Load(), originals.Load(), later.Load())
				}
				wantOriginal := int32(0)
				if kind == "callback cancellation" {
					wantOriginal = 1
				}
				if originals.Load() != wantOriginal {
					t.Fatal("original callbacks ran after failed preflight", originals.Load(), wantOriginal)
				}
				switch kind {
				case "canceled context", "callback cancellation":
					if !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) {
						t.Fatal(got.err)
					}
				case "wrong service":
					if !errors.Is(got.err, resource.ErrUnsupported) {
						t.Fatal(got.err)
					}
				default:
					if !errors.Is(got.err, resource.ErrInvalidOption) {
						t.Fatal(got.err)
					}
				}
				volumeReadContractOperation(t, got.err, operation)
			})
		}
	}
}

func TestGetVolumeByIDRejectsUnsafeIDsAfterOptionsWithoutHTTP(t *testing.T) {
	for _, id := range []string{"", ".", "..", "disk/name", "disk\\name", "disk?query", "disk#fragment", "disk%2Fname", "disk name", "disk\u00a0name", "disk\x00name", string([]byte{'i', 0xff})} {
		t.Run(id, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			var calls, originals atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			result, err := blockstorage.GetVolumeByID(context.Background(), client, blockstorage.GetVolumeByIDRequest{ID: id}, func(*blockstorage.VolumeReadOpts) error { originals.Add(1); return nil })
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || calls.Load() != 0 || originals.Load() != 1 {
				t.Fatal(result, err, calls.Load(), originals.Load())
			}
			volumeReadContractOperation(t, err, "GetVolumeByID")
		})
	}
}

func TestVolumeReadLocationIsConsumedOnlyByActualRowsWithoutBorrowingResponseFailure(t *testing.T) {
	for _, rows := range []string{`[]`, `[{"id":"wire"}]`} {
		t.Run(rows, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := volumeReadContractClient(cloud)
			body := volumeReadContractPage(rows, "")
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Proof", "origin")
				testcloud.JSON(w, 200, body)
			})
			location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`invalid JSON`)}}
			prepared, err := blockstorage.PrepareVolumeReadOptions(context.Background(), blockstorage.WithVolumeReadLocation(location))
			if err != nil || prepared.Location == nil {
				t.Fatal("options eagerly consumed caller location", prepared, err)
			}
			result, err := blockstorage.ListVolumes(context.Background(), client, blockstorage.WithVolumeReadOptions(prepared))
			if rows == `[]` {
				if err != nil || result == nil || string(result.Value) != "[]" {
					t.Fatal(result, err)
				}
				return
			}
			var accepted *resource.ResponseError
			if result == nil || result.Value != nil || result.Volumes != nil || len(result.Pages) != 1 || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &accepted) || string(result.Pages[0].Body) != body || result.Pages[0].Header.Get("X-Proof") != "origin" {
				t.Fatal(result, err)
			}
		})
	}
}
