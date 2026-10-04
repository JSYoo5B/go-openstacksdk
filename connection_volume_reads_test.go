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
	v3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
	sdk "gophercloudsdk"
	"gophercloudsdk/blockstorage"
	"gophercloudsdk/internal/testcloud"
	"gophercloudsdk/resource"
)

type connectionReadOutcome struct {
	value    json.RawMessage
	volume   *resource.RawResource
	volumes  []*resource.RawResource
	observed *blockstorage.GetVolumesPage
	pages    []*blockstorage.GetVolumesPage
	exists   *bool
}

func connectionReadCall(ctx context.Context, conn *sdk.Connection, operation string, options ...blockstorage.VolumeReadOption) (*connectionReadOutcome, error) {
	switch operation {
	case "ListVolumes":
		result, err := conn.ListVolumes(ctx, options...)
		if result == nil {
			return nil, err
		}
		return &connectionReadOutcome{value: result.Value, volumes: result.Volumes, pages: result.Pages}, err
	case "GetVolumeByID":
		result, err := conn.GetVolumeByID(ctx, blockstorage.GetVolumeByIDRequest{ID: "volume"}, options...)
		if result == nil {
			return nil, err
		}
		return &connectionReadOutcome{value: result.Value, volume: result.Volume, observed: result.Observed, pages: result.Pages}, err
	default:
		result, err := conn.VolumeExists(ctx, blockstorage.VolumeExistsRequest{NameOrID: "volume"}, options...)
		if result == nil {
			return nil, err
		}
		return &connectionReadOutcome{value: result.Value, volume: result.Volume, observed: result.Observed, pages: result.Pages, exists: result.Exists}, err
	}
}

func connectionReadFields(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(string(raw), err)
	}
	return fields
}

func connectionReadLocation(t *testing.T, operation string, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	if operation == "ListVolumes" {
		var rows []json.RawMessage
		if err := json.Unmarshal(raw, &rows); err != nil || len(rows) != 1 {
			t.Fatal(string(raw), err)
		}
		raw = rows[0]
	}
	return connectionReadFields(t, connectionReadFields(t, raw)["location"])
}

func connectionReadRecordScope(provider *gophercloud.ProviderClient, id string) error {
	recorded := v3.CreateResult{}
	recorded.Header = http.Header{"X-Subject-Token": {"scope-token"}}
	recorded.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": id, "name": "token-name-not-configured"}}}
	return provider.SetTokenAndAuthResult(recorded)
}

func connectionReadRespond(w http.ResponseWriter, r *http.Request, row string) {
	if strings.HasSuffix(r.URL.Path, "/detail") {
		testcloud.JSON(w, http.StatusOK, `{"volumes":[`+row+`]}`)
	} else {
		testcloud.JSON(w, http.StatusOK, `{"volume":`+row+`}`)
	}
}

func connectionReadCheckOperation(t *testing.T, operation string, err error) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "volume" {
		t.Fatal("wrong operation context", operation, err, wrapped)
	}
}

func TestConnectionVolumeReadsCachedCinderOriginalOptionsPrecedeSelection(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var callbacks, locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("original callback must precede Cinder v3 selection", callbacks.Load(), opts)
					return "", errors.New("unexpected service selection")
				}
				return cloud.Server.URL + "/connection-reads/v3/project/", nil
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			var cached *gophercloud.ServiceClient
			cloud.Mux.HandleFunc("GET /connection-reads/v3/project/volumes/", func(w http.ResponseWriter, r *http.Request) {
				index := requests.Add(1)
				if r.Header.Get("X-Auth-Token") != "live-token" || r.URL.RawQuery != "" || (index == 2 && r.Header.Get("X-Original") != "after-callback") {
					t.Error("captured request did not follow original options", r.URL, r.Header)
				}
				connectionReadRespond(w, r, `{"id":"volume","name":"data"}`)
			})
			for range 2 {
				result, err := connectionReadCall(context.Background(), conn, operation, func(*blockstorage.VolumeReadOpts) error {
					callbacks.Add(1)
					cloud.Provider.SetToken("live-token")
					if cached != nil {
						cached.MoreHeaders = map[string]string{"X-Original": "after-callback"}
					}
					return nil
				})
				if err != nil || result == nil || len(result.value) == 0 {
					t.Fatal(result, err)
				}
				if operation == "ListVolumes" {
					if len(result.volumes) != 1 || len(result.pages) != 1 || result.observed != nil {
						t.Fatal(result)
					}
				} else if result.volume == nil || result.observed == nil || len(result.pages) != 0 {
					t.Fatal(result)
				}
				if operation == "VolumeExists" && (result.exists == nil || !*result.exists) {
					t.Fatal(result)
				}
				service, err := conn.BlockStorageV3(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				if cached != nil && cached != service.RawClient() {
					t.Fatal("cached Cinder changed")
				}
				cached = service.RawClient()
			}
			if locates.Load() != 1 || callbacks.Load() != 2 || requests.Load() != 2 {
				t.Fatal(locates.Load(), callbacks.Load(), requests.Load())
			}
		})
	}
}

func TestConnectionVolumeReadsLocationFactoriesOwnInputsAndLeaveConnectionDefault(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		for _, factory := range []string{"location", "bulk"} {
			t.Run(operation+"/"+factory, func(t *testing.T) {
				cloud := testcloud.New(t)
				defaultName, callName, callCloud := "connection-project", "per-call-project", "per-call-cloud"
				defaults := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &defaultName}}
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(defaults), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-reads/v3/project/"))
				if err != nil {
					t.Fatal(err)
				}
				input := resource.CloudLocation{Cloud: &callCloud, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &callName}}
				option := blockstorage.WithVolumeReadLocation(input)
				if factory == "bulk" {
					option = blockstorage.WithVolumeReadOptions(blockstorage.VolumeReadOpts{Location: &input})
				}
				callName, callCloud = "caller-mutated-project", "caller-mutated-cloud"
				input.Project.ID[0] = '!'
				cloud.Mux.HandleFunc("GET /connection-reads/v3/project/volumes/", func(w http.ResponseWriter, r *http.Request) {
					connectionReadRespond(w, r, `{"id":"volume","project_id":"scope","availability_zone":false,"location":{"cloud":"untrusted-wire"}}`)
				})
				for range 2 {
					result, err := connectionReadCall(context.Background(), conn, operation, option)
					if err != nil || result == nil {
						t.Fatal(result, err)
					}
					location := connectionReadLocation(t, operation, result.value)
					project := connectionReadFields(t, location["project"])
					if string(location["cloud"]) != `"per-call-cloud"` || string(location["zone"]) != "false" || string(project["name"]) != `"per-call-project"` || string(project["id"]) != `"scope"` {
						t.Fatal(string(result.value))
					}
					result.value[0] = '!'
				}
				current, err := conn.CurrentLocation()
				if err != nil || current.Cloud != nil || current.Project.Name == nil || *current.Project.Name != "connection-project" || string(current.Project.ID) != `"scope"` {
					t.Fatal("operation replaced connection defaults", current, err)
				}
			})
		}
	}
}

func TestConnectionVolumeReadsOriginalCallbackSliceAndRetainedLocationAreSnapshots(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-reads/v3/project/"))
			if err != nil {
				t.Fatal(err)
			}
			name := "owned-callback"
			input := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name}}
			var first, second, injected int
			var originals []blockstorage.VolumeReadOption
			originals = []blockstorage.VolumeReadOption{
				func(opts *blockstorage.VolumeReadOpts) error {
					first++
					opts.Location = &input
					originals[1] = func(*blockstorage.VolumeReadOpts) error { injected++; return errors.New("caller replacement ran") }
					return nil
				},
				func(opts *blockstorage.VolumeReadOpts) error {
					second++
					// The previous callback's retained pointer is outside the owned option state.
					name = "caller-mutated"
					input.Project.ID[0] = '!'
					if opts.Location == nil || opts.Location.Project.Name == nil || *opts.Location.Project.Name != "owned-callback" || string(opts.Location.Project.ID) != `"scope"` {
						t.Error("previous callback did not yield an owned snapshot", opts)
					}
					return nil
				},
			}
			cloud.Mux.HandleFunc("GET /connection-reads/v3/project/volumes/", func(w http.ResponseWriter, r *http.Request) {
				connectionReadRespond(w, r, `{"id":"volume","project_id":"scope"}`)
			})
			result, err := connectionReadCall(context.Background(), conn, operation, originals...)
			if err != nil || result == nil || first != 1 || second != 1 || injected != 0 {
				t.Fatal(result, err, first, second, injected)
			}
			project := connectionReadFields(t, connectionReadLocation(t, operation, result.value)["project"])
			if string(project["name"]) != `"owned-callback"` || string(project["id"]) != `"scope"` {
				t.Fatal(string(result.value))
			}
		})
	}
}

func TestConnectionVolumeReadsRecordedScopeIsCapturedAfterOptionsAndRefreshedPerCall(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			if err := connectionReadRecordScope(cloud.Provider, "before-options"); err != nil {
				t.Fatal(err)
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("region"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-reads/v3/project/"))
			if err != nil {
				t.Fatal(err)
			}
			var requests atomic.Int32
			cloud.Mux.HandleFunc("GET /connection-reads/v3/project/volumes/", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if err := connectionReadRecordScope(cloud.Provider, "in-flight"); err != nil {
					t.Error(err)
				}
				connectionReadRespond(w, r, `{"id":"volume","project_id":false,"availability_zone":null}`)
			})
			result, err := connectionReadCall(context.Background(), conn, operation, func(*blockstorage.VolumeReadOpts) error {
				return connectionReadRecordScope(cloud.Provider, "after-options")
			})
			if err != nil || result == nil {
				t.Fatal(result, err)
			}
			location := connectionReadLocation(t, operation, result.value)
			project := connectionReadFields(t, location["project"])
			if string(project["id"]) != `"after-options"` || string(project["name"]) != "null" || string(location["region_name"]) != `"region"` || string(location["cloud"]) != "null" {
				t.Fatal("execution adopted in-flight scope or inferred token names", string(result.value))
			}
			result, err = connectionReadCall(context.Background(), conn, operation)
			if err != nil || result == nil {
				t.Fatal(result, err)
			}
			project = connectionReadFields(t, connectionReadLocation(t, operation, result.value)["project"])
			if string(project["id"]) != `"in-flight"` || requests.Load() != 2 {
				t.Fatal(string(result.value), requests.Load())
			}
		})
	}
}

func TestConnectionVolumeReadsPreflightAndOptionFailuresSelectNoService(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		for _, kind := range []string{"nil context", "nil connection", "already canceled", "nil option", "option error", "callback cancellation"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				var locates atomic.Int32
				cloud.Provider.EndpointLocator = func(gophercloud.EndpointOpts) (string, error) {
					locates.Add(1)
					return "", errors.New("unexpected locator")
				}
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, optionError := errors.New("caller stopped read"), errors.New("read option rejected")
				current, currentContext := conn, context.Context(ctx)
				count := 0
				options := []blockstorage.VolumeReadOption{func(*blockstorage.VolumeReadOpts) error { count++; return nil }}
				want, wantCount := resource.ErrInvalidOption, 0
				switch kind {
				case "nil context":
					currentContext = nil
				case "nil connection":
					current = nil
				case "already canceled":
					cancel(cause)
					want = context.Canceled
				case "nil option":
					options = append(options, nil)
					wantCount = 1
				case "option error":
					options[0] = func(*blockstorage.VolumeReadOpts) error { count++; return optionError }
					want, wantCount = optionError, 1
				case "callback cancellation":
					options = []blockstorage.VolumeReadOption{func(*blockstorage.VolumeReadOpts) error { count++; cancel(cause); return optionError }, func(*blockstorage.VolumeReadOpts) error { count++; return nil }}
					want, wantCount = context.Canceled, 1
				}
				result, err := connectionReadCall(currentContext, current, operation, options...)
				if result != nil || !errors.Is(err, want) || count != wantCount || locates.Load() != 0 {
					t.Fatal(result, err, count, locates.Load())
				}
				connectionReadCheckOperation(t, operation, err)
				if kind == "already canceled" || kind == "callback cancellation" {
					if !errors.Is(err, cause) {
						t.Fatal("lost cancellation cause", err)
					}
				}
				if kind == "callback cancellation" && !errors.Is(err, optionError) {
					t.Fatal("lost callback failure", err)
				}
			})
		}
	}
}

func TestConnectionVolumeReadsGetterFailureRetainsOperationAndCancellationCause(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		for _, canceled := range []bool{false, true} {
			t.Run(operation+"/"+map[bool]string{false: "getter error", true: "getter cancellation"}[canceled], func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, getterError := errors.New("Cinder scope selection canceled"), errors.New("Cinder catalog unavailable")
				count, locates := 0, 0
				cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
					locates++
					if count != 1 || opts.Type != "block-storage" || opts.Version != 3 {
						t.Error("wrong service or order", count, opts)
					}
					if canceled {
						cancel(cause)
					}
					return "", getterError
				}
				conn, err := sdk.FromProvider(cloud.Provider)
				if err != nil {
					t.Fatal(err)
				}
				result, err := connectionReadCall(ctx, conn, operation, func(*blockstorage.VolumeReadOpts) error { count++; return nil })
				if result != nil || !errors.Is(err, getterError) || count != 1 || locates != 1 {
					t.Fatal(result, err, count, locates)
				}
				connectionReadCheckOperation(t, operation, err)
				if canceled && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
					t.Fatal("getter cancellation lost", err)
				}
			})
		}
	}
}

func TestConnectionVolumeReadsRecordedScopeFailurePrecedesGetterAndOwnedOverrideBypassesIt(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			invalid := v3.CreateResult{}
			invalid.Header = http.Header{"X-Subject-Token": {"recorded-token"}}
			invalid.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": 123}}}
			if err := cloud.Provider.SetTokenAndAuthResult(invalid); err != nil {
				t.Fatal(err)
			}
			count, locates := 0, 0
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates++
				if opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(opts)
				}
				return cloud.Server.URL + "/connection-reads/v3/project/", nil
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			callback := func(*blockstorage.VolumeReadOpts) error { count++; return nil }
			result, err := connectionReadCall(context.Background(), conn, operation, callback)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || count != 1 || locates != 0 {
				t.Fatal(result, err, count, locates)
			}
			connectionReadCheckOperation(t, operation, err)
			cloud.Mux.HandleFunc("GET /connection-reads/v3/project/volumes/", func(w http.ResponseWriter, r *http.Request) {
				connectionReadRespond(w, r, `{"id":"volume","project_id":"owned-scope"}`)
			})
			result, err = connectionReadCall(context.Background(), conn, operation, callback,
				blockstorage.WithVolumeReadLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"owned-scope"`)}}))
			if err != nil || result == nil || count != 2 || locates != 1 {
				t.Fatal(result, err, count, locates)
			}
			project := connectionReadFields(t, connectionReadLocation(t, operation, result.value)["project"])
			if string(project["id"]) != `"owned-scope"` {
				t.Fatal(string(result.value))
			}
		})
	}
}

type connectionReadRoundTripper func(*http.Request) (*http.Response, error)

func (f connectionReadRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type connectionReadBody struct {
	content               []byte
	readError, closeError error
	close                 func()
	consumed              bool
}

func (body *connectionReadBody) Read(target []byte) (int, error) {
	if body.consumed {
		return 0, io.EOF
	}
	body.consumed = true
	return copy(target, body.content), body.readError
}
func (body *connectionReadBody) Close() error { body.close(); return body.closeError }

func TestConnectionVolumeReadsAcceptedProofSurvivesReadCloseCancellationAndSourceFailure(t *testing.T) {
	for _, operation := range []string{"ListVolumes", "GetVolumeByID", "VolumeExists"} {
		for _, kind := range []string{"read close cancellation", "source change"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-reads/v3/project/"))
				if err != nil {
					t.Fatal(err)
				}
				selected, err := conn.BlockStorageV3(context.Background())
				if err != nil {
					t.Fatal(err)
				}
				client := selected.RawClient()
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, readError, closeError := errors.New("accepted read canceled"), errors.New("read evidence failed"), errors.New("close evidence failed")
				body := []byte(`{"volume":{"id":"volume"}}`)
				target := "/connection-reads/v3/project/volumes/volume"
				if operation == "ListVolumes" {
					body = []byte(`{"volumes":[{"id":"volume"}]}`)
					target = "/connection-reads/v3/project/volumes/detail"
				}
				calls := 0
				cloud.Provider.HTTPClient.Transport = connectionReadRoundTripper(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.Method != http.MethodGet || r.URL.Path != target || r.URL.RawQuery != "" {
						t.Error(r.Method, r.URL)
					}
					content := &connectionReadBody{content: bytes.Clone(body), close: func() {}}
					if kind == "read close cancellation" {
						content.readError, content.closeError = readError, closeError
						content.close = func() { cancel(cause) }
					} else {
						content.readError = io.EOF
						content.close = func() { client.Endpoint = cloud.Server.URL + "/changed/v3/project/" }
					}
					return &http.Response{StatusCode: 200, Status: "200 OK", Header: http.Header{"Content-Type": {"application/json"}, "X-Evidence": {"actual"}}, Body: content, Request: r}, nil
				})
				count := 0
				result, err := connectionReadCall(ctx, conn, operation, func(*blockstorage.VolumeReadOpts) error { count++; return nil })
				if result == nil || err == nil || result.value != nil || result.volume != nil || result.volumes != nil || result.exists != nil || count != 1 || calls != 1 {
					t.Fatal(result, err, count, calls)
				}
				connectionReadCheckOperation(t, operation, err)
				proof := result.observed
				if operation == "ListVolumes" {
					if len(result.pages) != 1 || result.observed != nil {
						t.Fatal(result)
					}
					proof = result.pages[0]
				} else if len(result.pages) != 0 {
					t.Fatal("unexpected fallback", result)
				}
				if proof == nil || proof.StatusCode != 200 || proof.Header.Get("X-Evidence") != "actual" || !bytes.Equal(proof.Body, body) {
					t.Fatal(proof, string(body))
				}
				if kind == "read close cancellation" {
					for _, sentinel := range []error{readError, closeError, context.Canceled, cause} {
						if !errors.Is(err, sentinel) {
							t.Fatal("lost completed-response cause", sentinel, err)
						}
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestConnectionVolumeReadsMember404IsErrorButCompletedExistenceAbsenceIsFalse(t *testing.T) {
	cloud := testcloud.New(t)
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-reads/v3/project/"))
	if err != nil {
		t.Fatal(err)
	}
	var gets, lists atomic.Int32
	cloud.Mux.HandleFunc("GET /connection-reads/v3/project/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		w.Header().Set("X-Rejected", "member-not-found")
		testcloud.JSON(w, http.StatusNotFound, `{"itemNotFound":{"message":"absent"}}`)
	})
	cloud.Mux.HandleFunc("GET /connection-reads/v3/project/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		index := lists.Add(1)
		if r.URL.Query().Get("name") != "volume" {
			t.Error(r.URL)
		}
		if index == 1 {
			testcloud.JSON(w, 200, `{"volumes":[]}`)
			return
		}
		w.Header().Set("X-Rejected", "list-forbidden")
		testcloud.JSON(w, http.StatusForbidden, `{"forbidden":{"message":"not authorized"}}`)
	})
	byID, err := conn.GetVolumeByID(context.Background(), blockstorage.GetVolumeByIDRequest{ID: "volume"})
	var rejected gophercloud.ErrUnexpectedResponseCode
	if byID == nil || byID.Value != nil || byID.Volume != nil || byID.Observed != nil || len(byID.Pages) != 0 || !errors.As(err, &rejected) || rejected.Actual != 404 || rejected.ResponseHeader.Get("X-Rejected") != "member-not-found" || lists.Load() != 0 {
		t.Fatal(byID, err, rejected, lists.Load())
	}
	connectionReadCheckOperation(t, "GetVolumeByID", err)
	absent, err := conn.VolumeExists(context.Background(), blockstorage.VolumeExistsRequest{NameOrID: "volume"})
	if err != nil || absent == nil || absent.Exists == nil || *absent.Exists || absent.Value != nil || absent.Volume != nil || absent.Observed != nil || len(absent.Pages) != 1 || absent.Pages[0].StatusCode != 200 {
		t.Fatal(absent, err)
	}
	failed, err := conn.VolumeExists(context.Background(), blockstorage.VolumeExistsRequest{NameOrID: "volume"})
	rejected = gophercloud.ErrUnexpectedResponseCode{}
	if failed == nil || failed.Exists != nil || failed.Value != nil || failed.Volume != nil || failed.Observed != nil || len(failed.Pages) != 0 || !errors.As(err, &rejected) || rejected.Actual != 403 || rejected.ResponseHeader.Get("X-Rejected") != "list-forbidden" || gets.Load() != 3 || lists.Load() != 2 {
		t.Fatal(failed, err, rejected, gets.Load(), lists.Load())
	}
	connectionReadCheckOperation(t, "VolumeExists", err)
}
