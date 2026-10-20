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

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/blockstorage"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type connectionAccessContractOutcome struct {
	resolved          *blockstorage.GetVolumeTypeResult
	typeID, projectID string
	value             json.RawMessage
	accesses          []*resource.RawResource
	observed, applied *blockstorage.VolumeTypesPage
}

func connectionAccessContractCall(ctx context.Context, conn *sdk.Connection, operation, identity, project string, options ...blockstorage.VolumeTypeReadOption) (*connectionAccessContractOutcome, error) {
	if operation == "GetVolumeTypeAccess" {
		result, err := conn.GetVolumeTypeAccess(ctx, blockstorage.GetVolumeTypeAccessRequest{NameOrID: identity}, options...)
		if result == nil {
			return nil, err
		}
		return &connectionAccessContractOutcome{resolved: result.Resolved, typeID: result.TypeID, value: result.Value, accesses: result.Accesses, observed: result.Observed}, err
	}
	input := blockstorage.VolumeTypeAccessRequest{NameOrID: identity, ProjectID: project}
	var result *blockstorage.VolumeTypeAccessActionResult
	var err error
	if operation == "AddVolumeTypeAccess" {
		result, err = conn.AddVolumeTypeAccess(ctx, input, options...)
	} else {
		result, err = conn.RemoveVolumeTypeAccess(ctx, input, options...)
	}
	if result == nil {
		return nil, err
	}
	return &connectionAccessContractOutcome{resolved: result.Resolved, typeID: result.TypeID, projectID: result.ProjectID, applied: result.Applied}, err
}

func connectionAccessContractCheckResolved(t *testing.T, result *connectionAccessContractOutcome) {
	t.Helper()
	if result == nil || result.resolved == nil || result.resolved.Type == nil || result.resolved.Observed == nil || result.resolved.Observed.StatusCode != 200 || len(result.resolved.Pages) != 0 || string(result.resolved.Type.Body["id"]) != `"resolved-type"` {
		t.Fatal("lost independent lookup evidence", result)
	}
}

func connectionAccessContractInstallHandlers(t *testing.T, cloud *testcloud.Cloud, operation string) {
	t.Helper()
	cloud.Mux.HandleFunc("GET /connection-access/v3/project/types/type", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "is_public=none" {
			t.Error(r.URL)
		}
		testcloud.JSON(w, 200, `{"volume_type":{"id":"resolved-type","name":"fast"}}`)
	})
	if operation == "GetVolumeTypeAccess" {
		cloud.Mux.HandleFunc("GET /connection-access/v3/project/types/resolved-type/os-volume-type-access", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
			testcloud.JSON(w, 200, `{"volume_type_access":[{"project_id":"foreign","location":"wire-only"}]}`)
		})
	} else {
		cloud.Mux.HandleFunc("POST /connection-access/v3/project/types/resolved-type/action", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.RawQuery != "" {
				t.Error(r.URL)
			}
			w.WriteHeader(202)
		})
	}
}

func TestConnectionVolumeTypeAccessOriginalsPrecedeCachedCinderAndPreserveRawAcknowledgements(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var callbacks, locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("originals must precede only Cinder v3 selection", callbacks.Load(), opts)
					return "", errors.New("unexpected service selection")
				}
				return cloud.Server.URL + "/connection-access/v3/project/", nil
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			var cached *gophercloud.ServiceClient
			cloud.Mux.HandleFunc("GET /connection-access/v3/project/types/type", func(w http.ResponseWriter, r *http.Request) {
				index := requests.Add(1)
				if r.URL.RawQuery != "is_public=none" || r.Header.Get("X-Auth-Token") != "live-token" || (index == 3 && r.Header.Get("X-Original") != "after-original") {
					t.Error(r.URL, r.Header)
				}
				w.Header().Set("X-Lookup", "separate")
				testcloud.JSON(w, 200, `{"volume_type":{"id":"resolved-type","name":"fast"}}`)
			})
			if operation == "GetVolumeTypeAccess" {
				cloud.Mux.HandleFunc("GET /connection-access/v3/project/types/resolved-type/os-volume-type-access", func(w http.ResponseWriter, r *http.Request) {
					index := requests.Add(1)
					if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live-token" || (index == 4 && r.Header.Get("X-Original") != "after-original") {
						t.Error(r.URL, r.Header)
					}
					w.Header().Set("X-Access", "actual")
					testcloud.JSON(w, 203, `{"volume_type_access":[{"project_id":"literal","volume_type_id":"resolved-type","unknown":9007199254740993}],"next":"must-not-follow"}`)
				})
			} else {
				cloud.Mux.HandleFunc("POST /connection-access/v3/project/types/resolved-type/action", func(w http.ResponseWriter, r *http.Request) {
					index := requests.Add(1)
					if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live-token" || (index == 4 && r.Header.Get("X-Original") != "after-original") {
						t.Error(r.URL, r.Header)
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
						return
					}
					fields := connectionReadFields(t, body)
					key := "addProjectAccess"
					if operation == "RemoveVolumeTypeAccess" {
						key = "removeProjectAccess"
					}
					payload := connectionReadFields(t, fields[key])
					var project string
					if err := json.Unmarshal(payload["project"], &project); err != nil {
						t.Error(err)
					}
					want := ""
					if index == 4 {
						want = "project/../?%界\x00"
					}
					if len(fields) != 1 || len(payload) != 1 || project != want {
						t.Error(string(body), project, want)
					}
					w.Header().Set("X-Action", "actual")
					w.WriteHeader(203)
					_, _ = w.Write([]byte{0xff, 0x00, 'a'})
				})
			}
			for index := range 2 {
				identity, project := "type", ""
				if index == 1 {
					project = "project/../?%界\x00"
				}
				result, err := connectionAccessContractCall(context.Background(), conn, operation, identity, project, func(*blockstorage.VolumeTypeReadOpts) error {
					callbacks.Add(1)
					cloud.Provider.SetToken("live-token")
					identity, project = "callback-mutated-type", "callback-mutated-project"
					if cached != nil {
						cached.MoreHeaders = map[string]string{"X-Original": "after-original"}
					}
					return nil
				})
				if err != nil {
					t.Fatal(result, err)
				}
				connectionAccessContractCheckResolved(t, result)
				if result.typeID != "resolved-type" || result.resolved.Observed.Header.Get("X-Lookup") != "separate" {
					t.Fatal(result)
				}
				if operation == "GetVolumeTypeAccess" {
					if result.observed == nil || result.observed.StatusCode != 203 || result.observed.Header.Get("X-Access") != "actual" || len(result.accesses) != 1 || string(result.accesses[0].Body["unknown"]) != "9007199254740993" || result.accesses[0].StatusCode != 203 || result.applied != nil {
						t.Fatal(result)
					}
					proof := bytes.Clone(result.observed.Body)
					result.value[0] = '!'
					result.accesses[0].Body["unknown"][0] = '!'
					result.accesses[0].Header.Set("X-Access", "caller-mutated")
					if !bytes.Equal(result.observed.Body, proof) || result.observed.Header.Get("X-Access") != "actual" || result.resolved.Observed.Header.Get("X-Lookup") != "separate" {
						t.Fatal("raw outputs alias independent proof")
					}
				} else {
					want := ""
					if index == 1 {
						want = "project/../?%界\x00"
					}
					if result.projectID != want || result.applied == nil || result.applied.StatusCode != 203 || result.applied.Header.Get("X-Action") != "actual" || !bytes.Equal(result.applied.Body, []byte{0xff, 0x00, 'a'}) || result.observed != nil {
						t.Fatal(result)
					}
				}
				service, err := conn.BlockStorageV3(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if cached != nil && cached != service.RawClient() {
					t.Fatal("Cinder cache changed")
				}
				cached = service.RawClient()
			}
			if callbacks.Load() != 2 || locates.Load() != 1 || requests.Load() != 4 {
				t.Fatal(callbacks.Load(), locates.Load(), requests.Load())
			}
		})
	}
}

func TestConnectionVolumeTypeAccessOwnsLocationFactoriesAndOriginalCallbackSlice(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, factory := range []string{"location", "bulk", "callbacks"} {
			t.Run(operation+"/"+factory, func(t *testing.T) {
				cloud := testcloud.New(t)
				defaultName, name, cloudName := "connection-default", "owned-call", "owned-cloud"
				defaults := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &defaultName}}
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(defaults), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-access/v3/project/"))
				if err != nil {
					t.Fatal(err)
				}
				input := resource.CloudLocation{Cloud: &cloudName, Zone: json.RawMessage("false"), Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name}}
				var options []blockstorage.VolumeTypeReadOption
				first, second, injected := 0, 0, 0
				switch factory {
				case "location":
					options = []blockstorage.VolumeTypeReadOption{blockstorage.WithVolumeTypeReadLocation(input)}
				case "bulk":
					options = []blockstorage.VolumeTypeReadOption{blockstorage.WithVolumeTypeReadOptions(blockstorage.VolumeTypeReadOpts{Location: &input})}
				default:
					options = make([]blockstorage.VolumeTypeReadOption, 2)
					options[0] = func(opts *blockstorage.VolumeTypeReadOpts) error {
						first++
						opts.Location = &input
						options[1] = func(*blockstorage.VolumeTypeReadOpts) error {
							injected++
							return errors.New("replacement must not run")
						}
						return nil
					}
					options[1] = func(opts *blockstorage.VolumeTypeReadOpts) error {
						second++
						name, cloudName = "mutated-project", "mutated-cloud"
						input.Project.ID[0], input.Zone[0] = '!', '!'
						if opts.Location == nil || opts.Location.Project.Name == nil || *opts.Location.Project.Name != "owned-call" || string(opts.Location.Project.ID) != `"scope"` || string(opts.Location.Zone) != "false" {
							t.Error("prior callback state was not owned", opts)
						}
						return nil
					}
				}
				if factory != "callbacks" {
					name, cloudName = "mutated-project", "mutated-cloud"
					input.Project.ID[0], input.Zone[0] = '!', '!'
				}
				connectionAccessContractInstallHandlers(t, cloud, operation)
				result, err := connectionAccessContractCall(context.Background(), conn, operation, "type", "project", options...)
				if err != nil {
					t.Fatal(result, err)
				}
				connectionAccessContractCheckResolved(t, result)
				location := connectionReadFields(t, connectionReadFields(t, result.resolved.Value)["location"])
				project := connectionReadFields(t, location["project"])
				if string(location["cloud"]) != `"owned-cloud"` || string(location["zone"]) != "false" || string(project["id"]) != `"scope"` || string(project["name"]) != `"owned-call"` {
					t.Fatal(string(result.resolved.Value))
				}
				if factory == "callbacks" && (first != 1 || second != 1 || injected != 0) {
					t.Fatal(first, second, injected)
				}
				if operation == "GetVolumeTypeAccess" && (len(result.accesses) != 1 || string(result.accesses[0].Body["location"]) != `"wire-only"` || string(result.accesses[0].Body["project_id"]) != `"foreign"`) {
					t.Fatal("ACL rows gained computed descriptors", result)
				}
				current, err := conn.CurrentLocation()
				if err != nil || current.Cloud != nil || current.Project.Name == nil || *current.Project.Name != "connection-default" || string(current.Project.ID) != `"scope"` {
					t.Fatal("operation overwrote connection location", current, err)
				}
			})
		}
	}
}

func TestConnectionVolumeTypeAccessRecordedScopeSnapshotsBetweenStagesAndRefreshesNextCall(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			if err := connectionReadRecordScope(cloud.Provider, "before-original"); err != nil {
				t.Fatal(err)
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("region-owned"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-access/v3/project/"))
			if err != nil {
				t.Fatal(err)
			}
			var lookups atomic.Int32
			cloud.Mux.HandleFunc("GET /connection-access/v3/project/types/type", func(w http.ResponseWriter, r *http.Request) {
				lookups.Add(1)
				if r.URL.RawQuery != "is_public=none" {
					t.Error(r.URL)
				}
				if err := connectionReadRecordScope(cloud.Provider, "between-stages"); err != nil {
					t.Error(err)
				}
				testcloud.JSON(w, 200, `{"volume_type":{"id":"resolved-type"}}`)
			})
			cloud.Mux.HandleFunc("GET /connection-access/v3/project/types/resolved-type/os-volume-type-access", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"volume_type_access":null}`) })
			cloud.Mux.HandleFunc("POST /connection-access/v3/project/types/resolved-type/action", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202) })
			callbacks := 0
			for index := range 2 {
				result, err := connectionAccessContractCall(context.Background(), conn, operation, "type", "project", func(*blockstorage.VolumeTypeReadOpts) error {
					callbacks++
					if index == 0 {
						return connectionReadRecordScope(cloud.Provider, "after-original")
					}
					return nil
				})
				if err != nil {
					t.Fatal(result, err)
				}
				connectionAccessContractCheckResolved(t, result)
				location := connectionReadFields(t, connectionReadFields(t, result.resolved.Value)["location"])
				project := connectionReadFields(t, location["project"])
				want := `"after-original"`
				if index == 1 {
					want = `"between-stages"`
				}
				if string(project["id"]) != want || string(project["name"]) != "null" || string(location["region_name"]) != `"region-owned"` {
					t.Fatal("wrong recorded scope snapshot", string(result.resolved.Value))
				}
				if operation == "GetVolumeTypeAccess" && (string(result.value) != "null" || result.accesses != nil) {
					t.Fatal(result)
				}
			}
			if callbacks != 2 || lookups.Load() != 2 {
				t.Fatal(callbacks, lookups.Load())
			}
		})
	}
}

func TestConnectionVolumeTypeAccessPreflightAndGetterFailuresKeepOuterOperationAndCause(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, kind := range []string{"nil context", "nil connection", "already canceled", "nil option", "option error", "option cancellation", "getter error", "getter cancellation"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, originalError, getterError := errors.New("access canceled"), errors.New("access original failed"), errors.New("access locator failed")
				callbacks, locates := 0, 0
				cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
					locates++
					if callbacks != 1 || opts.Type != "block-storage" || opts.Version != 3 {
						t.Error(callbacks, opts)
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
				options := []blockstorage.VolumeTypeReadOption{func(*blockstorage.VolumeTypeReadOpts) error {
					callbacks++
					if kind == "option cancellation" {
						cancel(cause)
					}
					if kind == "option error" || kind == "option cancellation" {
						return originalError
					}
					return nil
				}}
				want := resource.ErrInvalidOption
				wantCallbacks, wantLocates := 0, 0
				switch kind {
				case "nil context":
					ctx = nil
				case "nil connection":
					conn = nil
				case "already canceled":
					cancel(cause)
					want = context.Canceled
				case "nil option":
					options = append(options, nil)
					wantCallbacks = 1
				case "option error":
					want = originalError
					wantCallbacks = 1
				case "option cancellation":
					want = context.Canceled
					wantCallbacks = 1
				default:
					want = getterError
					wantCallbacks, wantLocates = 1, 1
				}
				result, err := connectionAccessContractCall(ctx, conn, operation, "type", "project", options...)
				if result != nil || !errors.Is(err, want) || callbacks != wantCallbacks || locates != wantLocates {
					t.Fatal(result, err, callbacks, locates)
				}
				connectionTypeContractCheckOperation(t, operation, err)
				if strings.Contains(kind, "cancellation") || kind == "already canceled" {
					if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
						t.Fatal("lost cancel cause", err)
					}
				}
				if kind == "option cancellation" && !errors.Is(err, originalError) {
					t.Fatal("lost original failure", err)
				}
			})
		}
	}
}

func TestConnectionVolumeTypeAccessSecondStageProofSurvivesReadCloseCancellationSourceDriftAndRejection(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, kind := range []string{"read close cancellation", "source change", "native forbidden"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-access/v3/project/"))
				if err != nil {
					t.Fatal(err)
				}
				service, err := conn.BlockStorageV3(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				client := service.RawClient()
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				readError, closeError, cause := errors.New("access read failed"), errors.New("access close failed"), errors.New("access close canceled")
				payload := []byte{0xff, 0x00, 'a'}
				if operation == "GetVolumeTypeAccess" {
					payload = []byte(`{"volume_type_access":[{"project_id":"project"}]}`)
				}
				calls, callbacks := 0, 0
				cloud.Provider.HTTPClient.Transport = connectionReadRoundTripper(func(r *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						if r.Method != http.MethodGet || r.URL.Path != "/connection-access/v3/project/types/type" || r.URL.RawQuery != "is_public=none" {
							t.Error(r.Method, r.URL)
						}
						return &http.Response{StatusCode: 200, Header: http.Header{"X-Lookup": {"complete"}}, Body: io.NopCloser(strings.NewReader(`{"volume_type":{"id":"resolved-type"}}`)), Request: r}, nil
					}
					path, method := "/connection-access/v3/project/types/resolved-type/action", http.MethodPost
					if operation == "GetVolumeTypeAccess" {
						path, method = "/connection-access/v3/project/types/resolved-type/os-volume-type-access", http.MethodGet
					}
					if calls != 2 || r.URL.Path != path || r.Method != method || r.URL.RawQuery != "" {
						t.Error(calls, r.Method, r.URL)
					}
					if kind == "native forbidden" {
						return &http.Response{StatusCode: 403, Header: http.Header{"X-Rejected": {"actual"}}, Body: io.NopCloser(strings.NewReader(`{"error":"denied"}`)), Request: r}, nil
					}
					stream := &connectionReadBody{content: bytes.Clone(payload), close: func() {
						if kind == "source change" {
							client.Endpoint = cloud.Server.URL + "/changed/v3/project/"
						} else {
							cancel(cause)
						}
					}}
					if kind == "read close cancellation" {
						stream.readError, stream.closeError = readError, closeError
					}
					return &http.Response{StatusCode: 203, Header: http.Header{"X-Current": {"actual"}}, Body: stream, Request: r}, nil
				})
				result, err := connectionAccessContractCall(ctx, conn, operation, "type", "project", func(*blockstorage.VolumeTypeReadOpts) error { callbacks++; return nil })
				if err == nil || calls != 2 || callbacks != 1 {
					t.Fatal(result, err, calls, callbacks)
				}
				connectionAccessContractCheckResolved(t, result)
				connectionTypeContractCheckOperation(t, operation, err)
				if result.value != nil || result.accesses != nil || result.resolved.Observed.Header.Get("X-Lookup") != "complete" {
					t.Fatal(result)
				}
				proof := result.applied
				if operation == "GetVolumeTypeAccess" {
					proof = result.observed
				}
				var accepted *resource.ResponseError
				if kind == "native forbidden" {
					var native gophercloud.ErrUnexpectedResponseCode
					if proof != nil || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"error":"denied"}` || native.ResponseHeader.Get("X-Rejected") != "actual" || errors.As(err, &accepted) {
						t.Fatal("rejected evidence replaced by lookup success", result, err)
					}
				} else {
					if proof == nil || proof.StatusCode != 203 || proof.Header.Get("X-Current") != "actual" || !bytes.Equal(proof.Body, payload) || !errors.As(err, &accepted) || accepted.StatusCode != 203 {
						t.Fatal(result, err, accepted)
					}
					if kind == "read close cancellation" {
						for _, want := range []error{readError, closeError, cause, context.Canceled} {
							if !errors.Is(err, want) {
								t.Fatal("lost joined cause", want, err)
							}
						}
					} else if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				}
			})
		}
	}
}

func TestConnectionVolumeTypeAccessLocalResolutionFailuresRetainLookupWithoutSecondStage(t *testing.T) {
	for _, operation := range []string{"GetVolumeTypeAccess", "AddVolumeTypeAccess", "RemoveVolumeTypeAccess"} {
		for _, kind := range []string{"absent", "missing response ID", "null response ID", "invalid owned location", "invalid project UTF8"} {
			if kind == "invalid project UTF8" && operation == "GetVolumeTypeAccess" {
				continue
			}
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-access/v3/project/"))
				if err != nil {
					t.Fatal(err)
				}
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET /connection-access/v3/project/types/type", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if r.URL.RawQuery != "is_public=none" {
						t.Error(r.URL)
					}
					if kind == "absent" {
						testcloud.JSON(w, 404, `{"error":"not found"}`)
						return
					}
					row := `{"id":"resolved-type"}`
					if kind == "missing response ID" {
						row = `{}`
					}
					if kind == "null response ID" {
						row = `{"id":null}`
					}
					testcloud.JSON(w, 200, `{"volume_type":`+row+`}`)
				})
				cloud.Mux.HandleFunc("GET /connection-access/v3/project/types", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if kind != "absent" || r.URL.RawQuery != "is_public=none" {
						t.Error(kind, r.URL)
					}
					testcloud.JSON(w, 200, `{"volume_types":[]}`)
				})
				var options []blockstorage.VolumeTypeReadOption
				if kind == "invalid owned location" {
					options = append(options, blockstorage.WithVolumeTypeReadLocation(resource.CloudLocation{Zone: json.RawMessage("!")}))
				}
				project := "project"
				if kind == "invalid project UTF8" {
					project = string([]byte{0xff})
				}
				result, err := connectionAccessContractCall(context.Background(), conn, operation, "type", project, options...)
				want := resource.ErrInvalidOption
				wantRequests := int32(1)
				if kind == "absent" {
					want, wantRequests = resource.ErrNotFound, 2
				}
				if result == nil || result.resolved == nil || !errors.Is(err, want) || requests.Load() != wantRequests || result.typeID != "" && kind != "invalid project UTF8" || result.observed != nil || result.applied != nil || result.value != nil || result.accesses != nil {
					t.Fatal(result, err, requests.Load())
				}
				connectionTypeContractCheckOperation(t, operation, err)
				var accepted *resource.ResponseError
				if errors.As(err, &accepted) {
					t.Fatal("local validation borrowed accepted HTTP evidence", err, accepted)
				}
				if kind == "absent" {
					if result.resolved.Type != nil || result.resolved.Observed != nil || len(result.resolved.Pages) != 1 || result.resolved.Pages[0].StatusCode != 200 {
						t.Fatal(result)
					}
				} else if result.resolved.Observed == nil || result.resolved.Observed.StatusCode != 200 {
					t.Fatal("lookup proof lost", result)
				}
				if kind == "invalid project UTF8" && (result.typeID != "resolved-type" || result.projectID != project || result.resolved.Type == nil) {
					t.Fatal(result)
				}
			})
		}
	}
}
