package gophercloudsdk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	v3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	sdk "gophercloudsdk"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

const snapshotConnectionPath = "/snapshot-connection/v3/project/"

var snapshotConnectionOperations = []string{"ListVolumeSnapshots", "SearchVolumeSnapshots", "GetVolumeSnapshot", "GetVolumeSnapshotByID"}

type snapshotConnectionOutcome struct {
	value     json.RawMessage
	rows      []*resource.RawResource
	selected  *resource.RawResource
	pages     []*blockstorage.VolumeSnapshotsPage
	observed  *blockstorage.VolumeSnapshotsPage
	requested string
	seeded    bool
}

func snapshotConnectionCall(ctx context.Context, conn *sdk.Connection, operation, identity string, list []blockstorage.VolumeSnapshotListOption, search []blockstorage.VolumeSnapshotSearchOption, read []blockstorage.VolumeSnapshotReadOption) (*snapshotConnectionOutcome, error) {
	switch operation {
	case "ListVolumeSnapshots":
		value, err := conn.ListVolumeSnapshots(ctx, list...)
		if value == nil {
			return nil, err
		}
		return &snapshotConnectionOutcome{value: value.Value, rows: value.Snapshots, pages: value.Pages}, err
	case "SearchVolumeSnapshots":
		value, err := conn.SearchVolumeSnapshots(ctx, blockstorage.SearchVolumeSnapshotsRequest{NameOrID: identity}, search...)
		if value == nil {
			return nil, err
		}
		return &snapshotConnectionOutcome{value: value.Value, rows: value.Snapshots, pages: value.Pages}, err
	case "GetVolumeSnapshot":
		value, err := conn.GetVolumeSnapshot(ctx, blockstorage.GetVolumeSnapshotRequest{NameOrID: identity}, search...)
		if value == nil {
			return nil, err
		}
		return &snapshotConnectionOutcome{value: value.Value, selected: value.Snapshot, pages: value.Pages, observed: value.Observed, requested: value.RequestedID, seeded: value.SeededID}, err
	default:
		value, err := conn.GetVolumeSnapshotByID(ctx, blockstorage.GetVolumeSnapshotByIDRequest{ID: identity}, read...)
		if value == nil {
			return nil, err
		}
		return &snapshotConnectionOutcome{value: value.Value, selected: value.Snapshot, pages: value.Pages, observed: value.Observed, requested: value.RequestedID, seeded: value.SeededID}, err
	}
}

func snapshotConnectionCallbacks(callback func() error) ([]blockstorage.VolumeSnapshotListOption, []blockstorage.VolumeSnapshotSearchOption, []blockstorage.VolumeSnapshotReadOption) {
	return []blockstorage.VolumeSnapshotListOption{func(*blockstorage.VolumeSnapshotListOpts) error { return callback() }},
		[]blockstorage.VolumeSnapshotSearchOption{func(*blockstorage.VolumeSnapshotSearchOpts) error { return callback() }},
		[]blockstorage.VolumeSnapshotReadOption{func(*blockstorage.VolumeSnapshotReadOpts) error { return callback() }}
}

func snapshotConnectionOperation(t *testing.T, operation string, err error) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "volume snapshot" || wrapped.Cause == nil {
		t.Fatal(operation, err, wrapped)
	}
}

func snapshotConnectionFields(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		t.Fatal(string(raw), err)
	}
	return fields
}

func snapshotConnectionRow(t *testing.T, operation string, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	if operation == "ListVolumeSnapshots" || operation == "SearchVolumeSnapshots" {
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil || len(rows) != 1 {
			t.Fatal(string(raw), err)
		}
		raw = rows[0]
	}
	return snapshotConnectionFields(t, raw)
}

func snapshotConnectionRecordScope(provider *gophercloud.ProviderClient, project, token string) error {
	result := v3.CreateResult{}
	result.Header = http.Header{"X-Subject-Token": {token}}
	result.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": project, "name": "token-name-is-not-configured"}}}
	return provider.SetTokenAndAuthResult(result)
}

func TestConnectionVolumeSnapshotOriginalOptionsPrecedeCachedCinderSelectionForAllEntries(t *testing.T) {
	for _, operation := range snapshotConnectionOperations {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var callbacks, locates, reads atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("originals must precede only Cinder v3", callbacks.Load(), opts)
				}
				return cloud.Server.URL + snapshotConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			var cached *gophercloud.ServiceClient
			cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
				index := reads.Add(1)
				body, readErr := io.ReadAll(r.Body)
				if operation == "GetVolumeSnapshot" || operation == "GetVolumeSnapshotByID" || r.URL.RawQuery != "" || len(body) != 0 || readErr != nil || r.Header.Get("X-Auth-Token") != "live-snapshot-token" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.60" || index == 2 && r.Header.Get("X-Original") != "after-original" {
					t.Error(r.URL, r.Header, string(body), readErr)
				}
				w.Header().Set("X-Physical", "page")
				testcloud.JSON(w, 203, `{"snapshots":[{"id":"item","name":"item","size":"7","unknown":9007199254740993}]}`)
			})
			cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/item", func(w http.ResponseWriter, r *http.Request) {
				index := reads.Add(1)
				body, readErr := io.ReadAll(r.Body)
				if operation == "ListVolumeSnapshots" || operation == "SearchVolumeSnapshots" || r.URL.RawQuery != "" || len(body) != 0 || readErr != nil || r.Header.Get("X-Auth-Token") != "live-snapshot-token" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.60" || index == 2 && r.Header.Get("X-Original") != "after-original" {
					t.Error(r.URL, r.Header, string(body), readErr)
				}
				w.Header().Set("X-Physical", "member")
				testcloud.JSON(w, 203, `{"snapshot":{"id":"response-item","name":"item","size":"7","unknown":9007199254740993}}`)
			})
			for range 2 {
				list, search, read := snapshotConnectionCallbacks(func() error {
					callbacks.Add(1)
					cloud.Provider.SetToken("live-snapshot-token")
					if cached != nil {
						cached.MoreHeaders = map[string]string{"X-Original": "after-original"}
					}
					return nil
				})
				out, err := snapshotConnectionCall(context.Background(), conn, operation, "item", list, search, read)
				if err != nil || out == nil {
					t.Fatal(out, err)
				}
				row := snapshotConnectionRow(t, operation, out.value)
				if len(row) != 16 || string(row["size"]) != "7" {
					t.Fatal(string(out.value))
				}
				var physical *blockstorage.VolumeSnapshotsPage
				var raw *resource.RawResource
				if operation == "ListVolumeSnapshots" || operation == "SearchVolumeSnapshots" {
					if len(out.rows) != 1 || len(out.pages) != 1 || out.observed != nil {
						t.Fatal(out)
					}
					physical, raw = out.pages[0], out.rows[0]
				} else {
					if out.selected == nil || out.observed == nil || len(out.pages) != 0 || out.requested != "item" || out.seeded || string(row["id"]) != `"response-item"` {
						t.Fatal(out, string(out.value))
					}
					physical, raw = out.observed, out.selected
				}
				if physical.StatusCode != 203 || raw.StatusCode != 203 || string(raw.Body["unknown"]) != "9007199254740993" {
					t.Fatal(out, raw)
				}
				before := bytes.Clone(physical.Body)
				raw.Body["unknown"][0] = '!'
				raw.Header.Set("X-Physical", "mutated")
				out.value[0] = '!'
				if !bytes.Equal(physical.Body, before) || physical.Header.Get("X-Physical") == "mutated" {
					t.Fatal("raw/logical result borrows physical proof", out)
				}
				service, err := conn.BlockStorageV3(context.Background())
				if err != nil || cached != nil && cached != service.RawClient() {
					t.Fatal("cached Cinder identity changed", service, err)
				}
				cached = service.RawClient()
			}
			if callbacks.Load() != 2 || locates.Load() != 1 || reads.Load() != 2 {
				t.Fatal(callbacks.Load(), locates.Load(), reads.Load())
			}
		})
	}
}

func TestConnectionVolumeSnapshotRecordedScopeIsCapturedAfterOptionsBeforeGetterAndRetainedAcrossPages(t *testing.T) {
	cloud := testcloud.New(t)
	if err := snapshotConnectionRecordScope(cloud.Provider, "before-original", "before-token"); err != nil {
		t.Fatal(err)
	}
	var callbacks, locates, reads atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
			t.Error(callbacks.Load(), opts)
		}
		if err := snapshotConnectionRecordScope(cloud.Provider, "at-getter", "getter-token"); err != nil {
			t.Error(err)
			return "", err
		}
		return cloud.Server.URL + snapshotConnectionPath, nil
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("owned-region"))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
		index := reads.Add(1)
		wantToken := "page-token"
		if index == 1 {
			wantToken = "getter-token"
		}
		if r.Header.Get("X-Auth-Token") != wantToken {
			t.Error(r.Header, index, wantToken)
		}
		if r.URL.RawQuery == "" {
			if index == 1 {
				// SetToken clears native recorded auth; restore a NEW result so
				// both live token and next-call scope refresh are observable.
				cloud.Provider.SetToken("cleared-recorded-token")
				if err := snapshotConnectionRecordScope(cloud.Provider, "between-pages", "page-token"); err != nil {
					t.Error(err)
				}
			}
			testcloud.JSON(w, 203, `{"snapshots":[{"id":"first","location":"wire","availability_zone":"wire-zone"}],"next":"?marker=next"}`)
		} else {
			if r.URL.RawQuery != "marker=next" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 203, `{"snapshots":[{"id":"second"}]}`)
		}
	})
	for index := range 2 {
		out, err := conn.ListVolumeSnapshots(context.Background(), func(*blockstorage.VolumeSnapshotListOpts) error {
			callbacks.Add(1)
			if index == 0 {
				return snapshotConnectionRecordScope(cloud.Provider, "after-original", "original-token")
			}
			return nil
		})
		if err != nil || out == nil || len(out.Snapshots) != 2 || len(out.Pages) != 2 {
			t.Fatal(out, err)
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(out.Value, &rows); err != nil || len(rows) != 2 {
			t.Fatal(string(out.Value), err)
		}
		want := `"after-original"`
		if index == 1 {
			want = `"between-pages"`
		}
		for _, raw := range rows {
			location := snapshotConnectionFields(t, snapshotConnectionFields(t, raw)["location"])
			project := snapshotConnectionFields(t, location["project"])
			if string(project["id"]) != want || string(project["name"]) != "null" || string(location["cloud"]) != "null" || string(location["region_name"]) != `"owned-region"` || string(location["zone"]) != "null" {
				t.Fatal("call location reread after options", string(raw))
			}
		}
		current, err := conn.CurrentLocation()
		if err != nil || string(current.Project.ID) != `"between-pages"` || current.RegionName == nil || *current.RegionName != "owned-region" {
			t.Fatal(current, err)
		}
		current.Project.ID[0] = '!'
		*current.RegionName = "caller-mutated"
	}
	if callbacks.Load() != 2 || locates.Load() != 1 || reads.Load() != 4 {
		t.Fatal(callbacks.Load(), locates.Load(), reads.Load())
	}
}

func TestConnectionVolumeSnapshotExplicitLocationIsOwnedAndZoneIgnoredWithoutChangingDefaults(t *testing.T) {
	for _, operation := range snapshotConnectionOperations {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			defaultCloud, defaultName := "default-cloud", "default-name"
			defaults := resource.CloudLocation{Cloud: &defaultCloud, Project: resource.CloudProject{ID: json.RawMessage(`"default-project"`), Name: &defaultName}}
			cloudName, region, name, domain := "owned-cloud", "owned-region", "owned-name", "owned-domain"
			owned := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Zone: json.RawMessage(`{"unused":"zone"}`), Project: resource.CloudProject{ID: json.RawMessage(`"owned-project"`), Name: &name, DomainName: &domain}}
			list := []blockstorage.VolumeSnapshotListOption{blockstorage.WithVolumeSnapshotListLocation(owned)}
			search := []blockstorage.VolumeSnapshotSearchOption{blockstorage.WithVolumeSnapshotSearchLocation(owned)}
			read := []blockstorage.VolumeSnapshotReadOption{blockstorage.WithVolumeSnapshotReadLocation(owned)}
			var locates, reads atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(opts)
				}
				cloudName, region, name, domain = "changed", "changed", "changed", "changed"
				owned.Project.ID[0] = '!'
				owned.Zone[0] = '!'
				return cloud.Server.URL + snapshotConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(defaults))
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/", func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				if strings.HasSuffix(r.URL.Path, "/detail") {
					testcloud.JSON(w, 200, `{"snapshots":[{"id":"item","project_id":"owned-project","location":"wire","availability_zone":"wire"}]}`)
				} else {
					testcloud.JSON(w, 200, `{"snapshot":{"id":"item","project_id":"owned-project","location":"wire","availability_zone":"wire"}}`)
				}
			})
			out, err := snapshotConnectionCall(context.Background(), conn, operation, "item", list, search, read)
			if err != nil || out == nil || locates.Load() != 1 || reads.Load() != 1 {
				t.Fatal(out, err, locates.Load(), reads.Load())
			}
			location := snapshotConnectionFields(t, snapshotConnectionRow(t, operation, out.value)["location"])
			project := snapshotConnectionFields(t, location["project"])
			if string(location["cloud"]) != `"owned-cloud"` || string(location["region_name"]) != `"owned-region"` || string(location["zone"]) != "null" || string(project["id"]) != `"owned-project"` || string(project["name"]) != `"owned-name"` || string(project["domain_name"]) != `"owned-domain"` {
				t.Fatal(string(out.value))
			}
			current, err := conn.CurrentLocation()
			if err != nil || current.Cloud == nil || *current.Cloud != "default-cloud" || string(current.Project.ID) != `"default-project"` || current.Project.Name == nil || *current.Project.Name != "default-name" {
				t.Fatal("call override changed Connection defaults", current, err)
			}
		})
	}
}

func TestConnectionVolumeSnapshotListControlsArePerCallAndDoNotChangeCachedDefaults(t *testing.T) {
	cloud := testcloud.New(t)
	var callbacks, locates, reads atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
			t.Error(callbacks.Load(), opts)
		}
		return cloud.Server.URL + snapshotConnectionPath, nil
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		if r.URL.Query().Get("status") != "available" || r.URL.Query().Get("limit") != "1" || len(r.URL.Query()) != 2 || r.Header.Get("X-Trace") != "owned-header" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.41" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 203, `{"snapshots":[{"id":"first","description":"keep"},{"size":"²"}],"links":[null]}`)
	})
	cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		if r.URL.RawQuery != "" || r.Header.Get("X-Trace") != "" || r.Header.Get("Accept") != "application/json" || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.60" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"snapshots":[{"id":"default"}]}`)
	})
	headers := map[string]string{"X-Trace": "owned-header"}
	headerOption := blockstorage.WithVolumeSnapshotListHeaders(headers)
	headers["X-Trace"] = "caller-mutated"
	result, err := conn.ListVolumeSnapshots(context.Background(), func(*blockstorage.VolumeSnapshotListOpts) error { callbacks.Add(1); return nil },
		blockstorage.WithVolumeSnapshotListDetailed(false), blockstorage.WithVolumeSnapshotListPagination(false), blockstorage.WithVolumeSnapshotListMaxItems(1),
		blockstorage.WithVolumeSnapshotListMicroversion("3.41"), headerOption, blockstorage.WithVolumeSnapshotListFilters(json.RawMessage(`{"status":"available","description":"keep"}`)))
	if err != nil || result == nil || len(result.Snapshots) != 1 || len(result.Pages) != 1 || string(result.Snapshots[0].Body["id"]) != `"first"` {
		t.Fatal(result, err)
	}
	service, err := conn.BlockStorageV3(context.Background())
	if err != nil || service.RawClient().Microversion != "3.60" || len(service.RawClient().MoreHeaders) != 0 {
		t.Fatal("per-call control changed cached source", service, err)
	}
	result, err = conn.ListVolumeSnapshots(context.Background())
	if err != nil || result == nil || len(result.Snapshots) != 1 || string(result.Snapshots[0].Body["id"]) != `"default"` || callbacks.Load() != 1 || locates.Load() != 1 || reads.Load() != 2 {
		t.Fatal(result, err, callbacks.Load(), locates.Load(), reads.Load())
	}
}

func TestConnectionVolumeSnapshotPreflightOriginalAndGetterErrorsPreserveOuterOperationAndCause(t *testing.T) {
	for _, operation := range snapshotConnectionOperations {
		for _, kind := range []string{"nil context", "nil connection", "already canceled", "nil option", "option error", "option cancellation", "getter error", "getter cancellation", "invalid location"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, optionError, getterError := errors.New("snapshot call canceled"), errors.New("snapshot original failed"), errors.New("snapshot Cinder lookup failed")
				var callbacks, locates atomic.Int32
				cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
					locates.Add(1)
					if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
						t.Error(callbacks.Load(), opts)
					}
					if kind == "getter cancellation" {
						cancel(cause)
					}
					return "", getterError
				}
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				list, search, read := snapshotConnectionCallbacks(func() error {
					callbacks.Add(1)
					if kind == "option cancellation" {
						cancel(cause)
						return optionError
					}
					if kind == "option error" {
						return optionError
					}
					return nil
				})
				want, wantCallbacks, wantLocates := resource.ErrInvalidOption, int32(0), int32(0)
				switch kind {
				case "nil context":
					ctx = nil
				case "nil connection":
					conn = nil
				case "already canceled":
					cancel(cause)
					want = context.Canceled
				case "nil option":
					list = append(list, nil)
					search = append(search, nil)
					read = append(read, nil)
					wantCallbacks = 1
				case "option error":
					want = optionError
					wantCallbacks = 1
				case "option cancellation":
					want = context.Canceled
					wantCallbacks = 1
				case "invalid location":
					invalid := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`invalid`)}}
					list = append(list, blockstorage.WithVolumeSnapshotListLocation(invalid))
					search = append(search, blockstorage.WithVolumeSnapshotSearchLocation(invalid))
					read = append(read, blockstorage.WithVolumeSnapshotReadLocation(invalid))
					wantCallbacks = 1
				default:
					want = getterError
					wantCallbacks, wantLocates = 1, 1
				}
				out, err := snapshotConnectionCall(ctx, conn, operation, "item", list, search, read)
				if out != nil || !errors.Is(err, want) || callbacks.Load() != wantCallbacks || locates.Load() != wantLocates {
					t.Fatal(out, err, callbacks.Load(), locates.Load(), wantCallbacks, wantLocates)
				}
				snapshotConnectionOperation(t, operation, err)
				var physical *resource.ResponseError
				if errors.As(err, &physical) {
					t.Fatal("local preflight/getter invented response", err)
				}
				if kind == "option cancellation" && (!errors.Is(err, optionError) || !errors.Is(err, cause)) || kind == "getter cancellation" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, getterError)) || kind == "already canceled" && !errors.Is(err, cause) {
					t.Fatal("custom cause lost", err)
				}
			})
		}
	}
}

func TestConnectionVolumeSnapshotRawListBindingCollisionsFailBeforeServiceSelection(t *testing.T) {
	for _, raw := range []string{`{"details":true}`, `{"base_path":"/other"}`, `{"resource_type":null}`, `{"self":false}`, `{"__conflicting_attrs":{"paginated":false}}`, `{"__conflicting_attrs":{"session":null}}`} {
		t.Run(raw, func(t *testing.T) {
			cloud := testcloud.New(t)
			var callbacks, locates atomic.Int32
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				return "", errors.New("should not select Cinder")
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			out, err := conn.ListVolumeSnapshots(context.Background(), func(*blockstorage.VolumeSnapshotListOpts) error { callbacks.Add(1); return nil }, blockstorage.WithVolumeSnapshotListFilters(json.RawMessage(raw)))
			if out != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks.Load() != 1 || locates.Load() != 0 {
				t.Fatal(out, err, callbacks.Load(), locates.Load())
			}
			snapshotConnectionOperation(t, "ListVolumeSnapshots", err)
		})
	}
}

func TestConnectionVolumeSnapshotMemberSeedPreservesRawShapeAndByID404NeverFallsBack(t *testing.T) {
	for _, operation := range []string{"GetVolumeSnapshot", "GetVolumeSnapshotByID"} {
		for _, tc := range []struct {
			name, body, id string
			status         int
			seeded         bool
		}{
			{"missing canonical", `{"snapshot":{"name":"actual","unknown":9007199254740993}}`, `"item"`, 203, true},
			{"empty", "", `"item"`, 204, true},
			{"malformed", `{malformed`, `"item"`, 200, true},
			{"explicit null", `{"snapshot":{"id":null}}`, "null", 200, false},
			{"explicit false", `{"id":false}`, "false", 200, false},
			{"explicit object", `{"snapshot":{"id":{"opaque":9007199254740993}}}`, `{"opaque":9007199254740993}`, 200, false},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				cloud := testcloud.New(t)
				var locates, reads, pages atomic.Int32
				cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
					locates.Add(1)
					if opts.Type != "block-storage" || opts.Version != 3 {
						t.Error(opts)
					}
					return cloud.Server.URL + snapshotConnectionPath, nil
				}
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/item", func(w http.ResponseWriter, r *http.Request) {
					reads.Add(1)
					if r.URL.RawQuery != "" {
						t.Error(r.URL)
					}
					w.Header().Set("X-Seed-Proof", "actual")
					testcloud.JSON(w, tc.status, tc.body)
				})
				cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/detail", func(w http.ResponseWriter, r *http.Request) { pages.Add(1); testcloud.JSON(w, 200, `{"snapshots":[]}`) })
				out, err := snapshotConnectionCall(context.Background(), conn, operation, "item", nil, nil, nil)
				if err != nil || out == nil || out.selected == nil || out.observed == nil || out.observed.StatusCode != tc.status || out.observed.Header.Get("X-Seed-Proof") != "actual" || out.requested != "item" || out.seeded != tc.seeded || len(out.pages) != 0 || locates.Load() != 1 || reads.Load() != 1 || pages.Load() != 0 {
					t.Fatal(out, err, locates.Load(), reads.Load(), pages.Load())
				}
				row := snapshotConnectionFields(t, out.value)
				if string(row["id"]) != tc.id {
					t.Fatal(string(out.value))
				}
				_, rawHasID := out.selected.Body["id"]
				if rawHasID == tc.seeded || !bytes.Equal(out.observed.Body, []byte(tc.body)) {
					t.Fatal("logical seed fabricated wire ID", out.selected, string(out.observed.Body))
				}
			})
		}
	}
	cloud := testcloud.New(t)
	var members, pages atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		if opts.Type != "block-storage" || opts.Version != 3 {
			t.Error(opts)
		}
		return cloud.Server.URL + snapshotConnectionPath, nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/item", func(w http.ResponseWriter, r *http.Request) {
		members.Add(1)
		w.Header().Set("X-Rejected", "actual404")
		testcloud.JSON(w, 404, `{"error":"missing"}`)
	})
	cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
		pages.Add(1)
		if r.URL.Query().Get("name") != "item" || len(r.URL.Query()) != 1 {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"snapshots":[]}`)
	})
	byID, err := conn.GetVolumeSnapshotByID(context.Background(), blockstorage.GetVolumeSnapshotByIDRequest{ID: "item"})
	var rejected gophercloud.ErrUnexpectedResponseCode
	if byID == nil || byID.Value != nil || byID.Snapshot != nil || byID.Observed != nil || len(byID.Pages) != 0 || !errors.As(err, &rejected) || rejected.Actual != 404 || rejected.ResponseHeader.Get("X-Rejected") != "actual404" || pages.Load() != 0 {
		t.Fatal(byID, err, rejected, pages.Load())
	}
	snapshotConnectionOperation(t, "GetVolumeSnapshotByID", err)
	absent, err := conn.GetVolumeSnapshot(context.Background(), blockstorage.GetVolumeSnapshotRequest{NameOrID: "item"})
	if err != nil || absent == nil || absent.Value != nil || absent.Snapshot != nil || absent.Observed != nil || len(absent.Pages) != 1 || members.Load() != 2 || pages.Load() != 1 {
		t.Fatal(absent, err, members.Load(), pages.Load())
	}
}

func TestConnectionVolumeSnapshotOriginalSliceAndRetainedOptionPointersCannotChangePreparedPolicy(t *testing.T) {
	for _, operation := range snapshotConnectionOperations {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var callbacks, locates, reads atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if callbacks.Load() != 2 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(callbacks.Load(), opts)
				}
				return cloud.Server.URL + snapshotConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/", func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if operation == "ListVolumeSnapshots" {
					if r.URL.Query().Get("status") != "available" || len(r.URL.Query()) != 1 {
						t.Error(r.URL)
					}
				} else if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				if operation == "GetVolumeSnapshotByID" {
					testcloud.JSON(w, 200, `{"snapshot":{"id":"item","status":"available"}}`)
				} else {
					testcloud.JSON(w, 200, `{"snapshots":[{"id":"item","status":"available"}]}`)
				}
			})
			location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"owned-project"`)}}
			filters := json.RawMessage(`{"status":"available"}`)
			var retainedLocation *resource.CloudLocation
			var retainedFilters *json.RawMessage
			mutate := func() error {
				callbacks.Add(1)
				retainedLocation.Project.ID[1] = 'X'
				if retainedFilters != nil {
					(*retainedFilters)[0] = '!'
				}
				return nil
			}
			var list []blockstorage.VolumeSnapshotListOption
			var search []blockstorage.VolumeSnapshotSearchOption
			var read []blockstorage.VolumeSnapshotReadOption
			list = []blockstorage.VolumeSnapshotListOption{
				func(opts *blockstorage.VolumeSnapshotListOpts) error {
					callbacks.Add(1)
					opts.Location = &location
					opts.Filters = &filters
					retainedLocation, retainedFilters = opts.Location, opts.Filters
					list[1] = func(*blockstorage.VolumeSnapshotListOpts) error {
						return errors.New("replaced original should not run")
					}
					return nil
				},
				func(*blockstorage.VolumeSnapshotListOpts) error { return mutate() },
			}
			search = []blockstorage.VolumeSnapshotSearchOption{
				func(opts *blockstorage.VolumeSnapshotSearchOpts) error {
					callbacks.Add(1)
					opts.Location = &location
					opts.Filters = &filters
					retainedLocation, retainedFilters = opts.Location, opts.Filters
					search[1] = func(*blockstorage.VolumeSnapshotSearchOpts) error {
						return errors.New("replaced original should not run")
					}
					return nil
				},
				func(*blockstorage.VolumeSnapshotSearchOpts) error { return mutate() },
			}
			read = []blockstorage.VolumeSnapshotReadOption{
				func(opts *blockstorage.VolumeSnapshotReadOpts) error {
					callbacks.Add(1)
					opts.Location = &location
					retainedLocation = opts.Location
					read[1] = func(*blockstorage.VolumeSnapshotReadOpts) error {
						return errors.New("replaced original should not run")
					}
					return nil
				},
				func(*blockstorage.VolumeSnapshotReadOpts) error { return mutate() },
			}
			out, err := snapshotConnectionCall(context.Background(), conn, operation, "item", list, search, read)
			if err != nil || out == nil || callbacks.Load() != 2 || locates.Load() != 1 || reads.Load() != 1 {
				t.Fatal(out, err, callbacks.Load(), locates.Load(), reads.Load())
			}
			row := snapshotConnectionRow(t, operation, out.value)
			project := snapshotConnectionFields(t, snapshotConnectionFields(t, row["location"])["project"])
			if string(project["id"]) != `"owned-project"` {
				t.Fatal("retained original pointer changed prepared value", string(out.value))
			}
		})
	}
}

func TestConnectionVolumeSnapshotFailuresKeepOnlyActualPhysicalProofAndCustomCancellation(t *testing.T) {
	for _, operation := range snapshotConnectionOperations {
		t.Run(operation+"/descriptor", func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(opts)
				}
				return cloud.Server.URL + snapshotConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			body := `{"snapshot":{"id":"item","size":"²"}}`
			if operation == "ListVolumeSnapshots" || operation == "SearchVolumeSnapshots" {
				body = `{"snapshots":[{"id":"item","size":"²"}]}`
			}
			cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Actual", "accepted-descriptor-failure")
				testcloud.JSON(w, 203, body)
			})
			out, err := snapshotConnectionCall(context.Background(), conn, operation, "item", nil, nil, nil)
			var physical *resource.ResponseError
			if out == nil || err == nil || out.value != nil || out.selected != nil || out.rows != nil || !errors.As(err, &physical) || physical.StatusCode != 203 {
				t.Fatal(out, err, physical)
			}
			snapshotConnectionOperation(t, operation, err)
			proof := out.observed
			if operation == "ListVolumeSnapshots" || operation == "SearchVolumeSnapshots" {
				if len(out.pages) != 1 || proof != nil {
					t.Fatal(out)
				}
				proof = out.pages[0]
			} else if len(out.pages) != 0 {
				t.Fatal(out)
			}
			if proof == nil || proof.StatusCode != 203 || proof.Header.Get("X-Actual") != "accepted-descriptor-failure" || string(proof.Body) != body {
				t.Fatal(proof)
			}
		})
	}
	for _, operation := range []string{"ListVolumeSnapshots", "SearchVolumeSnapshots"} {
		for _, kind := range []string{"rejected second page", "blocked second page"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
					if opts.Type != "block-storage" || opts.Version != 3 {
						t.Error(opts)
					}
					return cloud.Server.URL + snapshotConnectionPath, nil
				}
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("snapshot second-page custom cancellation")
				entered := make(chan struct{})
				var requests atomic.Int32
				first := `{"snapshots":[{"id":"first"}],"next":"?marker=next"}`
				cloud.Mux.HandleFunc("GET "+snapshotConnectionPath+"snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.URL.RawQuery == "" {
						w.Header().Set("X-First", "admitted")
						testcloud.JSON(w, 203, first)
						return
					}
					if r.URL.RawQuery != "marker=next" {
						t.Error(r.URL)
					}
					if kind == "blocked second page" {
						close(entered)
						<-r.Context().Done()
						return
					}
					w.Header().Set("X-Rejected", "actual403")
					testcloud.JSON(w, 403, `{"error":"forbidden"}`)
				})
				type completed struct {
					out *snapshotConnectionOutcome
					err error
				}
				done := make(chan completed, 1)
				go func() {
					out, err := snapshotConnectionCall(ctx, conn, operation, "", nil, nil, nil)
					done <- completed{out, err}
				}()
				timer := time.NewTimer(5 * time.Second)
				defer timer.Stop()
				if kind == "blocked second page" {
					select {
					case <-entered:
						cancel(cause)
					case finished := <-done:
						t.Fatal("call returned before second page", finished.out, finished.err)
					case <-timer.C:
						cancel(cause)
						t.Fatal("second page not reached")
					}
				}
				var finished completed
				select {
				case finished = <-done:
				case <-timer.C:
					cancel(cause)
					t.Fatal("snapshot call did not finish")
				}
				out, err := finished.out, finished.err
				if out == nil || err == nil || out.value != nil || out.rows != nil || out.selected != nil || out.observed != nil || len(out.pages) != 1 || requests.Load() != 2 || out.pages[0].StatusCode != 203 || out.pages[0].Header.Get("X-First") != "admitted" || string(out.pages[0].Body) != first {
					t.Fatal(out, err, requests.Load())
				}
				snapshotConnectionOperation(t, operation, err)
				if kind == "blocked second page" {
					var physical *resource.ResponseError
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || errors.As(err, &physical) {
						t.Fatal("cancellation borrowed prior physical failure", err)
					}
				} else {
					var rejected gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &rejected) || rejected.Actual != 403 || rejected.ResponseHeader.Get("X-Rejected") != "actual403" {
						t.Fatal(err, rejected)
					}
				}
			})
		}
	}
}
