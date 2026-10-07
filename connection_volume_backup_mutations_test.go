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

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type backupMutationConnectionOutcome struct {
	create *blockstorage.CreateVolumeBackupResult
	delete *blockstorage.DeleteVolumeBackupResult
}

type backupMutationConnectionTransport func(*http.Request) (*http.Response, error)

type backupMutationConnectionFaultBody struct {
	reader                   *strings.Reader
	readErr, closeErr, cause error
	cancel                   context.CancelCauseFunc
}

func backupMutationConnectionCall(ctx context.Context, conn *sdk.Connection, family string, create []blockstorage.CreateVolumeBackupOption, deleted []blockstorage.DeleteVolumeBackupOption) (*backupMutationConnectionOutcome, error) {
	if family == "create" {
		value, err := conn.CreateVolumeBackup(ctx, blockstorage.CreateVolumeBackupRequest{VolumeID: "literal/volume"}, create...)
		if value == nil {
			return nil, err
		}
		return &backupMutationConnectionOutcome{create: value}, err
	}
	value, err := conn.DeleteVolumeBackup(ctx, blockstorage.DeleteVolumeBackupRequest{NameOrID: "requested"}, deleted...)
	if value == nil {
		return nil, err
	}
	return &backupMutationConnectionOutcome{delete: value}, err
}

func backupMutationConnectionName(family string) string {
	if family == "create" {
		return "CreateVolumeBackup"
	}
	return "DeleteVolumeBackup"
}

func backupMutationConnectionWire(t *testing.T, r *http.Request, method, path, token string) []byte {
	t.Helper()
	var body []byte
	var err error
	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
	}
	if err != nil || r.Method != method || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("X-OpenStack-Volume-API-Version") != "3.60" || method != http.MethodPost && len(body) != 0 {
		t.Error("fixed Cinder wire policy", r.Method, r.URL, r.Header, string(body), err)
	}
	return body
}

func backupMutationConnectionLocation(t *testing.T, raw json.RawMessage, project string) {
	t.Helper()
	location := snapshotConnectionFields(t, snapshotConnectionFields(t, raw)["location"])
	scope := snapshotConnectionFields(t, location["project"])
	if string(scope["id"]) != project {
		t.Fatal("call scope changed", string(raw))
	}
}

func TestConnectionBackupMutationsOriginalsPrecedeOnlyCinderV3AndReuseCacheWithoutReadiness(t *testing.T) {
	for _, family := range []string{"create", "delete"} {
		t.Run(family, func(t *testing.T) {
			cloud := testcloud.New(t)
			var callbacks, locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("originals/Cinder v3", callbacks.Load(), opts)
				}
				return cloud.Server.URL + backupConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			var cached *gophercloud.ServiceClient
			cloud.Mux.HandleFunc("POST "+backupConnectionPath+"backups", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				body := backupMutationConnectionWire(t, r, "POST", backupConnectionPath+"backups", "backup-live-token")
				fields := snapshotConnectionFields(t, snapshotConnectionFields(t, body)["backup"])
				if family != "create" || len(fields) != 6 || string(fields["name"]) != "null" || string(fields["description"]) != "null" || string(fields["snapshot_id"]) != "null" || string(fields["incremental"]) != "false" || fields["is_incremental"] != nil || string(fields["volume_id"]) != `"literal/volume"` || string(fields["force"]) != "false" {
					t.Error(string(body))
				}
				if callbacks.Load() == 2 && r.Header.Get("X-Original") != "second-owned" {
					t.Error(r.Header)
				}
				w.Header().Set("X-Phase", "created")
				testcloud.JSON(w, 203, `{"backup":{"id":"created","status":"creating","unknown":900719925474099312345}}`)
			})
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/requested", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				backupMutationConnectionWire(t, r, "GET", backupConnectionPath+"backups/requested", "backup-live-token")
				if family != "delete" || callbacks.Load() == 2 && r.Header.Get("X-Original") != "second-owned" {
					t.Error(family, r.Header)
				}
				testcloud.JSON(w, 200, `{"backup":{"id":"selected","status":"available"}}`)
			})
			cloud.Mux.HandleFunc("DELETE "+backupConnectionPath+"backups/selected", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				backupMutationConnectionWire(t, r, "DELETE", backupConnectionPath+"backups/selected", "backup-live-token")
				if family != "delete" || callbacks.Load() == 2 && r.Header.Get("X-Original") != "second-owned" {
					t.Error(family, r.Header)
				}
				w.Header().Set("X-Phase", "deleted")
				testcloud.JSON(w, 202, "opaque acknowledgement")
			})
			for range 2 {
				original := func() error {
					callbacks.Add(1)
					cloud.Provider.SetToken("backup-live-token")
					if cached != nil {
						cached.MoreHeaders = map[string]string{"X-Original": "second-owned"}
					}
					return nil
				}
				create := []blockstorage.CreateVolumeBackupOption{func(next *blockstorage.CreateVolumeBackupOpts) error { next.Wait = new(bool); return original() }}
				deleted := []blockstorage.DeleteVolumeBackupOption{func(*blockstorage.DeleteVolumeBackupOpts) error { return original() }}
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				out, err := backupMutationConnectionCall(ctx, conn, family, create, deleted)
				cancel()
				if err != nil || out == nil {
					t.Fatal(out, err)
				}
				if family == "create" {
					value := out.create
					if value.Value == nil || value.CreatedValue == nil || value.Created == nil || value.LastAccepted == nil || value.CreatedBackup == nil || value.Backup == nil || value.Ready != nil || value.ReadyBackup != nil || string(snapshotConnectionFields(t, value.Value)["status"]) != `"creating"` || value.Created.StatusCode != 203 || value.Created == value.LastAccepted {
						t.Fatal(value)
					}
					proof := bytes.Clone(value.Created.Body)
					value.CreatedBackup.Body["unknown"][0] = '!'
					value.Value[0] = '!'
					value.LastAccepted.Body[0] = '!'
					if !bytes.Equal(value.Created.Body, proof) {
						t.Fatal("owned create phases alias", value)
					}
				} else {
					value := out.delete
					if value.Deleted == nil || !*value.Deleted || value.Resolved == nil || value.Resolved.Observed == nil || value.Resolved.Backup == nil || value.Applied == nil || value.LastAccepted == nil || value.Applied == value.LastAccepted || value.Applied.StatusCode != 202 || string(value.Applied.Body) != "opaque acknowledgement" || value.Ready != nil || value.Absent != nil || string(value.BackupID) != `"selected"` {
						t.Fatal(value)
					}
					value.LastAccepted.Body[0] = '!'
					if string(value.Applied.Body) != "opaque acknowledgement" {
						t.Fatal("owned delete phases alias", value)
					}
				}
				service, err := conn.BlockStorageV3(context.Background())
				if err != nil || service == nil || cached != nil && cached != service.RawClient() {
					t.Fatal("cached proxy changed", service, err)
				}
				cached = service.RawClient()
			}
			wantRequests := int32(2)
			if family == "delete" {
				wantRequests = 4
			}
			if callbacks.Load() != 2 || locates.Load() != 1 || requests.Load() != wantRequests || cached.Microversion != "3.60" {
				t.Fatal(callbacks.Load(), locates.Load(), requests.Load(), cached)
			}
		})
	}
}

func TestConnectionBackupMutationsCaptureRecordedScopeAfterOriginalsBeforeGetterAcrossWait(t *testing.T) {
	for _, family := range []string{"create", "delete"} {
		t.Run(family, func(t *testing.T) {
			cloud := testcloud.New(t)
			if err := snapshotConnectionRecordScope(cloud.Provider, "before-original", "before-token"); err != nil {
				t.Fatal(err)
			}
			var callbacks, locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(callbacks.Load(), opts)
				}
				if err := snapshotConnectionRecordScope(cloud.Provider, "during-getter", "getter-token"); err != nil {
					return "", err
				}
				return cloud.Server.URL + backupConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("call-region"), sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("POST "+backupConnectionPath+"backups", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				backupMutationConnectionWire(t, r, "POST", backupConnectionPath+"backups", "getter-token")
				if err := snapshotConnectionRecordScope(cloud.Provider, "between-phases", "later-token"); err != nil {
					t.Error(err)
				}
				testcloud.JSON(w, 202, `{"backup":{"id":"created","status":"creating","availability_zone":{"wire":"zone"}}}`)
			})
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/requested", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				backupMutationConnectionWire(t, r, "GET", backupConnectionPath+"backups/requested", "getter-token")
				if err := snapshotConnectionRecordScope(cloud.Provider, "between-phases", "later-token"); err != nil {
					t.Error(err)
				}
				testcloud.JSON(w, 200, `{"backup":{"id":"selected","status":"available","availability_zone":{"wire":"zone"}}}`)
			})
			cloud.Mux.HandleFunc("DELETE "+backupConnectionPath+"backups/selected", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				backupMutationConnectionWire(t, r, "DELETE", backupConnectionPath+"backups/selected", "later-token")
				testcloud.JSON(w, 202, "opaque")
			})
			for _, id := range []string{"created", "selected"} {
				id := id
				cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/"+id, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					backupMutationConnectionWire(t, r, "GET", backupConnectionPath+"backups/"+id, "later-token")
					status := "available"
					if family == "delete" {
						status = "deleted"
					}
					testcloud.JSON(w, 203, `{"backup":{"status":"`+status+`"}}`)
				})
			}
			original := func() error {
				callbacks.Add(1)
				return snapshotConnectionRecordScope(cloud.Provider, "after-original", "original-token")
			}
			create := []blockstorage.CreateVolumeBackupOption{func(*blockstorage.CreateVolumeBackupOpts) error { return original() }}
			deleted := []blockstorage.DeleteVolumeBackupOption{func(next *blockstorage.DeleteVolumeBackupOpts) error {
				yes := true
				next.Wait = &yes
				return original()
			}}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			out, err := backupMutationConnectionCall(ctx, conn, family, create, deleted)
			if err != nil || out == nil {
				t.Fatal(out, err)
			}
			var views []json.RawMessage
			if family == "create" {
				views = []json.RawMessage{out.create.CreatedValue, out.create.Value, out.create.Ready}
			} else {
				if out.delete.Deleted == nil || !*out.delete.Deleted {
					t.Fatal(out.delete)
				}
				views = []json.RawMessage{out.delete.Resolved.Value, out.delete.Ready}
			}
			for _, view := range views {
				backupMutationConnectionLocation(t, view, `"after-original"`)
				zone := snapshotConnectionFields(t, snapshotConnectionFields(t, view)["location"])["zone"]
				if string(zone) != `{"wire":"zone"}` {
					t.Fatal("Backup zone did not survive partial wait", string(view))
				}
				location := snapshotConnectionFields(t, snapshotConnectionFields(t, view)["location"])
				if string(location["region_name"]) != `"call-region"` {
					t.Fatal(string(view))
				}
			}
			current, err := conn.CurrentLocation()
			if err != nil || string(current.Project.ID) != `"between-phases"` {
				t.Fatal(current, err)
			}
			wantRequests := int32(2)
			if family == "delete" {
				wantRequests = 3
			}
			if callbacks.Load() != 1 || locates.Load() != 1 || requests.Load() != wantRequests {
				t.Fatal(callbacks.Load(), locates.Load(), requests.Load())
			}
		})
	}
}

func TestConnectionBackupMutationsExplicitLocationAndRetainedOriginalsStayOwned(t *testing.T) {
	for _, family := range []string{"create", "delete"} {
		t.Run(family, func(t *testing.T) {
			cloud := testcloud.New(t)
			defaultCloud := "default-cloud"
			defaults := resource.CloudLocation{Cloud: &defaultCloud, Project: resource.CloudProject{ID: json.RawMessage(`"default-project"`)}}
			name, region, project := "owned-cloud", "owned-region", "owned-project-name"
			location := resource.CloudLocation{Cloud: &name, RegionName: &region, Zone: json.RawMessage(`{"unused":"zone"}`), Project: resource.CloudProject{ID: json.RawMessage(`"owned-project"`), Name: &project}}
			var retainedLocation *resource.CloudLocation
			var callbacks, locates atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if callbacks.Load() != 2 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(callbacks.Load(), opts)
				}
				retainedLocation.Project.ID[0] = '!'
				*retainedLocation.Cloud = "getter-mutated"
				return cloud.Server.URL + backupConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(defaults), sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			cloud.Mux.HandleFunc("POST "+backupConnectionPath+"backups", func(w http.ResponseWriter, r *http.Request) {
				body := backupMutationConnectionWire(t, r, "POST", backupConnectionPath+"backups", "test-token")
				payload := snapshotConnectionFields(t, snapshotConnectionFields(t, body)["backup"])
				if string(payload["name"]) != `"owned-name"` || string(payload["description"]) != `"owned-description"` || string(payload["snapshot_id"]) != `"owned-snapshot"` || string(payload["force"]) != "false" {
					t.Error(string(body))
				}
				testcloud.JSON(w, 202, `{"backup":{"id":"created","status":"creating"}}`)
			})
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/requested", func(w http.ResponseWriter, r *http.Request) {
				backupMutationConnectionWire(t, r, "GET", backupConnectionPath+"backups/requested", "test-token")
				testcloud.JSON(w, 200, `{"backup":{"id":"selected"}}`)
			})
			cloud.Mux.HandleFunc("DELETE "+backupConnectionPath+"backups/selected", func(w http.ResponseWriter, r *http.Request) {
				backupMutationConnectionWire(t, r, "DELETE", backupConnectionPath+"backups/selected", "test-token")
				testcloud.JSON(w, 202, "opaque")
			})
			var createRetained *blockstorage.CreateVolumeBackupOpts
			var deleteRetained *blockstorage.DeleteVolumeBackupOpts
			create := make([]blockstorage.CreateVolumeBackupOption, 2)
			create[0] = func(next *blockstorage.CreateVolumeBackupOpts) error {
				callbacks.Add(1)
				no := false
				next.Wait = &no
				next.Force = &no
				next.Location = &location
				ownedName, ownedDescription, ownedSnapshot := "owned-name", "owned-description", "owned-snapshot"
				next.Name, next.Description, next.SnapshotID = &ownedName, &ownedDescription, &ownedSnapshot
				createRetained = next
				create[1] = func(*blockstorage.CreateVolumeBackupOpts) error { t.Fatal("replaced original ran"); return nil }
				return nil
			}
			create[1] = func(next *blockstorage.CreateVolumeBackupOpts) error {
				callbacks.Add(1)
				*createRetained.Wait = true
				*createRetained.Force = true
				*createRetained.Name = "changed-name"
				*createRetained.Description = "changed-description"
				*createRetained.SnapshotID = "changed-snapshot"
				retainedLocation = createRetained.Location
				*retainedLocation.RegionName = "caller-mutated"
				return nil
			}
			deleted := make([]blockstorage.DeleteVolumeBackupOption, 2)
			deleted[0] = func(next *blockstorage.DeleteVolumeBackupOpts) error {
				callbacks.Add(1)
				next.Location = &location
				deleteRetained = next
				deleted[1] = func(*blockstorage.DeleteVolumeBackupOpts) error { t.Fatal("replaced original ran"); return nil }
				return nil
			}
			deleted[1] = func(*blockstorage.DeleteVolumeBackupOpts) error {
				callbacks.Add(1)
				retainedLocation = deleteRetained.Location
				*retainedLocation.RegionName = "caller-mutated"
				return nil
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			out, err := backupMutationConnectionCall(ctx, conn, family, create, deleted)
			if err != nil || out == nil {
				t.Fatal(out, err)
			}
			view := json.RawMessage(nil)
			if family == "create" {
				view = out.create.Value
				if out.create.Ready != nil {
					t.Fatal("retained wait pointer changed prepared policy", out.create)
				}
			} else {
				view = out.delete.Resolved.Value
				if out.delete.Deleted == nil || !*out.delete.Deleted || out.delete.Ready != nil {
					t.Fatal(out.delete)
				}
			}
			backupMutationConnectionLocation(t, view, `"owned-project"`)
			computed := snapshotConnectionFields(t, snapshotConnectionFields(t, view)["location"])
			if string(computed["cloud"]) != `"owned-cloud"` || string(computed["region_name"]) != `"owned-region"` || string(snapshotConnectionFields(t, computed["project"])["name"]) != `"owned-project-name"` {
				t.Fatal(string(view))
			}
			current, err := conn.CurrentLocation()
			if err != nil || current.Cloud == nil || *current.Cloud != "default-cloud" || string(current.Project.ID) != `"default-project"` || callbacks.Load() != 2 || locates.Load() != 1 {
				t.Fatal(current, err, callbacks.Load(), locates.Load())
			}
		})
	}
}

func TestConnectionBackupMutationEmptyDeleteIsServiceFreeEvenForNilConnection(t *testing.T) {
	cloud := testcloud.New(t)
	var locates atomic.Int32
	cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		return "", errors.New("empty delete must not select Cinder")
	}
	invalid := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`invalid`)}}
	validConn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(invalid))
	if err != nil {
		t.Fatal(err)
	}
	for _, conn := range []*sdk.Connection{nil, &sdk.Connection{}, validConn} {
		calls := 0
		negative := -time.Second
		value, err := conn.DeleteVolumeBackup(context.Background(), blockstorage.DeleteVolumeBackupRequest{}, func(next *blockstorage.DeleteVolumeBackupOpts) error {
			calls++
			yes := true
			next.Wait = &yes
			next.WaitPolicy = blockstorage.BackupMutationWaitOpts{Timeout: &negative, PollInterval: &negative}
			next.Location = &invalid
			return nil
		})
		if err != nil || value == nil || value.Deleted == nil || *value.Deleted || value.Resolved != nil || value.Applied != nil || value.LastAccepted != nil || value.Absent != nil || value.Ready != nil || value.BackupID != nil || calls != 1 || locates.Load() != 0 {
			t.Fatal(value, err, calls, locates.Load())
		}
		*value.Deleted = true
		again, err := conn.DeleteVolumeBackup(context.Background(), blockstorage.DeleteVolumeBackupRequest{})
		if err != nil || again == nil || again.Deleted == nil || *again.Deleted || again.Deleted == value.Deleted {
			t.Fatal("no-op bool shared across calls", again, err)
		}
	}
	for _, state := range []string{"nil context", "canceled", "callback error", "callback cancel", "nil option"} {
		t.Run(state, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause, optionErr := errors.New("empty delete cancel"), errors.New("empty delete original error")
			var selected context.Context = ctx
			want := resource.ErrInvalidOption
			calls := 0
			if state == "nil context" {
				selected = nil
			}
			if state == "canceled" {
				cancel(cause)
				want = context.Canceled
			}
			options := []blockstorage.DeleteVolumeBackupOption{func(*blockstorage.DeleteVolumeBackupOpts) error {
				calls++
				if state == "callback cancel" {
					cancel(cause)
				}
				if state == "callback error" || state == "callback cancel" {
					return optionErr
				}
				return nil
			}}
			wantCalls := 1
			if state == "nil context" || state == "canceled" {
				wantCalls = 0
			}
			if state == "nil option" {
				options = append(options, nil)
			}
			if state == "callback error" {
				want = optionErr
			}
			if state == "callback cancel" {
				want = context.Canceled
			}
			var conn *sdk.Connection
			value, err := conn.DeleteVolumeBackup(selected, blockstorage.DeleteVolumeBackupRequest{}, options...)
			if value != nil || !errors.Is(err, want) || calls != wantCalls || locates.Load() != 0 {
				t.Fatal(value, err, calls, wantCalls, locates.Load())
			}
			if (state == "canceled" || state == "callback cancel") && !errors.Is(err, cause) || state == "callback cancel" && !errors.Is(err, optionErr) {
				t.Fatal("empty path lost cause", err)
			}
			backupConnectionOperation(t, "DeleteVolumeBackup", err)
		})
	}
}

func TestConnectionBackupMutationsNonemptyPreflightAndGetterFailuresKeepOperationCauses(t *testing.T) {
	for _, family := range []string{"create", "delete"} {
		for _, state := range []string{"nil context", "nil connection", "zero connection", "already canceled", "nil option", "option error", "option cancellation", "invalid location", "getter error", "getter cancellation"} {
			t.Run(family+"/"+state, func(t *testing.T) {
				cloud := testcloud.New(t)
				var callbacks, locates atomic.Int32
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, originalErr, getterErr := errors.New("backup Connection cause"), errors.New("original option failed"), errors.New("Cinder getter failed")
				cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
					locates.Add(1)
					if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
						t.Error(callbacks.Load(), opts)
					}
					if state == "getter cancellation" {
						cancel(cause)
					}
					return "", getterErr
				}
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				callback := func() error {
					callbacks.Add(1)
					if state == "option cancellation" {
						cancel(cause)
						return originalErr
					}
					if state == "option error" {
						return originalErr
					}
					return nil
				}
				bad := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`invalid`)}}
				create := []blockstorage.CreateVolumeBackupOption{func(next *blockstorage.CreateVolumeBackupOpts) error {
					if state == "invalid location" {
						next.Location = &bad
					}
					return callback()
				}}
				deleted := []blockstorage.DeleteVolumeBackupOption{func(next *blockstorage.DeleteVolumeBackupOpts) error {
					if state == "invalid location" {
						next.Location = &bad
					}
					return callback()
				}}
				var selected context.Context = ctx
				want := resource.ErrInvalidOption
				wantCallbacks, wantLocates := int32(1), int32(0)
				switch state {
				case "nil context":
					selected = nil
					wantCallbacks = 0
				case "nil connection":
					conn = nil
					wantCallbacks = 0
				case "zero connection":
					conn = &sdk.Connection{}
				case "already canceled":
					cancel(cause)
					want = context.Canceled
					wantCallbacks = 0
				case "nil option":
					create = append(create, nil)
					deleted = append(deleted, nil)
				case "option error":
					want = originalErr
				case "option cancellation":
					want = context.Canceled
				case "getter error", "getter cancellation":
					want = getterErr
					wantLocates = 1
				}
				value, err := backupMutationConnectionCall(selected, conn, family, create, deleted)
				if value != nil || !errors.Is(err, want) || callbacks.Load() != wantCallbacks || locates.Load() != wantLocates {
					t.Fatal(value, err, callbacks.Load(), locates.Load(), wantCallbacks, wantLocates)
				}
				backupConnectionOperation(t, backupMutationConnectionName(family), err)
				var response *resource.ResponseError
				if errors.As(err, &response) {
					t.Fatal("preflight/getter fabricated HTTP evidence", err)
				}
				if (state == "already canceled" || state == "option cancellation" || state == "getter cancellation") && !errors.Is(err, cause) || state == "option cancellation" && !errors.Is(err, originalErr) || state == "getter cancellation" && !errors.Is(err, context.Canceled) {
					t.Fatal("lost original/cancel/getter cause", err)
				}
			})
		}
	}
}

func (f backupMutationConnectionTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func (b *backupMutationConnectionFaultBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if err == io.EOF {
		return n, b.readErr
	}
	return n, err
}

func (b *backupMutationConnectionFaultBody) Close() error { b.cancel(b.cause); return b.closeErr }

func TestConnectionBackupMutationsAcceptedIOFailuresJoinReadCloseAndCallerCause(t *testing.T) {
	for _, family := range []string{"create", "delete"} {
		t.Run(family, func(t *testing.T) {
			cloud := testcloud.New(t)
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(opts)
				}
				return cloud.Server.URL + backupConnectionPath, nil
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			readErr, closeErr, cause := errors.New("accepted backup read"), errors.New("accepted backup close"), errors.New("backup close canceled caller")
			var requests atomic.Int32
			body := `{"backup":{"id":"created","status":"available"}}`
			if family == "delete" {
				body = "opaque-delete"
			}
			cloud.Provider.HTTPClient.Transport = backupMutationConnectionTransport(func(r *http.Request) (*http.Response, error) {
				requests.Add(1)
				method, path := "POST", backupConnectionPath+"backups"
				var responseBody io.ReadCloser = &backupMutationConnectionFaultBody{reader: strings.NewReader(body), readErr: readErr, closeErr: closeErr, cancel: cancel, cause: cause}
				status := 203
				if family == "delete" && r.Method == "GET" {
					method, path = "GET", backupConnectionPath+"backups/requested"
					responseBody = io.NopCloser(strings.NewReader(`{"backup":{"id":"selected"}}`))
					status = 200
				} else if family == "delete" {
					method, path = "DELETE", backupConnectionPath+"backups/selected"
				}
				backupMutationConnectionWire(t, r, method, path, "test-token")
				return &http.Response{StatusCode: status, Header: http.Header{"X-Actual": {"accepted-IO"}}, Body: responseBody, Request: r}, nil
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			out, err := backupMutationConnectionCall(ctx, conn, family, nil, nil)
			var physical *resource.ResponseError
			if out == nil || !errors.As(err, &physical) || physical.StatusCode != 203 || string(physical.Body) != body || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
				t.Fatal(out, err, physical)
			}
			backupConnectionOperation(t, backupMutationConnectionName(family), err)
			var phase, last *blockstorage.VolumeBackupMutationPage
			if family == "create" {
				phase, last = out.create.Created, out.create.LastAccepted
				if out.create.Value != nil || out.create.CreatedValue != nil {
					t.Fatal(out.create)
				}
			} else {
				phase, last = out.delete.Applied, out.delete.LastAccepted
				if out.delete.Deleted != nil || out.delete.Resolved == nil || out.delete.Resolved.Observed == nil {
					t.Fatal(out.delete)
				}
			}
			if phase == nil || last == nil || phase == last || phase.StatusCode != 203 || string(phase.Body) != body || phase.Header.Get("X-Actual") != "accepted-IO" {
				t.Fatal(phase, last)
			}
			last.Body[0] = '!'
			physical.Header.Set("X-Actual", "mutated-error")
			if string(phase.Body) != body || phase.Header.Get("X-Actual") != "accepted-IO" {
				t.Fatal("physical error and stage proof share memory", phase)
			}
			want := int32(1)
			if family == "delete" {
				want = 2
			}
			if requests.Load() != want {
				t.Fatal(requests.Load(), want)
			}
		})
	}
}

func TestConnectionBackupMutationsBlockedFreshPollPreservesEarlierProofAndCustomCause(t *testing.T) {
	for _, family := range []string{"create", "delete"} {
		t.Run(family, func(t *testing.T) {
			cloud := testcloud.New(t)
			var requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(opts)
				}
				return cloud.Server.URL + backupConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("cancel blocked backup poll")
			entered := make(chan struct{})
			cloud.Mux.HandleFunc("POST "+backupConnectionPath+"backups", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				backupMutationConnectionWire(t, r, "POST", backupConnectionPath+"backups", "test-token")
				testcloud.JSON(w, 202, `{"backup":{"id":"created","status":"creating"}}`)
			})
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/requested", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				backupMutationConnectionWire(t, r, "GET", backupConnectionPath+"backups/requested", "test-token")
				testcloud.JSON(w, 200, `{"backup":{"id":"selected","status":"available"}}`)
			})
			cloud.Mux.HandleFunc("DELETE "+backupConnectionPath+"backups/selected", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				backupMutationConnectionWire(t, r, "DELETE", backupConnectionPath+"backups/selected", "test-token")
				testcloud.JSON(w, 202, "opaque")
			})
			id := "created"
			if family == "delete" {
				id = "selected"
			}
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/"+id, func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				backupMutationConnectionWire(t, r, "GET", backupConnectionPath+"backups/"+id, "test-token")
				close(entered)
				<-r.Context().Done()
			})
			type completed struct {
				value *backupMutationConnectionOutcome
				err   error
			}
			done := make(chan completed, 1)
			go func() {
				value, err := backupMutationConnectionCall(ctx, conn, family, nil, []blockstorage.DeleteVolumeBackupOption{blockstorage.WithDeleteVolumeBackupWait(true)})
				done <- completed{value, err}
			}()
			timer := time.NewTimer(5 * time.Second)
			defer timer.Stop()
			select {
			case <-entered:
				cancel(cause)
			case got := <-done:
				t.Fatal("returned before fresh poll", got.value, got.err)
			case <-timer.C:
				cancel(cause)
				t.Fatal("fresh poll not reached")
			}
			var got completed
			select {
			case got = <-done:
			case <-timer.C:
				cancel(cause)
				t.Fatal("canceled backup workflow did not return")
			}
			if got.value == nil || !errors.Is(got.err, context.Canceled) || !errors.Is(got.err, cause) {
				t.Fatal(got.value, got.err)
			}
			backupConnectionOperation(t, backupMutationConnectionName(family), got.err)
			var physical *resource.ResponseError
			if errors.As(got.err, &physical) {
				t.Fatal("blocked poll borrowed old accepted phase", got.err)
			}
			if family == "create" {
				value := got.value.create
				if value.Created == nil || value.Created.StatusCode != 202 || value.CreatedValue == nil || value.LastAccepted == nil || value.LastAccepted.StatusCode != 202 || value.Value != nil || value.Ready != nil {
					t.Fatal(value)
				}
			} else {
				value := got.value.delete
				if value.Deleted != nil || value.Resolved == nil || value.Resolved.Value == nil || value.Applied == nil || value.Applied.StatusCode != 202 || value.LastAccepted == nil || value.LastAccepted.StatusCode != 202 || value.Absent != nil || value.Ready != nil {
					t.Fatal(value)
				}
			}
			want := int32(2)
			if family == "delete" {
				want = 3
			}
			if requests.Load() != want {
				t.Fatal(requests.Load(), want)
			}
		})
	}
}

func TestConnectionBackupMutationCreateTextValidationPrecedesCinderSelection(t *testing.T) {
	for _, field := range []string{"volume", "name", "description", "snapshot"} {
		t.Run(field, func(t *testing.T) {
			cloud := testcloud.New(t)
			var callbacks, locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				return "", errors.New("invalid text must not reach Cinder")
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				t.Error("unexpected HTTP", r.Method, r.URL)
				w.WriteHeader(500)
			})
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			bad := string([]byte{0xff})
			volume := "literal/volume"
			if field == "volume" {
				volume = bad
			}
			result, err := conn.CreateVolumeBackup(context.Background(), blockstorage.CreateVolumeBackupRequest{VolumeID: volume}, func(next *blockstorage.CreateVolumeBackupOpts) error {
				callbacks.Add(1)
				switch field {
				case "name":
					next.Name = &bad
				case "description":
					next.Description = &bad
				case "snapshot":
					next.SnapshotID = &bad
				}
				return nil
			})
			var physical *resource.ResponseError
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &physical) || callbacks.Load() != 1 || locates.Load() != 0 || requests.Load() != 0 {
				t.Fatal(result, err, callbacks.Load(), locates.Load(), requests.Load())
			}
			backupConnectionOperation(t, "CreateVolumeBackup", err)
		})
	}
}

func TestConnectionBackupMutationsForceKeepsCachedSelectionAndNormalWaitPolicyIndependent(t *testing.T) {
	for _, selected := range []string{"", "3.60", "3.80"} {
		t.Run("selected="+selected, func(t *testing.T) {
			cloud := testcloud.New(t)
			var locates, callbacks, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(opts)
				}
				return cloud.Server.URL + backupConnectionPath, nil
			}
			options := []sdk.ConnectionOption{}
			if selected != "" {
				options = append(options, sdk.WithMicroversion(sdk.BlockStorage, selected))
			}
			conn, err := sdk.FromProvider(cloud.Provider, options...)
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.BlockStorageV3(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			original := service.RawClient()
			headers := map[string]string{"X-Captured": "owned"}
			if selected != "" {
				headers["openstack-api-version"] = "volume " + selected
				headers["x-openstack-volume-api-version"] = selected
			}
			expectedHeaders := make(map[string]string, len(headers))
			for key, value := range headers {
				expectedHeaders[key] = value
			}
			original.MoreHeaders = headers
			check := func(r *http.Request, method, path, token, version string) []byte {
				requests.Add(1)
				var data []byte
				if r.Body != nil {
					var readErr error
					data, readErr = io.ReadAll(r.Body)
					if readErr != nil {
						t.Error(readErr)
					}
				}
				generic := ""
				if version != "" {
					generic = "volume " + version
				}
				if r.Method != method || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != generic || r.Header.Get("X-OpenStack-Volume-API-Version") != version || r.Header.Get("X-Captured") != "owned" || method != "POST" && len(data) != 0 {
					t.Error(r.Method, r.URL, r.Header, string(data))
				}
				return data
			}
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/requested", func(w http.ResponseWriter, r *http.Request) {
				check(r, "GET", backupConnectionPath+"backups/requested", "test-token", selected)
				testcloud.JSON(w, 200, `{"backup":{"id":"selected","status":"available"}}`)
			})
			cloud.Mux.HandleFunc("POST "+backupConnectionPath+"backups/selected/action", func(w http.ResponseWriter, r *http.Request) {
				data := check(r, "POST", backupConnectionPath+"backups/selected/action", "test-token", "3.64")
				if string(data) != `{"os-force_delete":null}` {
					t.Error("force body became native empty object", string(data))
				}
				cloud.Provider.SetToken("after-action-token")
				testcloud.JSON(w, 203, "opaque-force")
			})
			cloud.Mux.HandleFunc("GET "+backupConnectionPath+"backups/selected", func(w http.ResponseWriter, r *http.Request) {
				check(r, "GET", backupConnectionPath+"backups/selected", "after-action-token", selected)
				testcloud.JSON(w, 200, `{"backup":{"status":"deleted"}}`)
			})
			cloud.Mux.HandleFunc("POST "+backupConnectionPath+"backups", func(w http.ResponseWriter, r *http.Request) {
				data := check(r, "POST", backupConnectionPath+"backups", "after-action-token", selected)
				fields := snapshotConnectionFields(t, snapshotConnectionFields(t, data)["backup"])
				if len(fields) != 6 || string(fields["force"]) != "false" || string(fields["incremental"]) != "false" {
					t.Error(string(data))
				}
				testcloud.JSON(w, 203, `{"backup":{"id":"created","status":"creating"}}`)
			})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			deleted, err := conn.DeleteVolumeBackup(ctx, blockstorage.DeleteVolumeBackupRequest{NameOrID: "requested"}, func(next *blockstorage.DeleteVolumeBackupOpts) error {
				callbacks.Add(1)
				yes := true
				next.Force, next.Wait = &yes, &yes
				return nil
			})
			if err != nil || deleted == nil || deleted.Deleted == nil || !*deleted.Deleted || deleted.Applied == nil || string(deleted.Applied.Body) != "opaque-force" || deleted.Applied.StatusCode != 203 || deleted.Ready == nil || deleted.ReadyBackup == nil || deleted.LastAccepted == nil || deleted.LastAccepted.StatusCode != 200 || deleted.Absent != nil {
				t.Fatal(deleted, err)
			}
			created, err := conn.CreateVolumeBackup(ctx, blockstorage.CreateVolumeBackupRequest{VolumeID: "literal/volume"}, blockstorage.WithCreateVolumeBackupWait(false))
			if err != nil || created == nil || created.Value == nil || created.Created == nil || created.Ready != nil {
				t.Fatal(created, err)
			}
			cached, err := conn.BlockStorageV3(context.Background())
			if err != nil || cached == nil || cached.RawClient() != original || original.ProviderClient != cloud.Provider || original.Microversion != selected || len(original.MoreHeaders) != len(expectedHeaders) || callbacks.Load() != 1 || locates.Load() != 1 || requests.Load() != 4 {
				t.Fatal(cached, err, original, callbacks.Load(), locates.Load(), requests.Load())
			}
			for key, want := range expectedHeaders {
				if original.MoreHeaders[key] != want {
					t.Fatal("force changed source headers", key, original.MoreHeaders)
				}
			}
		})
	}
}
