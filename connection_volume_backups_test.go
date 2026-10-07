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

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudbackup"
	"github.com/JSYoo5B/gophercloudsdk/internal/cloudsnapshot"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const backupConnectionPath = "/backup-connection/v3/project/"

var backupConnectionOperations = []string{"ListVolumeBackups", "SearchVolumeBackups", "GetVolumeBackup"}

type backupConnectionOutcome struct {
	value     json.RawMessage
	rows      []*resource.RawResource
	selected  *resource.RawResource
	pages     []*blockstorage.VolumeBackupsPage
	observed  *blockstorage.VolumeBackupsPage
	requested string
	seeded    bool
}

func backupConnectionCall(ctx context.Context, conn *sdk.Connection, operation, identity string, list []blockstorage.VolumeBackupListOption, search []blockstorage.VolumeBackupSearchOption) (*backupConnectionOutcome, error) {
	switch operation {
	case "ListVolumeBackups":
		result, err := conn.ListVolumeBackups(ctx, list...)
		if result == nil {
			return nil, err
		}
		return &backupConnectionOutcome{value: result.Value, rows: result.Backups, pages: result.Pages}, err
	case "SearchVolumeBackups":
		result, err := conn.SearchVolumeBackups(ctx, blockstorage.SearchVolumeBackupsRequest{NameOrID: identity}, search...)
		if result == nil {
			return nil, err
		}
		return &backupConnectionOutcome{value: result.Value, rows: result.Backups, pages: result.Pages}, err
	case "GetVolumeBackup":
		result, err := conn.GetVolumeBackup(ctx, blockstorage.GetVolumeBackupRequest{NameOrID: identity}, search...)
		if result == nil {
			return nil, err
		}
		return &backupConnectionOutcome{value: result.Value, selected: result.Backup, pages: result.Pages, observed: result.Observed, requested: result.RequestedID, seeded: result.SeededID}, err
	default:
		panic("unknown backup fixture operation")
	}
}

func backupConnectionCallbacks(callback func() error) ([]blockstorage.VolumeBackupListOption, []blockstorage.VolumeBackupSearchOption) {
	return []blockstorage.VolumeBackupListOption{func(*blockstorage.VolumeBackupListOpts) error { return callback() }},
		[]blockstorage.VolumeBackupSearchOption{func(*blockstorage.VolumeBackupSearchOpts) error { return callback() }}
}
func backupConnectionOperation(t *testing.T, operation string, err error) {
	t.Helper()
	var outer *resource.OperationError
	if !errors.As(err, &outer) || outer.Operation != operation || outer.Resource != "volume backup" {
		t.Fatal("backup operation lost", err, outer)
	}
}
func backupConnectionRow(t *testing.T, operation string, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	if operation != "GetVolumeBackup" {
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil || len(rows) != 1 {
			t.Fatal(string(raw), err)
		}
		raw = rows[0]
	}
	return snapshotConnectionFields(t, raw)
}
func backupConnectionEndpoint(t *testing.T, cloud *testcloud.Cloud) {
	t.Helper()
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		if opts.Type != "block-storage" || opts.Version != 3 {
			t.Error(opts)
		}
		return cloud.Server.URL + backupConnectionPath, nil
	}
}
func backupConnectionWire(t *testing.T, r *http.Request, path, token, version string) {
	t.Helper()
	if r.Method != "GET" || r.URL.Path != path || r.Header.Get("X-Auth-Token") != token || r.Header.Get("X-OpenStack-Volume-API-Version") != version {
		t.Error(r.Method, r.URL, r.Header)
	}
	if r.Body != nil {
		body, err := io.ReadAll(r.Body)
		if err != nil || len(body) != 0 {
			t.Error(string(body), err)
		}
	}
}

func TestConnectionVolumeBackupOriginalOptionsRunOnceBeforeCachedCinderForEveryEntry(t *testing.T) {
	for _, operation := range backupConnectionOperations {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var originals, locates, reads atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if originals.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(originals.Load(), opts)
				}
				return cloud.Server.URL + backupConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/", func(w http.ResponseWriter, r *http.Request) {
				call := reads.Add(1)
				path := backupConnectionPath + "backups/detail"
				if operation == "GetVolumeBackup" {
					path = backupConnectionPath + "backups/item"
				}
				backupConnectionWire(t, r, path, "original-token", "3.60")
				if r.URL.RawQuery != "" || (call == 2 && r.Header.Get("X-Original") != "second-call") {
					t.Error(r.URL, r.Header)
				}
				row := `{"id":"item","name":"item","force":"false","is_incremental":{},"size":"7","object_count":"9","links":{"opaque":1},"unknown":9007199254740993}`
				if operation == "GetVolumeBackup" {
					row = strings.Replace(row, `"id":"item"`, `"id":"response-item"`, 1)
					row = `{"backup":` + row + `}`
				} else {
					row = `{"backups":[` + row + `]}`
				}
				w.Header().Set("X-Backup-Proof", "owned")
				testcloud.JSON(w, 203, row)
			})
			var cached *gophercloud.ServiceClient
			for call := 0; call < 2; call++ {
				list, search := backupConnectionCallbacks(func() error {
					originals.Add(1)
					if cached != nil {
						cached.MoreHeaders = map[string]string{"X-Original": "second-call"}
					}
					return snapshotConnectionRecordScope(cloud.Provider, "original-project", "original-token")
				})
				out, err := backupConnectionCall(context.Background(), conn, operation, "item", list, search)
				if err != nil || out == nil {
					t.Fatal(out, err)
				}
				row := backupConnectionRow(t, operation, out.value)
				if len(row) != 24 || string(row["force"]) != "true" || string(row["is_incremental"]) != "false" || string(row["size"]) != "7" || string(row["object_count"]) != "9" || string(row["links"]) != `[{"opaque":1}]` || row["consumes_quota"] != nil {
					t.Fatal("snapshot model leaked into backup", string(out.value))
				}
				proof := out.observed
				selected := out.selected
				if operation != "GetVolumeBackup" {
					if len(out.rows) != 1 || len(out.pages) != 1 || proof != nil {
						t.Fatal(out)
					}
					proof, selected = out.pages[0], out.rows[0]
				} else if len(out.pages) != 0 || out.requested != "item" || out.seeded || string(row["id"]) != `"response-item"` {
					t.Fatal(out, string(out.value))
				}
				if proof == nil || selected == nil || proof.StatusCode != 203 || proof.Header.Get("X-Backup-Proof") != "owned" || string(selected.Body["unknown"]) != "9007199254740993" {
					t.Fatal(proof, selected)
				}
				saved := bytes.Clone(proof.Body)
				selected.Body["unknown"][0] = '!'
				selected.Header.Set("X-Backup-Proof", "changed")
				out.value[0] = '!'
				if !bytes.Equal(saved, proof.Body) || proof.Header.Get("X-Backup-Proof") != "owned" {
					t.Fatal("raw/normalized result changed physical proof", proof)
				}
				service, err := conn.BlockStorageV3(context.Background())
				if err != nil || (cached != nil && service.RawClient() != cached) {
					t.Fatal(service, err)
				}
				cached = service.RawClient()
			}
			if originals.Load() != 2 || locates.Load() != 1 || reads.Load() != 2 {
				t.Fatal(originals.Load(), locates.Load(), reads.Load())
			}
		})
	}
}

func TestConnectionVolumeBackupLocationIsCapturedAfterOriginalsBeforeGetterAndKeepsRawZoneAcrossLiveAuthPages(t *testing.T) {
	for _, operation := range []string{"ListVolumeBackups", "SearchVolumeBackups"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			if err := snapshotConnectionRecordScope(cloud.Provider, "before-original", "before-token"); err != nil {
				t.Fatal(err)
			}
			var originals, locates, reads atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if originals.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(originals.Load(), opts)
				}
				if err := snapshotConnectionRecordScope(cloud.Provider, "at-getter", "getter-token"); err != nil {
					return "", err
				}
				return cloud.Server.URL + backupConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("recorded-region"))
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/detail", func(w http.ResponseWriter, r *http.Request) {
				call := reads.Add(1)
				if r.URL.RawQuery == "" {
					token := "getter-token"
					if call > 2 {
						token = "page-token"
					}
					backupConnectionWire(t, r, backupConnectionPath+"backups/detail", token, "")
					if err := snapshotConnectionRecordScope(cloud.Provider, "between-pages", "page-token"); err != nil {
						t.Error(err)
					}
					testcloud.JSON(w, 200, `{"backups":[{"id":"first","availability_zone":{"opaque":9007199254740993}}],"backups_links":[{"rel":"next","href":"?marker=next"}]}`)
				} else {
					backupConnectionWire(t, r, backupConnectionPath+"backups/detail", "page-token", "")
					if r.URL.Query().Get("marker") != "next" || len(r.URL.Query()) != 1 {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, `{"backups":[{"id":"second","availability_zone":false}]}`)
				}
			})
			for call := 0; call < 2; call++ {
				list, search := backupConnectionCallbacks(func() error {
					originals.Add(1)
					if call == 0 {
						return snapshotConnectionRecordScope(cloud.Provider, "after-original", "original-token")
					}
					return nil
				})
				out, err := backupConnectionCall(context.Background(), conn, operation, "", list, search)
				if err != nil || out == nil || len(out.rows) != 2 || len(out.pages) != 2 {
					t.Fatal(out, err)
				}
				var rows []json.RawMessage
				if err := json.Unmarshal(out.value, &rows); err != nil || len(rows) != 2 {
					t.Fatal(string(out.value), err)
				}
				wantProject := `"after-original"`
				if call == 1 {
					wantProject = `"between-pages"`
				}
				for index, row := range rows {
					fields := snapshotConnectionFields(t, row)
					location := snapshotConnectionFields(t, fields["location"])
					project := snapshotConnectionFields(t, location["project"])
					wantZone := `{"opaque":9007199254740993}`
					if index == 1 {
						wantZone = "false"
					}
					if string(project["id"]) != wantProject || string(project["name"]) != "null" || string(location["region_name"]) != `"recorded-region"` || string(location["zone"]) != wantZone {
						t.Fatal("location recaptured after service/auth or lost backup zone", string(row))
					}
				}
				current, err := conn.CurrentLocation()
				if err != nil || string(current.Project.ID) != `"between-pages"` {
					t.Fatal(current, err)
				}
				current.Project.ID[0] = '!'
			}
			if originals.Load() != 2 || locates.Load() != 1 || reads.Load() != 4 {
				t.Fatal(originals.Load(), locates.Load(), reads.Load())
			}
		})
	}
}

func TestConnectionVolumeBackupExplicitLocationOwnsEveryFieldAndUsesBackupProjectAndZone(t *testing.T) {
	for _, operation := range backupConnectionOperations {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			defaultCloud, defaultName := "default-cloud", "default-name"
			defaults := resource.CloudLocation{Cloud: &defaultCloud, Project: resource.CloudProject{ID: json.RawMessage(`"default-project"`), Name: &defaultName}}
			cloudName, region, name, domain := "owned-cloud", "owned-region", "owned-name", "owned-domain"
			location := resource.CloudLocation{Cloud: &cloudName, RegionName: &region, Zone: json.RawMessage(`invalid-unused-zone`), Project: resource.CloudProject{ID: json.RawMessage(`"owned-project"`), Name: &name, DomainName: &domain}}
			list := []blockstorage.VolumeBackupListOption{blockstorage.WithVolumeBackupListLocation(location)}
			search := []blockstorage.VolumeBackupSearchOption{blockstorage.WithVolumeBackupSearchLocation(location)}
			var reads atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(opts)
				}
				cloudName, region, name, domain = "changed", "changed", "changed", "changed"
				location.Project.ID[0] = '!'
				location.Zone[0] = '!'
				return cloud.Server.URL + backupConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(defaults))
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/", func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				row := `{"id":"item","name":"item","os-backup-project-attr:project_id":"owned-project","project_id":"owned-project","location":"wire-ignored","availability_zone":["actual-zone",9007199254740993]}`
				if operation == "GetVolumeBackup" {
					row = `{"backup":` + row + `}`
				} else {
					row = `{"backups":[` + row + `]}`
				}
				testcloud.JSON(w, 200, row)
			})
			out, err := backupConnectionCall(context.Background(), conn, operation, "item", list, search)
			if err != nil || out == nil || reads.Load() != 1 {
				t.Fatal(out, err, reads.Load())
			}
			row := backupConnectionRow(t, operation, out.value)
			normalized := snapshotConnectionFields(t, row["location"])
			project := snapshotConnectionFields(t, normalized["project"])
			if string(normalized["cloud"]) != `"owned-cloud"` || string(normalized["region_name"]) != `"owned-region"` || string(normalized["zone"]) != `["actual-zone",9007199254740993]` || string(project["id"]) != `"owned-project"` || string(project["name"]) != `"owned-name"` || string(project["domain_name"]) != `"owned-domain"` {
				t.Fatal(string(out.value))
			}
			current, err := conn.CurrentLocation()
			if err != nil || current.Cloud == nil || *current.Cloud != "default-cloud" || string(current.Project.ID) != `"default-project"` || current.Project.Name == nil || *current.Project.Name != "default-name" {
				t.Fatal("override changed connection defaults", current, err)
			}
		})
	}
}

func TestConnectionVolumeBackupListControlsStayPerCallAndDoNotLeakSnapshotQueryOrModel(t *testing.T) {
	cloud := testcloud.New(t)
	var originals, locates, reads atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if originals.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
			t.Error(originals.Load(), opts)
		}
		return cloud.Server.URL + backupConnectionPath, nil
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		backupConnectionWire(t, r, backupConnectionPath+"backups", "test-token", "3.41")
		if r.URL.Query().Get("all_tenants") != "False" || r.URL.Query().Get("limit") != "1" || len(r.URL.Query()) != 2 || r.Header.Get("X-Trace") != "owned-header" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 203, `{"backups":[{"id":"first","force":"false","description":"keep"},{"object_count":"²"}],"links":[null]}`)
	})
	cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/detail", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		backupConnectionWire(t, r, backupConnectionPath+"backups/detail", "test-token", "3.60")
		if r.URL.RawQuery != "" || r.Header.Get("X-Trace") != "" {
			t.Error(r.URL, r.Header)
		}
		testcloud.JSON(w, 200, `{"backups":[{"id":"backup-default"}]}`)
	})
	cloud.Mux.HandleFunc("GET "+backupConnectionPath+"snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
		reads.Add(1)
		backupConnectionWire(t, r, backupConnectionPath+"snapshots/detail", "test-token", "3.60")
		if r.URL.RawQuery != "" {
			t.Error("snapshot Proxy must drop falsey all_projects", r.URL)
		}
		testcloud.JSON(w, 200, `{"snapshots":[{"id":"snapshot-default","force":"false","availability_zone":"must-not-be-zone"}]}`)
	})
	headers := map[string]string{"X-Trace": "owned-header"}
	headerOption := blockstorage.WithVolumeBackupListHeaders(headers)
	headers["X-Trace"] = "caller-mutated"
	result, err := conn.ListVolumeBackups(context.Background(), func(*blockstorage.VolumeBackupListOpts) error { originals.Add(1); return nil }, blockstorage.WithVolumeBackupListDetailed(false), blockstorage.WithVolumeBackupListPagination(false), blockstorage.WithVolumeBackupListMaxItems(1), blockstorage.WithVolumeBackupListMicroversion("3.41"), headerOption, blockstorage.WithVolumeBackupListFilters(json.RawMessage(`{"all_projects":false,"limit":1,"force":true,"description":"keep","consumes_quota":false}`)))
	if err != nil || result == nil || len(result.Backups) != 1 || len(result.Pages) != 1 || string(result.Backups[0].Body["id"]) != `"first"` {
		t.Fatal(result, err)
	}
	service, err := conn.BlockStorageV3(context.Background())
	if err != nil || service.RawClient().Microversion != "3.60" || len(service.RawClient().MoreHeaders) != 0 {
		t.Fatal("list policy altered cached source", service, err)
	}
	result, err = conn.ListVolumeBackups(context.Background())
	if err != nil || result == nil || len(result.Backups) != 1 || string(result.Backups[0].Body["id"]) != `"backup-default"` {
		t.Fatal(result, err)
	}
	snapshots, err := conn.ListVolumeSnapshots(context.Background(), blockstorage.WithVolumeSnapshotListFilters(json.RawMessage(`{"all_projects":false}`)))
	if err != nil || snapshots == nil || len(snapshots.Snapshots) != 1 {
		t.Fatal(snapshots, err)
	}
	fields := snapshotConnectionRow(t, "ListVolumeSnapshots", snapshots.Value)
	location := snapshotConnectionFields(t, fields["location"])
	if fields["force"] != nil || string(fields["is_forced"]) != "false" || string(location["zone"]) != "null" || originals.Load() != 1 || locates.Load() != 1 || reads.Load() != 3 {
		t.Fatal("backup config leaked into snapshot", string(snapshots.Value), originals.Load(), locates.Load(), reads.Load())
	}
}

func TestConnectionVolumeBackupPreflightAndGetterFailuresKeepBackupOperationAndJoinedCauses(t *testing.T) {
	for _, operation := range backupConnectionOperations {
		for _, kind := range []string{"nil context", "nil connection", "already canceled", "nil option", "original error", "original cancellation", "getter error", "getter cancellation", "invalid location"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, originalErr, getterErr := errors.New("backup custom cause"), errors.New("backup original failed"), errors.New("backup Cinder lookup failed")
				var originals, locates atomic.Int32
				cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
					locates.Add(1)
					if originals.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
						t.Error(originals.Load(), opts)
					}
					if kind == "getter cancellation" {
						cancel(cause)
					}
					return "", getterErr
				}
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				list, search := backupConnectionCallbacks(func() error {
					originals.Add(1)
					if kind == "original cancellation" {
						cancel(cause)
						return originalErr
					}
					if kind == "original error" {
						return originalErr
					}
					return nil
				})
				want, callbacks, getters := resource.ErrInvalidOption, int32(0), int32(0)
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
					callbacks = 1
				case "original error":
					want = originalErr
					callbacks = 1
				case "original cancellation":
					want = context.Canceled
					callbacks = 1
				case "invalid location":
					invalid := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`invalid`)}}
					list = append(list, blockstorage.WithVolumeBackupListLocation(invalid))
					search = append(search, blockstorage.WithVolumeBackupSearchLocation(invalid))
					callbacks = 1
				default:
					want = getterErr
					callbacks, getters = 1, 1
				}
				out, err := backupConnectionCall(ctx, conn, operation, "item", list, search)
				if out != nil || !errors.Is(err, want) || originals.Load() != callbacks || locates.Load() != getters {
					t.Fatal(out, err, originals.Load(), locates.Load())
				}
				backupConnectionOperation(t, operation, err)
				var response *resource.ResponseError
				if errors.As(err, &response) {
					t.Fatal("local/getter failure invented HTTP proof", err)
				}
				if kind == "original cancellation" && (!errors.Is(err, originalErr) || !errors.Is(err, cause)) || kind == "getter cancellation" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, getterErr)) || kind == "already canceled" && !errors.Is(err, cause) {
					t.Fatal("joined cause lost", err)
				}
			})
		}
	}
}

func TestConnectionVolumeBackupListBindingValidationPrecedesServiceSelection(t *testing.T) {
	for _, raw := range []string{`{"details":true}`, `{"base_path":"/snapshots"}`, `{"resource_type":null}`, `{"self":false}`, `{"__conflicting_attrs":{"paginated":false}}`, `{"__conflicting_attrs":{"session":null}}`, `{"headers":{"X-Test":1}}`, `{"max_items":"bad"}`, `{"microversion":false}`} {
		t.Run(raw, func(t *testing.T) {
			cloud := testcloud.New(t)
			var originals, locates atomic.Int32
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				return "", errors.New("getter must not run")
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			out, err := conn.ListVolumeBackups(context.Background(), func(*blockstorage.VolumeBackupListOpts) error { originals.Add(1); return nil }, blockstorage.WithVolumeBackupListFilters(json.RawMessage(raw)))
			if out != nil || !errors.Is(err, resource.ErrInvalidOption) || originals.Load() != 1 || locates.Load() != 0 {
				t.Fatal(out, err, originals.Load(), locates.Load())
			}
			backupConnectionOperation(t, "ListVolumeBackups", err)
		})
	}
}

func TestConnectionVolumeBackupCapturedOptionSliceAndRetainedPointersCannotReplacePreparedPolicy(t *testing.T) {
	for _, operation := range backupConnectionOperations {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var originals, locates, reads atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if originals.Load() != 2 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(originals.Load(), opts)
				}
				return cloud.Server.URL + backupConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/", func(w http.ResponseWriter, r *http.Request) {
				reads.Add(1)
				if operation == "ListVolumeBackups" {
					if r.URL.Query().Get("all_tenants") != "False" || len(r.URL.Query()) != 1 || r.Header.Get("X-Retained") != "owned" {
						t.Error(r.URL, r.Header)
					}
				} else if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"backups":[{"id":"item","force":"false"}]}`)
			})
			location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"owned-project"`)}}
			filters := json.RawMessage(`{"all_projects":false,"force":true}`)
			if operation != "ListVolumeBackups" {
				filters = json.RawMessage(`{"force":true}`)
			}
			headers := map[string]string{"X-Retained": "owned"}
			var retainedLocation *resource.CloudLocation
			var retainedFilters *json.RawMessage
			var retainedHeaders map[string]string
			mutate := func() error {
				originals.Add(1)
				retainedLocation.Project.ID[1] = 'X'
				(*retainedFilters)[0] = '!'
				if retainedHeaders != nil {
					retainedHeaders["X-Retained"] = "mutated"
				}
				return nil
			}
			var list []blockstorage.VolumeBackupListOption
			list = []blockstorage.VolumeBackupListOption{func(opts *blockstorage.VolumeBackupListOpts) error {
				originals.Add(1)
				opts.Location = &location
				opts.Filters = &filters
				opts.Headers = headers
				retainedLocation, retainedFilters, retainedHeaders = opts.Location, opts.Filters, opts.Headers
				list[1] = func(*blockstorage.VolumeBackupListOpts) error { return errors.New("replacement must not execute") }
				return nil
			}, func(*blockstorage.VolumeBackupListOpts) error { return mutate() }}
			var search []blockstorage.VolumeBackupSearchOption
			search = []blockstorage.VolumeBackupSearchOption{func(opts *blockstorage.VolumeBackupSearchOpts) error {
				originals.Add(1)
				opts.Location = &location
				opts.Filters = &filters
				retainedLocation, retainedFilters = opts.Location, opts.Filters
				search[1] = func(*blockstorage.VolumeBackupSearchOpts) error { return errors.New("replacement must not execute") }
				return nil
			}, func(*blockstorage.VolumeBackupSearchOpts) error { return mutate() }}
			out, err := backupConnectionCall(context.Background(), conn, operation, "item", list, search)
			if err != nil || out == nil || originals.Load() != 2 || locates.Load() != 1 || reads.Load() != 1 {
				t.Fatal(out, err, originals.Load(), locates.Load(), reads.Load())
			}
			row := backupConnectionRow(t, operation, out.value)
			project := snapshotConnectionFields(t, snapshotConnectionFields(t, row["location"])["project"])
			if string(project["id"]) != `"owned-project"` {
				t.Fatal("retained callback pointers changed policy", string(out.value))
			}
		})
	}
}

func TestConnectionVolumeBackupMemberSeedsOnlyLogicalMissingIDAndCleanFallbackKeepsListProof(t *testing.T) {
	for _, tc := range []struct {
		name, body, id string
		status         int
		seeded         bool
	}{
		{"missing canonical", `{"backup":{"name":"item","unknown":9007199254740993}}`, `"item"`, 203, true},
		{"empty", "", `"item"`, 204, true}, {"malformed", `{malformed`, `"item"`, 200, true},
		{"explicit null", `{"backup":{"id":null}}`, `null`, 200, false}, {"explicit object", `{"id":{"actual":9007199254740993}}`, `{"actual":9007199254740993}`, 200, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			backupConnectionEndpoint(t, cloud)
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			var members, pages atomic.Int32
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/item", func(w http.ResponseWriter, r *http.Request) {
				members.Add(1)
				backupConnectionWire(t, r, backupConnectionPath+"backups/item", "test-token", "")
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				w.Header().Set("X-Seed", "actual")
				testcloud.JSON(w, tc.status, tc.body)
			})
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/detail", func(w http.ResponseWriter, r *http.Request) { pages.Add(1); testcloud.JSON(w, 200, `{"backups":[]}`) })
			result, err := conn.GetVolumeBackup(context.Background(), blockstorage.GetVolumeBackupRequest{NameOrID: "item"})
			if err != nil || result == nil || result.Backup == nil || result.Observed == nil || result.Observed.StatusCode != tc.status || result.Observed.Header.Get("X-Seed") != "actual" || result.RequestedID != "item" || result.SeededID != tc.seeded || len(result.Pages) != 0 || members.Load() != 1 || pages.Load() != 0 {
				t.Fatal(result, err, members.Load(), pages.Load())
			}
			row := snapshotConnectionFields(t, result.Value)
			_, wireHasID := result.Backup.Body["id"]
			if string(row["id"]) != tc.id || wireHasID == tc.seeded || string(result.Observed.Body) != tc.body {
				t.Fatal("logical seed contaminated physical body", result, string(result.Value))
			}
		})
	}
	cloud := testcloud.New(t)
	backupConnectionEndpoint(t, cloud)
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	var members, pages atomic.Int32
	cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/item", func(w http.ResponseWriter, r *http.Request) {
		members.Add(1)
		w.Header().Set("X-Rejected", "member404")
		testcloud.JSON(w, 404, `{"error":"missing"}`)
	})
	cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/detail", func(w http.ResponseWriter, r *http.Request) {
		pages.Add(1)
		if r.URL.Query().Get("name") != "item" || len(r.URL.Query()) != 1 {
			t.Error(r.URL)
		}
		w.Header().Set("X-List", "accepted")
		testcloud.JSON(w, 203, `{"backups":[]}`)
	})
	result, err := conn.GetVolumeBackup(context.Background(), blockstorage.GetVolumeBackupRequest{NameOrID: "item"})
	if err != nil || result == nil || result.Value != nil || result.Backup != nil || result.Observed != nil || len(result.Pages) != 1 || result.Pages[0].StatusCode != 203 || result.Pages[0].Header.Get("X-List") != "accepted" || result.RequestedID != "" || result.SeededID || members.Load() != 1 || pages.Load() != 1 {
		t.Fatal(result, err, members.Load(), pages.Load())
	}
}

func TestConnectionVolumeBackupSelectionErrorRetainsBackupIdentityWithoutChangingSnapshotErrorShape(t *testing.T) {
	for _, expression := range []bool{false, true} {
		t.Run(map[bool]string{false: "nonnull falsey filters", true: "arbitrary expression"}[expression], func(t *testing.T) {
			cloud := testcloud.New(t)
			backupConnectionEndpoint(t, cloud)
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			var backupReads, snapshotReads, members atomic.Int32
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/detail", func(w http.ResponseWriter, r *http.Request) {
				backupReads.Add(1)
				if r.URL.RawQuery != "" {
					t.Error(r.URL)
				}
				testcloud.JSON(w, 200, `{"backups":[{"id":"first"},{"id":"second"}]}`)
			})
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"snapshots/detail", func(w http.ResponseWriter, r *http.Request) {
				snapshotReads.Add(1)
				testcloud.JSON(w, 200, `{"snapshots":[{"id":"first"},{"id":"second"}]}`)
			})
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/item", func(w http.ResponseWriter, r *http.Request) {
				members.Add(1)
				testcloud.JSON(w, 500, `{"error":"must not use member"}`)
			})
			backupOpts := []blockstorage.VolumeBackupSearchOption{blockstorage.WithVolumeBackupSearchFilters(json.RawMessage(`false`))}
			snapshotOpts := []blockstorage.VolumeSnapshotSearchOption{blockstorage.WithVolumeSnapshotSearchFilters(json.RawMessage(`false`))}
			if expression {
				backupOpts = []blockstorage.VolumeBackupSearchOption{blockstorage.WithVolumeBackupSearchExpression(`[].id`)}
				snapshotOpts = []blockstorage.VolumeSnapshotSearchOption{blockstorage.WithVolumeSnapshotSearchExpression(`[].id`)}
			}
			result, err := conn.GetVolumeBackup(context.Background(), blockstorage.GetVolumeBackupRequest{}, backupOpts...)
			var selected *cloudbackup.SelectionError
			var wrong *cloudsnapshot.SelectionError
			if result == nil || result.Value != nil || result.Backup != nil || result.Observed != nil || len(result.Pages) != 1 || !errors.Is(err, resource.ErrAmbiguous) || !errors.As(err, &selected) || selected.NameOrID != "" || selected.Length != 2 || errors.As(err, &wrong) || !strings.Contains(err.Error(), "backup") {
				t.Fatal(result, err, selected, wrong)
			}
			backupConnectionOperation(t, "GetVolumeBackup", err)
			snapshot, err := conn.GetVolumeSnapshot(context.Background(), blockstorage.GetVolumeSnapshotRequest{}, snapshotOpts...)
			var old *cloudsnapshot.SelectionError
			selected = nil
			if snapshot == nil || !errors.As(err, &old) || old.NameOrID != "" || old.Length != 2 || errors.As(err, &selected) || backupReads.Load() != 1 || snapshotReads.Load() != 1 || members.Load() != 0 {
				t.Fatal("backup error changed snapshot public shape", snapshot, err, old, selected, backupReads.Load(), snapshotReads.Load(), members.Load())
			}
		})
	}
}

type backupConnectionTransport func(*http.Request) (*http.Response, error)

func (f backupConnectionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type backupConnectionFaultBody struct {
	reader            *strings.Reader
	readErr, closeErr error
	onClose           func()
}

func (b *backupConnectionFaultBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if err == io.EOF {
		return n, b.readErr
	}
	return n, err
}
func (b *backupConnectionFaultBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

func TestConnectionVolumeBackupAcceptedIOJoinsReadCloseCancellationAndSourceDriftWithCurrentProof(t *testing.T) {
	for _, operation := range backupConnectionOperations {
		for _, kind := range []string{"IO and cancellation", "provider", "endpoint", "resource base", "type", "microversion"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				backupConnectionEndpoint(t, cloud)
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				service, err := conn.BlockStorageV3(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				source := service.RawClient()
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readErr, closeErr, cause := errors.New("backup accepted read"), errors.New("backup accepted Close"), errors.New("backup Close canceled caller")
				var requests atomic.Int32
				body := `{"backups":[{"id":"item"}]}`
				path := backupConnectionPath + "backups/detail"
				if operation == "GetVolumeBackup" {
					body = `{"backup":{"id":"item"}}`
					path = backupConnectionPath + "backups/item"
				}
				cloud.Provider.HTTPClient.Transport = backupConnectionTransport(func(r *http.Request) (*http.Response, error) {
					requests.Add(1)
					backupConnectionWire(t, r, path, "test-token", "")
					fault := &backupConnectionFaultBody{reader: strings.NewReader(body)}
					fault.onClose = func() {
						switch kind {
						case "IO and cancellation":
							cancel(cause)
							source.Type = "compute"
						case "provider":
							copy := *source.ProviderClient
							source.ProviderClient = &copy
						case "endpoint":
							source.Endpoint += "changed/"
						case "resource base":
							source.ResourceBase = source.Endpoint + "changed/"
						case "type":
							source.Type = "compute"
						case "microversion":
							source.Microversion = "3.61"
						}
					}
					if kind == "IO and cancellation" {
						fault.readErr, fault.closeErr = readErr, closeErr
					} else {
						fault.readErr = io.EOF
					}
					return &http.Response{StatusCode: 203, Header: http.Header{"Content-Type": {"application/json"}, "X-Actual": {"current-backup"}}, Body: fault, Request: r}, nil
				})
				out, err := backupConnectionCall(ctx, conn, operation, "item", nil, nil)
				var physical *resource.ResponseError
				if out == nil || out.value != nil || out.rows != nil || out.selected != nil || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &physical) || physical.StatusCode != 203 || physical.Header.Get("X-Actual") != "current-backup" || !bytes.Equal(physical.Body, []byte(body)) || requests.Load() != 1 {
					t.Fatal(out, err, physical, requests.Load())
				}
				backupConnectionOperation(t, operation, err)
				proof := out.observed
				if operation != "GetVolumeBackup" {
					if len(out.pages) != 1 || proof != nil {
						t.Fatal(out)
					}
					proof = out.pages[0]
				} else if len(out.pages) != 0 {
					t.Fatal(out)
				}
				if proof == nil || proof.StatusCode != 203 || string(proof.Body) != body || proof.Header.Get("X-Actual") != "current-backup" {
					t.Fatal(proof)
				}
				if kind == "IO and cancellation" && (!errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
					t.Fatal("accepted failure lost one independent cause", err)
				}
				physical.Header.Set("X-Actual", "changed")
				physical.Body[0] = '!'
				if proof.Header.Get("X-Actual") != "current-backup" || string(proof.Body) != body {
					t.Fatal("error evidence shared result storage", proof)
				}
			})
		}
	}
}

func TestConnectionVolumeBackupLaterFailureRetainsPagesWithoutBorrowingEarlierMemberEvidence(t *testing.T) {
	for _, operation := range backupConnectionOperations {
		for _, kind := range []string{"native second rejection", "second transport cancellation"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				backupConnectionEndpoint(t, cloud)
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause := errors.New("backup later request cancellation")
				first := `{"backups":[{"id":"first"}],"next":"?marker=next"}`
				var members, pages atomic.Int32
				cloud.Provider.HTTPClient.Transport = backupConnectionTransport(func(r *http.Request) (*http.Response, error) {
					if r.URL.Path == backupConnectionPath+"backups/item" {
						members.Add(1)
						return &http.Response{StatusCode: 404, Header: http.Header{"Content-Type": {"application/json"}, "X-Member": {"rejected"}}, Body: io.NopCloser(strings.NewReader(`{"error":"missing"}`)), Request: r}, nil
					}
					pages.Add(1)
					if r.URL.Path != backupConnectionPath+"backups/detail" || r.Method != "GET" {
						t.Error(r.Method, r.URL)
					}
					query := r.URL.Query()
					if operation == "GetVolumeBackup" && query.Get("name") != "item" {
						t.Error(r.URL)
					}
					if query.Get("marker") == "" {
						return &http.Response{StatusCode: 203, Header: http.Header{"Content-Type": {"application/json"}, "X-First": {"accepted"}}, Body: io.NopCloser(strings.NewReader(first)), Request: r}, nil
					}
					if query.Get("marker") != "next" {
						t.Error(r.URL)
					}
					if kind == "second transport cancellation" {
						cancel(cause)
						return nil, ctx.Err()
					}
					return &http.Response{StatusCode: 403, Header: http.Header{"Content-Type": {"application/json"}, "X-Rejected": {"second403"}}, Body: io.NopCloser(strings.NewReader(`{"error":"forbidden"}`)), Request: r}, nil
				})
				identity := ""
				if operation == "GetVolumeBackup" {
					identity = "item"
				}
				out, err := backupConnectionCall(ctx, conn, operation, identity, nil, nil)
				var physical *resource.ResponseError
				wantMembers := int32(0)
				if operation == "GetVolumeBackup" {
					wantMembers = 1
				}
				if out == nil || err == nil || out.value != nil || out.rows != nil || out.selected != nil || out.observed != nil || len(out.pages) != 1 || out.pages[0].StatusCode != 203 || out.pages[0].Header.Get("X-First") != "accepted" || string(out.pages[0].Body) != first || errors.As(err, &physical) || members.Load() != wantMembers || pages.Load() != 2 {
					t.Fatal(out, err, physical, members.Load(), pages.Load())
				}
				backupConnectionOperation(t, operation, err)
				if kind == "second transport cancellation" {
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
						t.Fatal("later cancellation lost cause", err)
					}
				} else {
					var rejected gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &rejected) || rejected.Actual != 403 || rejected.ResponseHeader.Get("X-Rejected") != "second403" {
						t.Fatal(err, rejected)
					}
				}
			})
		}
	}
}
