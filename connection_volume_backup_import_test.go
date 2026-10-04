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

	"github.com/gophercloud/gophercloud/v2"
	tokens "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	sdk "gophercloudsdk"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

func bipConnectionRequest(t *testing.T, r *http.Request, method, path, version, token string) []byte {
	t.Helper()
	var body []byte
	var err error
	if r.Body != nil {
		body, err = io.ReadAll(r.Body)
	}
	wantVersion := ""
	if version != "" {
		wantVersion = "volume " + version
	}
	if err != nil || r.Method != method || r.URL.Path != path || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != token || r.Header.Get("OpenStack-API-Version") != wantVersion || r.Header.Get("X-OpenStack-Volume-API-Version") != version || method == http.MethodGet && len(body) != 0 {
		t.Error("Connection import request changed", r.Method, r.URL, r.Header, string(body), err)
	}
	return body
}
func bipConnectionPayload(t *testing.T, r *http.Request, service, locator, version, token string) {
	t.Helper()
	outer := snapshotConnectionFields(t, bipConnectionRequest(t, r, http.MethodPost, backupConnectionPath+"backups/import_record", version, token))
	fields := snapshotConnectionFields(t, outer["backup-record"])
	serviceRaw, _ := json.Marshal(service)
	urlRaw, _ := json.Marshal(locator)
	if len(outer) != 1 || len(fields) != 2 || !bytes.Equal(fields["backup_service"], serviceRaw) || !bytes.Equal(fields["backup_url"], urlRaw) {
		t.Error("literal import body changed", outer, fields, string(serviceRaw), string(urlRaw))
	}
}
func bipConnectionView(t *testing.T, result *blockstorage.ImportVolumeBackupResult) {
	t.Helper()
	if result == nil || result.Value == nil {
		t.Fatal("no successful import view", result)
	}
	fields := snapshotConnectionFields(t, result.Value)
	if len(fields) != 24 || string(fields["location"]) != "null" || string(fields["id"]) != string(result.BackupID) {
		t.Fatal("import consumed scope or invented model", result, string(result.Value))
	}
}

type bipConnectionBody struct {
	reader            *strings.Reader
	readErr, closeErr error
	onClose           func()
}

func (b *bipConnectionBody) Read(p []byte) (int, error) {
	n, err := b.reader.Read(p)
	if err == io.EOF && b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *bipConnectionBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeErr
}

func TestConnectionImportVolumeBackupPreflightRejectsBeforeOnlyCinderSelection(t *testing.T) {
	for _, kind := range []string{"nil context", "nil connection", "zero connection", "custom cancellation", "invalid service UTF8", "invalid locator UTF8", "empty literal strings"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			var locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if kind != "empty literal strings" || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("preflight selected service", kind, opts)
				}
				return cloud.Server.URL + backupConnectionPath, nil
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			ctx := bepConnectionContext(t)
			cause := errors.New("Connection import custom cancellation")
			service, locator := "driver", "record"
			want := resource.ErrInvalidOption
			switch kind {
			case "nil context":
				ctx = nil
			case "nil connection":
				conn = nil
			case "zero connection":
				conn = &sdk.Connection{}
			case "custom cancellation":
				child, cancel := context.WithCancelCause(ctx)
				cancel(cause)
				ctx = child
				want = context.Canceled
			case "invalid service UTF8":
				service = string([]byte{0xff})
			case "invalid locator UTF8":
				locator = string([]byte{0xff})
			case "empty literal strings":
				service, locator = "", ""
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if kind != "empty literal strings" {
					t.Error("preflight reached HTTP", r.URL)
					w.WriteHeader(500)
					return
				}
				bipConnectionPayload(t, r, "", "", "3.60", "test-token")
				testcloud.JSON(w, 201, `{}`)
			})
			result, err := conn.ImportVolumeBackup(ctx, blockstorage.ImportVolumeBackupRequest{BackupService: service, BackupURL: locator})
			if kind == "empty literal strings" {
				if err != nil || result == nil || result.Applied == nil || string(result.BackupID) != "null" || result.Backup == nil || len(result.Backup.Body) != 0 || locates.Load() != 1 || requests.Load() != 1 {
					t.Fatal(result, err, locates.Load(), requests.Load())
				}
				bipConnectionView(t, result)
			} else {
				var physical *resource.ResponseError
				if result != nil || !errors.Is(err, want) || errors.As(err, &physical) || locates.Load() != 0 || requests.Load() != 0 {
					t.Fatal(result, err, physical, locates.Load(), requests.Load())
				}
				if kind == "custom cancellation" && !errors.Is(err, cause) {
					t.Fatal("custom cause lost", err)
				}
				backupConnectionOperation(t, "ImportVolumeBackup", err)
			}
		})
	}
}

func TestConnectionImportVolumeBackupUsesCachedCinderAndIgnoresInvalidLocation(t *testing.T) {
	for _, scope := range []string{"invalid configured location", "invalid recorded scope"} {
		t.Run(scope, func(t *testing.T) {
			cloud := testcloud.New(t)
			var locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("import selected another service", opts)
				}
				return cloud.Server.URL + backupConnectionPath, nil
			}
			options := []sdk.ConnectionOption{sdk.WithMicroversion(sdk.BlockStorage, "3.80")}
			token := "test-token"
			scopeErr := errors.New("recorded import scope invalid")
			if scope == "invalid configured location" {
				badText := string([]byte{0xff})
				bad := resource.CloudLocation{Cloud: &badText, Zone: json.RawMessage("not-json"), Project: resource.CloudProject{ID: json.RawMessage("not-json")}}
				if _, err := bad.ForResource(nil, nil); !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal("invalid location fixture is valid", err)
				}
				options = append(options, sdk.WithCloudLocation(bad))
			} else {
				recorded := &tokens.CreateResult{}
				recorded.Header = http.Header{"X-Subject-Token": {"scope-token"}}
				if err := cloud.Provider.SetTokenAndAuthResult(recorded); err != nil {
					t.Fatal(err)
				}
				recorded.Err = scopeErr
				token = "scope-token"
			}
			conn, err := sdk.FromProvider(cloud.Provider, options...)
			if err != nil {
				t.Fatal(err)
			}
			if scope == "invalid recorded scope" {
				if _, err := conn.CurrentLocation(); !errors.Is(err, scopeErr) {
					t.Fatal("scope fixture did not fail", err)
				}
			}
			reply := `{"backup":{"name":"returned","project_id":{"raw":"project"},"availability_zone":false,"location":{"wire":"ignored"}}}`
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				call := requests.Add(1)
				bipConnectionPayload(t, r, "driver", "literal/record % value", "3.80", token)
				if call == 2 && r.Header.Get("X-Captured") != "next owned" {
					t.Error("next call did not own current headers", r.Header)
				}
				w.Header().Set("X-Proof", "actual")
				testcloud.JSON(w, 203, reply)
			})
			first, err := conn.ImportVolumeBackup(bepConnectionContext(t), blockstorage.ImportVolumeBackupRequest{BackupService: "driver", BackupURL: "literal/record % value"})
			if err != nil || first == nil || first.Applied == nil || first.Backup == nil || first.Microversion != "3.80" || len(first.Discovery) != 0 {
				t.Fatal(first, err)
			}
			bipConnectionView(t, first)
			if string(first.Backup.Body["location"]) != `{"wire":"ignored"}` || string(first.BackupID) != "null" {
				t.Fatal("physical scope replaced logical model", first)
			}
			cached, err := conn.BlockStorageV3(bepConnectionContext(t))
			if err != nil || cached == nil {
				t.Fatal(cached, err)
			}
			source := cached.RawClient()
			source.MoreHeaders = map[string]string{"X-Captured": "next owned"}
			cloud.Provider.SetToken("later live token")
			token = "later live token"
			first.Applied.Body[0] = '!'
			first.Applied.Header.Set("X-Proof", "caller")
			second, err := conn.ImportVolumeBackup(bepConnectionContext(t), blockstorage.ImportVolumeBackupRequest{BackupService: "driver", BackupURL: "literal/record % value"})
			if err != nil || second == nil || second.Applied == nil || second.Applied.Header.Get("X-Proof") != "actual" || string(second.Applied.Body) != reply || second.Microversion != "3.80" || len(second.Discovery) != 0 || locates.Load() != 1 || requests.Load() != 2 || source.Microversion != "3.80" || source.ProviderClient != cloud.Provider {
				t.Fatal(second, err, locates.Load(), requests.Load(), source)
			}
			bipConnectionView(t, second)
			again, err := conn.BlockStorageV3(bepConnectionContext(t))
			if err != nil || again != cached {
				t.Fatal("import replaced cached Cinder", again, err)
			}
		})
	}
}

func TestConnectionImportVolumeBackupDiscoveryPolicyIsPerCallAndDoesNotMutateCache(t *testing.T) {
	cloud := testcloud.New(t)
	var locates, discovery, posts atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if opts.Type != "block-storage" || opts.Version != 3 {
			t.Error(opts)
		}
		return cloud.Server.URL + backupConnectionPath, nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	cached, err := conn.BlockStorageV3(bepConnectionContext(t))
	if err != nil || cached == nil {
		t.Fatal(cached, err)
	}
	source := cached.RawClient()
	source.MoreHeaders = map[string]string{"X-Captured": "first owned"}
	cloud.Provider.SetToken("first token")
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			call := discovery.Add(1)
			wantHeader, token, maximum := "first owned", "first token", "3.80"
			if call == 2 {
				wantHeader, token, maximum = "second owned", "second token", "3.40"
			}
			bipConnectionRequest(t, r, http.MethodGet, "/backup-connection/v3/", "", token)
			if r.Header.Get("X-Captured") != wantHeader {
				t.Error("discovery missed header snapshot", r.Header)
			}
			source.MoreHeaders["X-Captured"] = "changed during discovery"
			cloud.Provider.SetToken("live after discovery")
			w.Header().Set("X-Proof", "discovery")
			testcloud.JSON(w, 300, `{"version":{"id":"v3.0","max_version":"`+maximum+`"}}`)
			return
		}
		call := posts.Add(1)
		version, header := "3.64", "first owned"
		if call == 2 {
			version, header = "3.40", "second owned"
		}
		bipConnectionPayload(t, r, "driver", "record", version, "live after discovery")
		if r.Header.Get("X-Captured") != header {
			t.Error("POST adopted later ordinary headers", r.Header)
		}
		w.Header().Set("X-Proof", "post")
		testcloud.JSON(w, 201, `{"backup":{"id":"imported"}}`)
	})
	first, err := conn.ImportVolumeBackup(bepConnectionContext(t), blockstorage.ImportVolumeBackupRequest{BackupService: "driver", BackupURL: "record"})
	if err != nil || first == nil || first.Microversion != "3.64" || len(first.Discovery) != 1 || first.Discovery[0].StatusCode != 300 || first.Applied == nil || first.Applied.Header.Get("X-Proof") != "post" || source.Microversion != "" {
		t.Fatal(first, err, source)
	}
	bipConnectionView(t, first)
	source.MoreHeaders = map[string]string{"X-Captured": "second owned"}
	cloud.Provider.SetToken("second token")
	first.Discovery[0].Body[0] = '!'
	first.Discovery[0].Header.Set("X-Proof", "caller discovery")
	first.Applied.Body[0] = '?'
	first.Value[0] = '#'
	second, err := conn.ImportVolumeBackup(bepConnectionContext(t), blockstorage.ImportVolumeBackupRequest{BackupService: "driver", BackupURL: "record"})
	if err != nil || second == nil || second.Microversion != "3.40" || len(second.Discovery) != 1 || second.Discovery[0].Header.Get("X-Proof") != "discovery" || string(second.Discovery[0].Body) != `{"version":{"id":"v3.0","max_version":"3.40"}}` || second.Applied == nil || second.Applied.Header.Get("X-Proof") != "post" || source.Microversion != "" || locates.Load() != 1 || discovery.Load() != 2 || posts.Load() != 2 {
		t.Fatal(second, err, locates.Load(), discovery.Load(), posts.Load(), source)
	}
	bipConnectionView(t, second)
	again, err := conn.BlockStorageV3(bepConnectionContext(t))
	if err != nil || again != cached {
		t.Fatal("per-call import version replaced cached service", again, err)
	}
}

func TestConnectionImportVolumeBackupGetterFailuresJoinCallerCausesWithoutProof(t *testing.T) {
	for _, kind := range []string{"getter error", "getter error with cancellation", "successful canceled getter"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(bepConnectionContext(t))
			defer cancel(nil)
			getterErr, cause := errors.New("import Cinder getter failed"), errors.New("getter canceled import")
			var locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(opts)
				}
				if kind != "getter error" {
					cancel(cause)
				}
				if kind == "successful canceled getter" {
					return cloud.Server.URL + backupConnectionPath, nil
				}
				return "", getterErr
			}
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				t.Error("failed getter reached import", r.URL)
				w.WriteHeader(500)
			})
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithMicroversion(sdk.BlockStorage, "3.60"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := conn.ImportVolumeBackup(ctx, blockstorage.ImportVolumeBackupRequest{BackupService: "driver", BackupURL: "record"})
			var physical *resource.ResponseError
			if result != nil || err == nil || errors.As(err, &physical) || locates.Load() != 1 || requests.Load() != 0 {
				t.Fatal(result, err, physical, locates.Load(), requests.Load())
			}
			backupConnectionOperation(t, "ImportVolumeBackup", err)
			if kind != "successful canceled getter" && !errors.Is(err, getterErr) {
				t.Fatal("getter cause lost", err)
			}
			if kind != "getter error" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal("getter cancellation lost", err)
			}
		})
	}
}

func TestConnectionImportVolumeBackupExecutionErrorsKeepOnlyActualStageProof(t *testing.T) {
	for _, stage := range []string{"discovery accepted IO", "POST accepted IO", "descriptor failure", "later discovery native rejection"} {
		t.Run(stage, func(t *testing.T) {
			cloud := testcloud.New(t)
			backupConnectionEndpoint(t, cloud)
			options := []sdk.ConnectionOption{}
			if stage == "POST accepted IO" || stage == "descriptor failure" {
				options = append(options, sdk.WithMicroversion(sdk.BlockStorage, "3.80"))
			}
			conn, err := sdk.FromProvider(cloud.Provider, options...)
			if err != nil {
				t.Fatal(err)
			}
			cached, err := conn.BlockStorageV3(bepConnectionContext(t))
			if err != nil || cached == nil {
				t.Fatal(cached, err)
			}
			source := cached.RawClient()
			ctx, cancel := context.WithCancelCause(bepConnectionContext(t))
			defer cancel(nil)
			readErr, closeErr, cause := errors.New("import phase Read"), errors.New("import phase Close"), errors.New("import phase custom cancellation")
			var calls atomic.Int32
			reply := `{"backup":{"id":"actual","name":"returned"}}`
			if stage == "discovery accepted IO" {
				reply = `{"version":{"id":"v3.0","max_version":"3.80"}}`
			}
			if stage == "descriptor failure" {
				reply = `{"backup":{"id":"actual","object_count":"²","unknown":false}}`
			}
			cloud.Provider.HTTPClient.Transport = backupConnectionTransport(func(r *http.Request) (*http.Response, error) {
				call := calls.Add(1)
				if stage == "later discovery native rejection" {
					path := "/backup-connection/v3/"
					code, body, proof := 200, `{}`, "earlier"
					if call == 2 {
						path, code, body, proof = "/backup-connection/", 403, "native rejected", "later"
					}
					bipConnectionRequest(t, r, http.MethodGet, path, "", "test-token")
					return &http.Response{StatusCode: code, Header: http.Header{"X-Proof": {proof}}, Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
				}
				if stage == "discovery accepted IO" {
					bipConnectionRequest(t, r, http.MethodGet, "/backup-connection/v3/", "", "test-token")
				} else {
					bipConnectionPayload(t, r, "driver", "record", "3.80", "test-token")
				}
				code := 201
				if stage == "discovery accepted IO" {
					code = 200
				}
				body := &bipConnectionBody{reader: strings.NewReader(reply)}
				if stage != "descriptor failure" {
					body.readErr, body.closeErr = readErr, closeErr
					body.onClose = func() { source.Microversion = "3.61"; cancel(cause) }
				}
				return &http.Response{StatusCode: code, Header: http.Header{"X-Proof": {"current"}}, Body: body, Request: r}, nil
			})
			result, err := conn.ImportVolumeBackup(ctx, blockstorage.ImportVolumeBackupRequest{BackupService: "driver", BackupURL: "record"})
			if result == nil || err == nil || result.Value != nil {
				t.Fatal(result, err)
			}
			backupConnectionOperation(t, "ImportVolumeBackup", err)
			if stage == "later discovery native rejection" {
				var native gophercloud.ErrUnexpectedResponseCode
				var physical *resource.ResponseError
				if !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != "native rejected" || native.ResponseHeader.Get("X-Proof") != "later" || errors.As(err, &physical) || len(result.Discovery) != 1 || string(result.Discovery[0].Body) != `{}` || result.Discovery[0].Header.Get("X-Proof") != "earlier" || result.Applied != nil || result.Backup != nil || calls.Load() != 2 {
					t.Fatal(result, err, native, physical, calls.Load())
				}
				return
			}
			var physical *resource.ResponseError
			if !errors.As(err, &physical) || physical.Header.Get("X-Proof") != "current" || string(physical.Body) != reply || calls.Load() != 1 {
				t.Fatal(result, err, physical, calls.Load())
			}
			if stage == "discovery accepted IO" {
				if len(result.Discovery) != 1 || result.Discovery[0].StatusCode != 200 || result.Discovery[0].Header.Get("X-Proof") != "current" || string(result.Discovery[0].Body) != reply || result.Applied != nil || result.Backup != nil {
					t.Fatal(result, err)
				}
			} else if len(result.Discovery) != 0 || result.Applied == nil || result.Applied.StatusCode != 201 || result.Applied.Header.Get("X-Proof") != "current" || string(result.Applied.Body) != reply {
				t.Fatal(result, err)
			}
			if stage == "descriptor failure" {
				if result.Backup == nil || string(result.Backup.Body["unknown"]) != "false" {
					t.Fatal("decoded actual response lost on descriptor error", result, err)
				}
			} else if result.Backup != nil || !errors.Is(err, readErr) || !errors.Is(err, closeErr) || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("accepted phase lost joined causes", result, err)
			}
			physical.Body[0] = '!'
			physical.Header.Set("X-Proof", "error caller")
			phase := result.Applied
			if stage == "discovery accepted IO" {
				phase = result.Discovery[0]
			}
			if string(phase.Body) != reply || phase.Header.Get("X-Proof") != "current" {
				t.Fatal("error proof aliases result phase", phase, physical)
			}
		})
	}
}
