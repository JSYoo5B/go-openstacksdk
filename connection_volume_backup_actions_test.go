package openstack_test

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

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	tokens "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

type backupProxyActionConnectionOutcome struct {
	applied   *blockstorage.VolumeBackupMutationPage
	value     json.RawMessage
	completed bool
	restore   *blockstorage.RestoreVolumeBackupResult
	reset     *blockstorage.ResetVolumeBackupStatusResult
}

func backupProxyActionConnectionContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}
func backupProxyActionConnectionCall(ctx context.Context, conn *sdk.Connection, family string, options ...blockstorage.RestoreVolumeBackupOption) (*backupProxyActionConnectionOutcome, error) {
	if family == "restore" {
		opts := append([]blockstorage.RestoreVolumeBackupOption{blockstorage.WithRestoreVolumeBackupName("new")}, options...)
		value, err := conn.RestoreVolumeBackup(ctx, blockstorage.RestoreVolumeBackupRequest{BackupID: "literal"}, opts...)
		if value == nil {
			return nil, err
		}
		return &backupProxyActionConnectionOutcome{applied: value.Applied, value: value.Value, restore: value}, err
	}
	value, err := conn.ResetVolumeBackupStatus(ctx, blockstorage.ResetVolumeBackupStatusRequest{BackupID: "literal", Status: "custom"})
	if value == nil {
		return nil, err
	}
	return &backupProxyActionConnectionOutcome{applied: value.Applied, completed: value.Completed, reset: value}, err
}
func backupProxyActionConnectionOperation(family string) string {
	if family == "restore" {
		return "RestoreVolumeBackup"
	}
	return "ResetVolumeBackupStatus"
}
func backupProxyActionConnectionWire(t *testing.T, r *http.Request, family, token, selected, name string) {
	t.Helper()
	version, path, body := selected, backupConnectionPath+"backups/literal/restore", `{"restore":{"name":"`+name+`"}}`
	if family == "reset" {
		version, path, body = "3.64", backupConnectionPath+"backups/literal/action", `{"os-reset_status":{"status":"custom"}}`
	}
	actual, err := io.ReadAll(r.Body)
	if err != nil {
		t.Error(err)
	}
	if r.Body != nil {
		_ = r.Body.Close()
	}
	if r.Method != http.MethodPost || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("X-Openstack-Volume-Api-Version") != version || r.Header.Get("Openstack-Api-Version") != "volume "+version || string(actual) != body || r.ContentLength != int64(len(body)) || len(r.TransferEncoding) != 0 {
		t.Error("Connection action wire changed", r.Method, r.URL, r.Header, string(actual), err, r.ContentLength, r.TransferEncoding)
	}
}
func backupProxyActionConnectionScope(t *testing.T, raw json.RawMessage, project string) map[string]json.RawMessage {
	t.Helper()
	location := snapshotConnectionFields(t, snapshotConnectionFields(t, raw)["location"])
	scope := snapshotConnectionFields(t, location["project"])
	if string(scope["id"]) != project {
		t.Fatal("captured call scope changed", string(raw), scope)
	}
	return location
}

func TestConnectionBackupProxyActionOriginalsRunOnceBeforeOnlyCachedCinderSelection(t *testing.T) {
	cloud := testcloud.New(t)
	var originals, locates, requests atomic.Int32
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if originals.Load() != 1 || o.Type != "block-storage" || o.Version != 3 {
			t.Error("wrong order/service", originals.Load(), o)
		}
		return cloud.Server.URL + backupConnectionPath, nil
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := requests.Add(1)
		family, name, token := "restore", "first", "first-token"
		if n == 2 {
			name, token = "second", "second-token"
		}
		if n == 3 {
			family, token = "reset", "second-token"
		}
		backupProxyActionConnectionWire(t, r, family, token, "3.60", name)
		w.Header().Set("X-Proof", "current facade")
		testcloud.JSON(w, 203, `{"restore":{"status":"available"}}`)
	})
	var cached *gophercloud.ServiceClient
	for _, name := range []string{"first", "second"} {
		out, err := backupProxyActionConnectionCall(backupProxyActionConnectionContext(t), conn, "restore", func(o *blockstorage.RestoreVolumeBackupOpts) error {
			originals.Add(1)
			o.Name = &name
			return snapshotConnectionRecordScope(cloud.Provider, name+"-project", name+"-token")
		})
		if err != nil || out == nil || out.value == nil || out.applied == nil || out.applied.StatusCode != 203 || out.applied.Header.Get("X-Proof") != "current facade" {
			t.Fatal(out, err)
		}
		backupProxyActionConnectionScope(t, out.value, `"`+name+`-project"`)
		service, err := conn.BlockStorageV3(backupProxyActionConnectionContext(t))
		if err != nil || cached != nil && service.RawClient() != cached {
			t.Fatal(service, err)
		}
		cached = service.RawClient()
	}
	out, err := backupProxyActionConnectionCall(backupProxyActionConnectionContext(t), conn, "reset")
	service, getterErr := conn.BlockStorageV3(backupProxyActionConnectionContext(t))
	if err != nil || out == nil || !out.completed || out.applied == nil || getterErr != nil || service.RawClient() != cached || cached.Microversion != "3.60" || originals.Load() != 2 || locates.Load() != 1 || requests.Load() != 3 {
		t.Fatal(out, err, service, getterErr, originals.Load(), locates.Load(), requests.Load())
	}
}

func TestConnectionBackupProxyActionRestoreScopeAndOptionsSnapshotPrecedeGetterAndRefreshNextCall(t *testing.T) {
	cloud := testcloud.New(t)
	if err := snapshotConnectionRecordScope(cloud.Provider, "before-original", "before-token"); err != nil {
		t.Fatal(err)
	}
	var originals, locates, requests atomic.Int32
	var retained *blockstorage.RestoreVolumeBackupOpts
	name := "first"
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if originals.Load() != 1 || o.Type != "block-storage" || o.Version != 3 {
			t.Error(originals.Load(), o)
		}
		name = "retained pointer changed"
		if retained != nil {
			retained.Seed = json.RawMessage(`{"id":"wrong"}`)
		}
		if err := snapshotConnectionRecordScope(cloud.Provider, "during-getter", "getter-token"); err != nil {
			t.Error(err)
		}
		return cloud.Server.URL + backupConnectionPath, nil
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"), sdk.WithRegion("configured-region"))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := requests.Add(1)
		wantName, token := "first", "getter-token"
		if n == 2 {
			wantName, token = "second", "second-token"
		}
		backupProxyActionConnectionWire(t, r, "restore", token, "3.60", wantName)
		if err := snapshotConnectionRecordScope(cloud.Provider, "during-HTTP", "after-http-token"); err != nil {
			t.Error(err)
		}
		w.Header().Set("X-Proof", "captured scope")
		testcloud.JSON(w, 203, `{"restore":{"availability_zone":"actual-zone"}}`)
	})
	option := func(o *blockstorage.RestoreVolumeBackupOpts) error {
		originals.Add(1)
		retained = o
		o.Name = &name
		project, token := "first-project", "first-token"
		if originals.Load() == 2 {
			project, token = "second-project", "second-token"
		}
		return snapshotConnectionRecordScope(cloud.Provider, project, token)
	}
	for _, want := range []string{"first-project", "second-project"} {
		if want == "second-project" {
			name = "second"
		}
		out, err := backupProxyActionConnectionCall(backupProxyActionConnectionContext(t), conn, "restore", option)
		if err != nil || out == nil || out.value == nil || out.applied == nil {
			t.Fatal(out, err)
		}
		location := backupProxyActionConnectionScope(t, out.value, `"`+want+`"`)
		if string(location["region_name"]) != `"configured-region"` || string(location["zone"]) != `"actual-zone"` {
			t.Fatal(string(out.value))
		}
	}
	current, err := conn.CurrentLocation()
	if err != nil || string(current.Project.ID) != `"during-HTTP"` || originals.Load() != 2 || locates.Load() != 1 || requests.Load() != 2 {
		t.Fatal(current, err, originals.Load(), locates.Load(), requests.Load())
	}
}

func TestConnectionBackupProxyActionRestoreCompleteLocationOverrideRemainsOwnedAndDoesNotChangeDefaults(t *testing.T) {
	cloud := testcloud.New(t)
	backupConnectionEndpoint(t, cloud)
	baseCloud, baseRegion, baseName, baseDomain, baseDomainName := "base-cloud", "base-region", "base-name", "base-domain", "base-domain-name"
	base := resource.CloudLocation{Cloud: &baseCloud, RegionName: &baseRegion, Zone: json.RawMessage(`"configured-zone"`), Project: resource.CloudProject{ID: json.RawMessage(`"base-project"`), Name: &baseName, DomainID: &baseDomain, DomainName: &baseDomainName}}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"), sdk.WithCloudLocation(base))
	if err != nil {
		t.Fatal(err)
	}
	customCloud, customRegion, customName, customDomain, customDomainName := "owned-cloud", "owned-region", "owned-name", "owned-domain", "owned-domain-name"
	custom := resource.CloudLocation{Cloud: &customCloud, RegionName: &customRegion, Zone: json.RawMessage(`"unused-override-zone"`), Project: resource.CloudProject{ID: json.RawMessage(`"owned-project"`), Name: &customName, DomainID: &customDomain, DomainName: &customDomainName}}
	option := blockstorage.WithRestoreVolumeBackupLocation(custom)
	customCloud = "caller changed"
	customRegion = "caller changed"
	customName = "caller changed"
	customDomain = "caller changed"
	customDomainName = "caller changed"
	custom.Project.ID[1] = '!'
	custom.Zone[1] = '!'
	baseCloud = "base input changed"
	base.Project.ID[1] = '!'
	var requests atomic.Int32
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		backupProxyActionConnectionWire(t, r, "restore", "test-token", "3.60", "new")
		w.Header().Set("X-Proof", "location actual")
		testcloud.JSON(w, 203, `{"restore":{"availability_zone":["actual-zone"]}}`)
	})
	out, err := backupProxyActionConnectionCall(backupProxyActionConnectionContext(t), conn, "restore", option)
	if err != nil || out == nil || out.value == nil || requests.Load() != 1 {
		t.Fatal(out, err, requests.Load())
	}
	location := backupProxyActionConnectionScope(t, out.value, `"owned-project"`)
	project := snapshotConnectionFields(t, location["project"])
	if string(location["cloud"]) != `"owned-cloud"` || string(location["region_name"]) != `"owned-region"` || string(location["zone"]) != `["actual-zone"]` || string(project["name"]) != `"owned-name"` || string(project["domain_id"]) != `"owned-domain"` || string(project["domain_name"]) != `"owned-domain-name"` {
		t.Fatal(string(out.value))
	}
	defaults, err := conn.CurrentLocation()
	if err != nil || defaults.Cloud == nil || *defaults.Cloud != "base-cloud" || string(defaults.Project.ID) != `"base-project"` || string(defaults.Zone) != `"configured-zone"` || defaults.Project.Name == nil || *defaults.Project.Name != "base-name" {
		t.Fatal(defaults, err)
	}
	*defaults.Cloud = "borrowed copy changed"
	defaults.Project.ID[1] = '?'
	again, err := conn.CurrentLocation()
	if err != nil || *again.Cloud != "base-cloud" || string(again.Project.ID) != `"base-project"` {
		t.Fatal("CurrentLocation returned shared storage", again, err)
	}
}

func TestConnectionBackupProxyActionResetDoesNotConsumeMalformedRecordedScope(t *testing.T) {
	cloud := testcloud.New(t)
	var locates, requests atomic.Int32
	cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if o.Type != "block-storage" || o.Version != 3 {
			t.Error(o)
		}
		return cloud.Server.URL + backupConnectionPath, nil
	}
	malformed := tokens.CreateResult{}
	malformed.Header = http.Header{"X-Subject-Token": {"malformed-scope-token"}}
	malformed.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": 42}}}
	if err := cloud.Provider.SetTokenAndAuthResult(malformed); err != nil {
		t.Fatal(err)
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.80"))
	if err != nil {
		t.Fatal(err)
	}
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		n := requests.Add(1)
		family := "reset"
		if n == 2 {
			family = "restore"
		}
		backupProxyActionConnectionWire(t, r, family, "malformed-scope-token", "3.80", "new")
		w.Header().Set("X-Proof", "opaque without scope")
		if family == "reset" {
			testcloud.JSON(w, 203, "not JSON")
			return
		}
		testcloud.JSON(w, 203, `{"restore":{}}`)
	})
	if _, err := conn.CurrentLocation(); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal("fixture failed to make current scope invalid", err)
	}
	reset, err := backupProxyActionConnectionCall(backupProxyActionConnectionContext(t), conn, "reset")
	if err != nil || reset == nil || !reset.completed || reset.applied == nil || string(reset.applied.Body) != "not JSON" || requests.Load() != 1 || locates.Load() != 1 {
		t.Fatal(reset, err, requests.Load(), locates.Load())
	}
	restore, err := backupProxyActionConnectionCall(backupProxyActionConnectionContext(t), conn, "restore")
	if restore != nil || !errors.Is(err, resource.ErrInvalidOption) || requests.Load() != 1 || locates.Load() != 1 {
		t.Fatal("restore ignored required location failure", restore, err, requests.Load(), locates.Load())
	}
	backupConnectionOperation(t, "RestoreVolumeBackup", err)
	explicit := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"explicit-project"`)}}
	restore, err = backupProxyActionConnectionCall(backupProxyActionConnectionContext(t), conn, "restore", blockstorage.WithRestoreVolumeBackupLocation(explicit))
	if err != nil || restore == nil || restore.value == nil || requests.Load() != 2 || locates.Load() != 1 {
		t.Fatal(restore, err, requests.Load(), locates.Load())
	}
	backupProxyActionConnectionScope(t, restore.value, `"explicit-project"`)
	service, err := conn.BlockStorageV3(backupProxyActionConnectionContext(t))
	if err != nil || service.RawClient().Microversion != "3.80" {
		t.Fatal("forced reset changed cached selected MV", service, err)
	}
}

func TestConnectionBackupProxyActionPreflightAndOriginalFailuresNeverSelectService(t *testing.T) {
	for _, family := range []string{"restore", "reset"} {
		states := []string{"nil context", "canceled context", "nil Connection", "zero Connection", "invalid ID", "unsafe ID", "invalid UTF8"}
		if family == "restore" {
			states = append(states, "callback failure", "callback cancellation", "no destination")
		}
		for _, state := range states {
			t.Run(family+"/"+state, func(t *testing.T) {
				cloud := testcloud.New(t)
				var locates, requests, callbacks atomic.Int32
				cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
					locates.Add(1)
					return cloud.Server.URL + backupConnectionPath, nil
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					t.Error("invalid facade reached HTTP", r.Method, r.URL)
					w.WriteHeader(500)
				})
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				var target *sdk.Connection = conn
				ctx, cancel := context.WithCancelCause(backupProxyActionConnectionContext(t))
				defer cancel(nil)
				var callCtx context.Context = ctx
				callbackCause, cancelCause := errors.New("original action option failed"), errors.New("original action option canceled")
				id, status, name := "literal", "custom", "new"
				switch state {
				case "nil context":
					callCtx = nil
				case "canceled context":
					cancel(cancelCause)
				case "nil Connection":
					target = nil
				case "zero Connection":
					target = &sdk.Connection{}
				case "invalid ID":
					id = ""
				case "unsafe ID":
					id = "../bad"
				case "invalid UTF8":
					if family == "restore" {
						name = string([]byte{0xff})
					} else {
						status = string([]byte{0xff})
					}
				case "no destination":
					name = ""
				}
				var applied *blockstorage.VolumeBackupMutationPage
				if family == "restore" {
					result, failed := target.RestoreVolumeBackup(callCtx, blockstorage.RestoreVolumeBackupRequest{BackupID: id}, func(o *blockstorage.RestoreVolumeBackupOpts) error {
						callbacks.Add(1)
						o.Name = &name
						if state == "callback failure" {
							return callbackCause
						}
						if state == "callback cancellation" {
							cancel(cancelCause)
						}
						return nil
					})
					err = failed
					if result != nil {
						applied = result.Applied
						t.Error("local failure allocated result", result)
					}
				} else {
					result, failed := target.ResetVolumeBackupStatus(callCtx, blockstorage.ResetVolumeBackupStatusRequest{BackupID: id, Status: status})
					err = failed
					if result != nil {
						applied = result.Applied
						t.Error("local failure allocated result", result)
					}
				}
				var accepted *resource.ResponseError
				if err == nil || applied != nil || errors.As(err, &accepted) || locates.Load() != 0 || requests.Load() != 0 {
					t.Fatal(err, applied, locates.Load(), requests.Load())
				}
				backupConnectionOperation(t, backupProxyActionConnectionOperation(family), err)
				wantCallbacks := int32(0)
				if family == "restore" && state != "nil context" && state != "canceled context" && state != "nil Connection" {
					wantCallbacks = 1
				}
				if callbacks.Load() != wantCallbacks {
					t.Fatal("original invoked after preflight or rerun", callbacks.Load(), wantCallbacks)
				}
				if state == "callback failure" && !errors.Is(err, callbackCause) {
					t.Fatal(err)
				}
				if (state == "callback cancellation" || state == "canceled context") && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("cancel cause lost", err)
				}
			})
		}
	}
}

func TestConnectionBackupProxyActionGetterFailuresKeepOuterOperationAndCallerCauses(t *testing.T) {
	for _, family := range []string{"restore", "reset"} {
		for _, state := range []string{"getter error", "getter error and cancellation", "successful canceled getter"} {
			t.Run(family+"/"+state, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(backupProxyActionConnectionContext(t))
				defer cancel(nil)
				getterCause, cancelCause := errors.New("backup action getter"), errors.New("backup action getter canceled")
				var locates, requests, originals atomic.Int32
				cloud.Provider.EndpointLocator = func(o gophercloud.EndpointOpts) (string, error) {
					locates.Add(1)
					if o.Type != "block-storage" || o.Version != 3 || family == "restore" && originals.Load() != 1 {
						t.Error(o, originals.Load())
					}
					if state != "getter error" {
						cancel(cancelCause)
					}
					if state == "successful canceled getter" {
						return cloud.Server.URL + backupConnectionPath, nil
					}
					return "", getterCause
				}
				cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					t.Error("getter failure reached HTTP", r.URL)
					w.WriteHeader(500)
				})
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				out, err := backupProxyActionConnectionCall(ctx, conn, family, func(o *blockstorage.RestoreVolumeBackupOpts) error { originals.Add(1); return nil })
				var accepted *resource.ResponseError
				if out != nil || err == nil || errors.As(err, &accepted) || locates.Load() != 1 || requests.Load() != 0 {
					t.Fatal(out, err, locates.Load(), requests.Load())
				}
				backupConnectionOperation(t, backupProxyActionConnectionOperation(family), err)
				if state != "successful canceled getter" && !errors.Is(err, getterCause) {
					t.Fatal("getter cause lost", err)
				}
				if state != "getter error" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cancelCause)) {
					t.Fatal("getter cancellation lost", err)
				}
			})
		}
	}
}

func TestConnectionBackupProxyActionAcceptedIOKeepsActualAppliedAndJoinedSourceCancellation(t *testing.T) {
	for _, family := range []string{"restore", "reset"} {
		t.Run(family, func(t *testing.T) {
			cloud := testcloud.New(t)
			backupConnectionEndpoint(t, cloud)
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			service, err := conn.BlockStorageV3(backupProxyActionConnectionContext(t))
			if err != nil {
				t.Fatal(err)
			}
			source := service.RawClient()
			ctx, cancel := context.WithCancelCause(backupProxyActionConnectionContext(t))
			defer cancel(nil)
			readCause, closeCause, cancelCause := errors.New("Connection action Read"), errors.New("Connection action Close"), errors.New("Connection action cancellation")
			body := []byte{0xff, 0, 'a', 'c', 't', 'u', 'a', 'l'}
			var requests, retries atomic.Int32
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries.Add(1)
				return errors.New("accepted action cannot retry")
			}
			cloud.Provider.HTTPClient.Transport = backupConnectionTransport(func(r *http.Request) (*http.Response, error) {
				requests.Add(1)
				backupProxyActionConnectionWire(t, r, family, "test-token", "3.60", "new")
				fault := &backupConnectionFaultBody{reader: strings.NewReader(string(body)), readErr: readCause, closeErr: closeCause, onClose: func() { source.Microversion = "3.61"; cancel(cancelCause) }}
				return &http.Response{StatusCode: 203, Header: http.Header{"X-Proof": {"actual facade"}}, Body: fault, Request: r}, nil
			})
			out, err := backupProxyActionConnectionCall(ctx, conn, family)
			var accepted *resource.ResponseError
			if out == nil || out.applied == nil || out.value != nil || out.completed || out.restore != nil && out.restore.Backup != nil || !errors.As(err, &accepted) || !errors.Is(err, readCause) || !errors.Is(err, closeCause) || !errors.Is(err, cancelCause) || !errors.Is(err, context.Canceled) || !errors.Is(err, resource.ErrInvalidOption) || accepted.StatusCode != 203 || !bytes.Equal(accepted.Body, body) || accepted.Header.Get("X-Proof") != "actual facade" || out.applied.StatusCode != 203 || !bytes.Equal(out.applied.Body, body) || out.applied.Header.Get("X-Proof") != "actual facade" || requests.Load() != 1 || retries.Load() != 0 {
				t.Fatal(out, err, accepted, requests.Load(), retries.Load())
			}
			backupConnectionOperation(t, backupProxyActionConnectionOperation(family), err)
			out.applied.Body[0] = '!'
			out.applied.Header.Set("X-Proof", "result changed")
			if !bytes.Equal(accepted.Body, body) || accepted.Header.Get("X-Proof") != "actual facade" {
				t.Fatal("result aliases error", accepted)
			}
			accepted.Body[1] = '?'
			accepted.Header.Set("X-Proof", "error changed")
			if out.applied.Body[1] != body[1] || out.applied.Header.Get("X-Proof") != "result changed" {
				t.Fatal("error aliases result", out.applied)
			}
		})
	}
}
