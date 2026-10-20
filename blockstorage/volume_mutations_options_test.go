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

func TestVolumeMutationPrepareDefaultsAndFactoriesOwnCompleteReusablePolicies(t *testing.T) {
	defaults, err := blockstorage.PrepareSetVolumeBootableOptions(context.Background())
	if err != nil || defaults.Bootable == nil || !*defaults.Bootable || defaults.Location != nil {
		t.Fatal(defaults, err)
	}
	name, description, cloudName, projectName := "owned name", "owned description", "owned cloud", "owned project"
	flag := false
	location := resource.CloudLocation{Cloud: &cloudName, Zone: json.RawMessage(`"owned zone"`), Project: resource.CloudProject{ID: json.RawMessage(`"owned project id"`), Name: &projectName}}
	metadata := map[string]string{"key": "owned"}
	fields := map[string]json.RawMessage{"status": json.RawMessage(`{"nested":[false,9007199254740993]}`)}
	uFactory := blockstorage.WithUpdateVolumeOptions(blockstorage.UpdateVolumeOpts{Attributes: blockstorage.UpdateVolumeAttributes{Name: &name, Description: &description, Metadata: metadata, Fields: fields}, Location: &location})
	attrsFactory := blockstorage.WithUpdateVolumeAttributes(blockstorage.UpdateVolumeAttributes{Name: &name, Description: &description, Metadata: metadata, Fields: fields})
	bFactory := blockstorage.WithSetVolumeBootableOptions(blockstorage.SetVolumeBootableOpts{Bootable: &flag, Location: &location})
	name, description, cloudName, projectName = "outside", "outside", "outside", "outside"
	flag = true
	metadata["key"] = "outside"
	fields["status"][0] = '!'
	location.Project.ID[1] = '!'
	for i := 0; i < 2; i++ {
		u, err := blockstorage.PrepareUpdateVolumeOptions(context.Background(), uFactory, attrsFactory)
		if err != nil || u.Attributes.Name == nil || *u.Attributes.Name != "owned name" || *u.Attributes.Description != "owned description" || u.Attributes.Metadata["key"] != "owned" || string(u.Attributes.Fields["status"]) != `{"nested":[false,9007199254740993]}` || u.Location == nil || *u.Location.Cloud != "owned cloud" || string(u.Location.Project.ID) != `"owned project id"` {
			t.Fatal(u, err)
		}
		b, err := blockstorage.PrepareSetVolumeBootableOptions(context.Background(), bFactory)
		if err != nil || b.Bootable == nil || *b.Bootable || b.Location == nil || *b.Location.Project.Name != "owned project" {
			t.Fatal(b, err)
		}
		*u.Attributes.Name = "returned mutated"
		u.Attributes.Fields["status"][0] = '!'
		u.Location.Project.ID[0] = '!'
		*b.Bootable = true
		*b.Location.Cloud = "returned mutated"
	}
	replacement, err := blockstorage.PrepareSetVolumeBootableOptions(context.Background(), blockstorage.WithSetVolumeBootable(false), blockstorage.WithSetVolumeBootableOptions(blockstorage.SetVolumeBootableOpts{}))
	if err != nil || replacement.Bootable == nil || !*replacement.Bootable {
		t.Fatal("replacement/default order", replacement, err)
	}
}

func TestVolumeMutationOriginalOptionsOwnSlicesPointersFieldsAndHeadersAcrossBothStages(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		t.Run(op, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			cloudName, projectName := "owned cloud", "owned project"
			location := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &projectName}}
			var order []int
			var calls atomic.Int32
			var uRetained *blockstorage.UpdateVolumeOpts
			var bRetained *blockstorage.SetVolumeBootableOpts
			var updates []blockstorage.UpdateVolumeOption
			var boots []blockstorage.SetVolumeBootableOption
			updates = []blockstorage.UpdateVolumeOption{blockstorage.WithUpdateVolumeLocation(location), blockstorage.WithUpdateVolumeDescription("requested"), func(next *blockstorage.UpdateVolumeOpts) error {
				uRetained = next
				order = append(order, 1)
				updates[3] = func(*blockstorage.UpdateVolumeOpts) error { return errors.New("replaced callback") }
				client.MoreHeaders["x-source"] = "option changed"
				cloud.Provider.SetToken("option-live")
				return nil
			}, func(next *blockstorage.UpdateVolumeOpts) error {
				order = append(order, 2)
				*uRetained.Attributes.Description = "retained changed"
				*uRetained.Location.Cloud = "retained changed"
				uRetained.Location.Project.ID[1] = '!'
				if *next.Attributes.Description != "requested" || *next.Location.Cloud != "owned cloud" || string(next.Location.Project.ID) != `"scope"` {
					t.Fatal("retained config aliases next original", next)
				}
				return nil
			}}
			boots = []blockstorage.SetVolumeBootableOption{blockstorage.WithSetVolumeBootableLocation(location), blockstorage.WithSetVolumeBootable(false), func(next *blockstorage.SetVolumeBootableOpts) error {
				bRetained = next
				order = append(order, 1)
				boots[3] = func(*blockstorage.SetVolumeBootableOpts) error { return errors.New("replaced callback") }
				client.MoreHeaders["x-source"] = "option changed"
				cloud.Provider.SetToken("option-live")
				return nil
			}, func(next *blockstorage.SetVolumeBootableOpts) error {
				order = append(order, 2)
				*bRetained.Bootable = true
				*bRetained.Location.Cloud = "retained changed"
				bRetained.Location.Project.ID[1] = '!'
				if *next.Bootable || *next.Location.Cloud != "owned cloud" || string(next.Location.Project.ID) != `"scope"` {
					t.Fatal("retained config aliases next original", next)
				}
				return nil
			}}
			cloudName, projectName = "caller changed", "caller changed"
			location.Project.ID[1] = '!'
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					vmLookup(t, r, "option-live")
					if op == "UpdateVolume" {
						*uRetained.Location.Project.Name = "HTTP retained changed"
					} else {
						*bRetained.Location.Project.Name = "HTTP retained changed"
					}
					client.MoreHeaders["x-source"] = "HTTP changed"
					cloud.Provider.SetToken("second-live")
					testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글","availability_zone":"wire zone"}}`)
				case 2:
					body := vmMutation(t, r, op, "second-live")
					if op == "UpdateVolume" {
						if string(vmFields(t, body["volume"])["description"]) != `"requested"` {
							t.Error(body)
						}
					} else if string(vmFields(t, body["os-set_bootable"])["bootable"]) != "false" {
						t.Error(body)
					}
					w.WriteHeader(204)
				default:
					t.Error("originals/relookup replay", r.URL)
					w.WriteHeader(500)
				}
			})
			got := vmCall(context.Background(), client, op, "ref", updates, boots)
			if got.err != nil || got.nilResult || got.resolved == nil || got.resolved.Volume == nil || got.applied == nil || calls.Load() != 2 || !reflect.DeepEqual(order, []int{1, 2}) {
				t.Fatal(got, calls.Load(), order)
			}
			computed := vmFields(t, vmFields(t, got.resolved.Value)["location"])
			project := vmFields(t, computed["project"])
			if string(computed["cloud"]) != `"owned cloud"` || string(computed["zone"]) != `"wire zone"` || string(project["id"]) != `"scope"` || string(project["name"]) != `"owned project"` {
				t.Fatal(string(got.resolved.Value))
			}
		})
	}
}

func TestVolumeMutationRecordedScopeSnapshotsAfterOriginalsBeforeLookup(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		t.Run(op, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			auth := &identity.CreateResult{}
			auth.Header = http.Header{"X-Subject-Token": {"scope-token"}}
			auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "before option"}}}
			if err := cloud.Provider.SetTokenAndAuthResult(auth); err != nil {
				t.Fatal(err)
			}
			var calls, callbacks atomic.Int32
			mutate := func() {
				callbacks.Add(1)
				auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "after option"}}}
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				switch calls.Add(1) {
				case 1:
					vmLookup(t, r, "scope-token")
					auth.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": "HTTP different scope"}}}
					testcloud.JSON(w, 200, `{"volume":{"id":"returned-한글"}}`)
				case 2:
					vmMutation(t, r, op, "scope-token")
					testcloud.JSON(w, 203, `{"volume":{"availability_zone":"response zone"}}`)
				default:
					t.Error(r.URL)
					w.WriteHeader(500)
				}
			})
			got := vmCall(context.Background(), client, op, "ref", append(vmChanged(), func(*blockstorage.UpdateVolumeOpts) error { mutate(); return nil }), []blockstorage.SetVolumeBootableOption{func(*blockstorage.SetVolumeBootableOpts) error { mutate(); return nil }})
			if got.err != nil || got.nilResult || got.resolved == nil || got.applied == nil || calls.Load() != 2 || callbacks.Load() != 1 {
				t.Fatal(got, calls.Load(), callbacks.Load())
			}
			values := []json.RawMessage{got.resolved.Value}
			if op == "UpdateVolume" {
				values = append(values, got.value)
			}
			for _, value := range values {
				location := vmFields(t, vmFields(t, value)["location"])
				if string(vmFields(t, location["project"])["id"]) != `"after option"` {
					t.Fatal("provider scope reread during stage", string(value))
				}
			}
		})
	}
}

func TestVolumeMutationPreflightContextRoleCallbacksAndProtectedFieldsStopBeforeHTTP(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, kind := range []string{"nil context", "canceled", "nil client", "wrong role", "nil option", "callback cancel", "callback error"} {
			t.Run(op+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vmClient(cloud)
				ctx := context.Background()
				cause := errors.New("preflight original cause")
				var calls atomic.Int32
				callbacks := 0
				wantCallbacks := 0
				want := resource.ErrInvalidOption
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
				u := []blockstorage.UpdateVolumeOption{func(*blockstorage.UpdateVolumeOpts) error { callbacks++; return nil }}
				b := []blockstorage.SetVolumeBootableOption{func(*blockstorage.SetVolumeBootableOpts) error { callbacks++; return nil }}
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
					u = append(u, nil)
					b = append(b, nil)
					wantCallbacks = 1
				case "callback cancel":
					next, cancel := context.WithCancelCause(ctx)
					ctx = next
					u = []blockstorage.UpdateVolumeOption{func(*blockstorage.UpdateVolumeOpts) error { callbacks++; cancel(cause); return nil }, u[0]}
					b = []blockstorage.SetVolumeBootableOption{func(*blockstorage.SetVolumeBootableOpts) error { callbacks++; cancel(cause); return nil }, b[0]}
					want = context.Canceled
					wantCallbacks = 1
				case "callback error":
					u = []blockstorage.UpdateVolumeOption{func(*blockstorage.UpdateVolumeOpts) error { callbacks++; return cause }, u[0]}
					b = []blockstorage.SetVolumeBootableOption{func(*blockstorage.SetVolumeBootableOpts) error { callbacks++; return cause }, b[0]}
					want = cause
					wantCallbacks = 1
				}
				got := vmCall(ctx, client, op, "ref", u, b)
				if !got.nilResult || !errors.Is(got.err, want) || callbacks != wantCallbacks || calls.Load() != 0 {
					t.Fatal(got, callbacks, calls.Load())
				}
				if want == context.Canceled && !errors.Is(got.err, cause) {
					t.Fatal(got.err)
				}
				volumeReadContractOperation(t, got.err, op)
			})
		}
	}
	for _, key := range []string{"id", "location", "base_path", "microversion", "headers", "uri", "prepend_key", "has_body", "retry_on_conflict", "commit_method", "allow_commit"} {
		t.Run(key, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			got := vmCall(context.Background(), client, "UpdateVolume", "ref", []blockstorage.UpdateVolumeOption{blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{key: json.RawMessage(`null`)})}, nil)
			if !got.nilResult || !errors.Is(got.err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(got, calls.Load())
			}
		})
	}
	for _, fields := range []map[string]json.RawMessage{{"status": json.RawMessage(`not-json`)}, {"name": json.RawMessage(`"` + string([]byte{0xff}) + `"`)}} {
		t.Run("invalid known JSON", func(t *testing.T) {
			cloud := testcloud.New(t)
			client := vmClient(cloud)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
			got := vmCall(context.Background(), client, "UpdateVolume", "ref", []blockstorage.UpdateVolumeOption{blockstorage.WithUpdateVolumeFields(fields)}, nil)
			if !got.nilResult || !errors.Is(got.err, resource.ErrInvalidOption) || calls.Load() != 0 {
				t.Fatal(got, calls.Load())
			}
		})
	}
}

func TestVolumeMutationOriginalSourceChangeIsTerminalBeforeRestoringCallback(t *testing.T) {
	for _, op := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, fact := range []string{"provider", "endpoint", "resource base", "type", "microversion"} {
			t.Run(op+"/"+fact, func(t *testing.T) {
				cloud := testcloud.New(t)
				client := vmClient(cloud)
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
				u := []blockstorage.UpdateVolumeOption{func(*blockstorage.UpdateVolumeOpts) error { mutate(); return nil }, func(*blockstorage.UpdateVolumeOpts) error { restores.Add(1); *client = original; return nil }}
				b := []blockstorage.SetVolumeBootableOption{func(*blockstorage.SetVolumeBootableOpts) error { mutate(); return nil }, func(*blockstorage.SetVolumeBootableOpts) error { restores.Add(1); *client = original; return nil }}
				got := vmCall(context.Background(), client, op, "ref", u, b)
				if !got.nilResult || !errors.Is(got.err, resource.ErrInvalidOption) || calls.Load() != 0 || restores.Load() != 0 {
					t.Fatal(got, calls.Load(), restores.Load())
				}
			})
		}
	}
}
