package gophercloudsdk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	v3 "github.com/gophercloud/gophercloud/v2/openstack/identity/v3/tokens"
)

type connectionTypeContractOutcome struct {
	value    json.RawMessage
	selected *resource.RawResource
	types    []*resource.RawResource
	observed *blockstorage.VolumeTypesPage
	pages    []*blockstorage.VolumeTypesPage
}

func connectionTypeContractCall(ctx context.Context, conn *sdk.Connection, operation, identity string, read []blockstorage.VolumeTypeReadOption, search []blockstorage.VolumeTypeSearchOption) (*connectionTypeContractOutcome, error) {
	switch operation {
	case "ListVolumeTypes":
		result, err := conn.ListVolumeTypes(ctx, read...)
		if result == nil {
			return nil, err
		}
		return &connectionTypeContractOutcome{value: result.Value, types: result.Types, pages: result.Pages}, err
	case "SearchVolumeTypes":
		result, err := conn.SearchVolumeTypes(ctx, blockstorage.SearchVolumeTypesRequest{NameOrID: identity}, search...)
		if result == nil {
			return nil, err
		}
		return &connectionTypeContractOutcome{value: result.Value, types: result.Types, pages: result.Pages}, err
	default:
		result, err := conn.GetVolumeType(ctx, blockstorage.GetVolumeTypeRequest{NameOrID: identity}, search...)
		if result == nil {
			return nil, err
		}
		return &connectionTypeContractOutcome{value: result.Value, selected: result.Type, observed: result.Observed, pages: result.Pages}, err
	}
}

func connectionTypeContractOptions(callback func() error) ([]blockstorage.VolumeTypeReadOption, []blockstorage.VolumeTypeSearchOption) {
	return []blockstorage.VolumeTypeReadOption{func(*blockstorage.VolumeTypeReadOpts) error { return callback() }},
		[]blockstorage.VolumeTypeSearchOption{func(*blockstorage.VolumeTypeSearchOpts) error { return callback() }}
}

func connectionTypeContractCheckOperation(t *testing.T, operation string, err error) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "volume type" {
		t.Fatal("wrong volume-type operation context", operation, err, wrapped)
	}
}

func connectionTypeContractRow(t *testing.T, operation string, value json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	if operation != "GetVolumeType" {
		var rows []json.RawMessage
		if err := json.Unmarshal(value, &rows); err != nil || len(rows) != 1 {
			t.Fatal(string(value), err)
		}
		value = rows[0]
	}
	return connectionReadFields(t, value)
}

func TestConnectionVolumeTypesCachedCinderOriginalOptionsPrecedeSelection(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var callbacks, locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("original option must precede Cinder v3 selection", callbacks.Load(), opts)
					return "", errors.New("unexpected selection")
				}
				return cloud.Server.URL + "/connection-types/v3/project/", nil
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			var cached *gophercloud.ServiceClient
			cloud.Mux.HandleFunc("GET /connection-types/v3/project/types", func(w http.ResponseWriter, r *http.Request) {
				index := requests.Add(1)
				if operation == "GetVolumeType" || r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live-type-token" || (index == 2 && r.Header.Get("X-Original") != "after-original") {
					t.Error(r.URL, r.Header)
				}
				testcloud.JSON(w, 200, `{"volume_types":[{"id":"type","name":"data","extra_specs":{"n":9007199254740993}}]}`)
			})
			cloud.Mux.HandleFunc("GET /connection-types/v3/project/types/type", func(w http.ResponseWriter, r *http.Request) {
				index := requests.Add(1)
				if operation != "GetVolumeType" || r.URL.RawQuery != "is_public=none" || r.Header.Get("X-Auth-Token") != "live-type-token" || (index == 2 && r.Header.Get("X-Original") != "after-original") {
					t.Error(r.URL, r.Header)
				}
				testcloud.JSON(w, 200, `{"volume_type":{"id":"response-type","name":"data","extra_specs":{"n":9007199254740993}}}`)
			})
			for range 2 {
				read, search := connectionTypeContractOptions(func() error {
					callbacks.Add(1)
					cloud.Provider.SetToken("live-type-token")
					if cached != nil {
						cached.MoreHeaders = map[string]string{"X-Original": "after-original"}
					}
					return nil
				})
				result, err := connectionTypeContractCall(context.Background(), conn, operation, "type", read, search)
				if err != nil || result == nil || len(result.value) == 0 {
					t.Fatal(result, err)
				}
				if operation == "GetVolumeType" {
					if result.selected == nil || result.observed == nil || len(result.pages) != 0 || string(result.selected.Body["id"]) != `"response-type"` {
						t.Fatal(result)
					}
				} else if len(result.types) != 1 || len(result.pages) != 1 || result.observed != nil {
					t.Fatal(result)
				}
				raw := result.selected
				if operation != "GetVolumeType" {
					raw = result.types[0]
				}
				if string(raw.Body["extra_specs"]) != `{"n":9007199254740993}` {
					t.Fatal(raw)
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
			if callbacks.Load() != 2 || locates.Load() != 1 || requests.Load() != 2 {
				t.Fatal(callbacks.Load(), locates.Load(), requests.Load())
			}
		})
	}
}

func TestConnectionGetVolumeTypeDefaultAndExplicitFalseyFilterRoutes(t *testing.T) {
	cases := []struct {
		name     string
		filters  *json.RawMessage
		rejected int
		unsafe   bool
	}{
		{name: "omitted/400", rejected: 400}, {name: "omitted/403", rejected: 403}, {name: "omitted/404", rejected: 404},
		{name: "explicit null", filters: connectionTypeContractRaw(`null`), rejected: 404},
		{name: "empty object", filters: connectionTypeContractRaw(`{}`)}, {name: "empty array", filters: connectionTypeContractRaw(`[]`)},
		{name: "false", filters: connectionTypeContractRaw(`false`)}, {name: "zero", filters: connectionTypeContractRaw(`0`)},
		{name: "empty string", filters: connectionTypeContractRaw(`""`)}, {name: "unsafe exact name", unsafe: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-types/v3/project/"))
			if err != nil {
				t.Fatal(err)
			}
			calls := []string{}
			identity := "data"
			if tc.unsafe {
				identity = "folder/data"
			}
			cloud.Mux.HandleFunc("GET /connection-types/v3/project/types/", func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.URL.String())
				if tc.rejected == 0 || r.URL.Path != "/connection-types/v3/project/types/data" || r.URL.RawQuery != "is_public=none" {
					t.Error(r.URL)
				}
				status := tc.rejected
				if status == 0 {
					status = http.StatusInternalServerError
				}
				testcloud.JSON(w, status, `{"error":"member rejected"}`)
			})
			cloud.Mux.HandleFunc("GET /connection-types/v3/project/types", func(w http.ResponseWriter, r *http.Request) {
				calls = append(calls, r.URL.String())
				expected := ""
				if tc.rejected != 0 || tc.unsafe {
					expected = "is_public=none"
				}
				if r.URL.RawQuery != expected {
					t.Error("wrong find/list query", r.URL, expected)
				}
				name, _ := json.Marshal(identity)
				testcloud.JSON(w, 200, `{"volume_types":[{"id":"actual","name":`+string(name)+`}]}`)
			})
			options := []blockstorage.VolumeTypeSearchOption{}
			if tc.filters != nil {
				options = append(options, blockstorage.WithVolumeTypeSearchFilters(*tc.filters))
			}
			result, err := conn.GetVolumeType(context.Background(), blockstorage.GetVolumeTypeRequest{NameOrID: identity}, options...)
			expected := 1
			if tc.rejected != 0 {
				expected = 2
			}
			if err != nil || result == nil || result.Type == nil || len(result.Pages) != 1 || result.Observed != nil || len(calls) != expected {
				t.Fatal(result, err, calls)
			}
			if string(result.Type.Body["id"]) != `"actual"` {
				t.Fatal(result.Type)
			}
		})
	}
}

func connectionTypeContractRaw(value string) *json.RawMessage {
	raw := json.RawMessage(value)
	return &raw
}

func TestConnectionVolumeTypesLocationSnapshotAndListMemberPhase(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			if err := connectionReadRecordScope(cloud.Provider, "before-original"); err != nil {
				t.Fatal(err)
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-types/v3/project/"), sdk.WithRegion("RegionTwo"))
			if err != nil {
				t.Fatal(err)
			}
			requests, callbacks := 0, 0
			cloud.Mux.HandleFunc("GET /connection-types/v3/project/types", func(w http.ResponseWriter, r *http.Request) {
				requests++
				if err := connectionReadRecordScope(cloud.Provider, "during-http"); err != nil {
					t.Error(err)
				}
				testcloud.JSON(w, 200, `{"volume_types":[{"id":"type","project_id":"foreign","availability_zone":"wire-zone","location":{"wrong":"wire"}}]}`)
			})
			cloud.Mux.HandleFunc("GET /connection-types/v3/project/types/type", func(w http.ResponseWriter, r *http.Request) {
				requests++
				if err := connectionReadRecordScope(cloud.Provider, "during-http"); err != nil {
					t.Error(err)
				}
				testcloud.JSON(w, 200, `{"volume_type":{"id":"type","project_id":"foreign","availability_zone":"wire-zone","location":{"wrong":"wire"}}}`)
			})
			read, search := connectionTypeContractOptions(func() error { callbacks++; return connectionReadRecordScope(cloud.Provider, "after-original") })
			first, err := connectionTypeContractCall(context.Background(), conn, operation, "type", read, search)
			if err != nil || first == nil {
				t.Fatal(first, err)
			}
			second, err := connectionTypeContractCall(context.Background(), conn, operation, "type", nil, nil)
			if err != nil || second == nil {
				t.Fatal(second, err)
			}
			for i, result := range []*connectionTypeContractOutcome{first, second} {
				row := connectionTypeContractRow(t, operation, result.value)
				location := connectionReadFields(t, row["location"])
				project := connectionReadFields(t, location["project"])
				expected := `"after-original"`
				if i == 1 {
					expected = `"during-http"`
				}
				if string(project["id"]) != expected || string(project["name"]) != "null" || string(location["zone"]) != "null" || string(location["region_name"]) != `"RegionTwo"` || len(row) != 6 {
					t.Fatal(string(result.value))
				}
				raw := result.selected
				if operation != "GetVolumeType" {
					raw = result.types[0]
				}
				if string(raw.Body["project_id"]) != `"foreign"` || string(raw.Body["availability_zone"]) != `"wire-zone"` || string(raw.Body["location"]) != `{"wrong":"wire"}` {
					t.Fatal(raw)
				}
			}
			if callbacks != 1 || requests != 2 {
				t.Fatal(callbacks, requests)
			}
		})
	}
	t.Run("only computed wire location in list", func(t *testing.T) {
		cloud := testcloud.New(t)
		conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-types/v3/project/"))
		if err != nil {
			t.Fatal(err)
		}
		cloud.Mux.HandleFunc("GET /connection-types/v3/project/types", func(w http.ResponseWriter, r *http.Request) {
			testcloud.JSON(w, 200, `{"volume_types":[{"location":false,"metadata":{"unknown":true}},{"location":"wire-only"},{"description":null,"location":"discarded"}]}`)
		})
		result, err := conn.ListVolumeTypes(context.Background())
		if err != nil || result == nil || len(result.Types) != 3 {
			t.Fatal(result, err)
		}
		var rows []json.RawMessage
		if err := json.Unmarshal(result.Value, &rows); err != nil {
			t.Fatal(err)
		}
		if string(connectionReadFields(t, rows[0])["location"]) != "false" || string(connectionReadFields(t, rows[1])["location"]) != `"wire-only"` || connectionReadFields(t, connectionReadFields(t, rows[2])["location"])["project"] == nil {
			t.Fatal(string(result.Value))
		}
	})
}

func TestConnectionVolumeTypesOwnedFactoriesAndOriginalOptionSliceSnapshots(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		for _, factory := range []string{"location", "bulk", "callbacks"} {
			t.Run(operation+"/"+factory, func(t *testing.T) {
				cloud := testcloud.New(t)
				defaultName, callName, callCloud := "default-project", "call-project", "call-cloud"
				defaults := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"default"`), Name: &defaultName}}
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(defaults), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-types/v3/project/"))
				if err != nil {
					t.Fatal(err)
				}
				supplied := resource.CloudLocation{Cloud: &callCloud, Zone: json.RawMessage(`{"owned":true}`), Project: resource.CloudProject{ID: json.RawMessage(`"call"`), Name: &callName}}
				var read []blockstorage.VolumeTypeReadOption
				var search []blockstorage.VolumeTypeSearchOption
				if factory == "bulk" {
					read = []blockstorage.VolumeTypeReadOption{blockstorage.WithVolumeTypeReadOptions(blockstorage.VolumeTypeReadOpts{Location: &supplied})}
					search = []blockstorage.VolumeTypeSearchOption{blockstorage.WithVolumeTypeSearchOptions(blockstorage.VolumeTypeSearchOpts{Location: &supplied})}
				} else {
					read = []blockstorage.VolumeTypeReadOption{blockstorage.WithVolumeTypeReadLocation(supplied)}
					search = []blockstorage.VolumeTypeSearchOption{blockstorage.WithVolumeTypeSearchLocation(supplied)}
				}
				callbacks, injected := 0, 0
				if factory == "callbacks" {
					read = make([]blockstorage.VolumeTypeReadOption, 2)
					search = make([]blockstorage.VolumeTypeSearchOption, 2)
					read[0] = func(o *blockstorage.VolumeTypeReadOpts) error {
						callbacks++
						o.Location = &supplied
						read[1] = func(*blockstorage.VolumeTypeReadOpts) error { injected++; return nil }
						return nil
					}
					read[1] = func(*blockstorage.VolumeTypeReadOpts) error {
						callbacks++
						callName = "mutation-after-first"
						supplied.Zone[0] = '!'
						return nil
					}
					search[0] = func(o *blockstorage.VolumeTypeSearchOpts) error {
						callbacks++
						o.Location = &supplied
						search[1] = func(*blockstorage.VolumeTypeSearchOpts) error { injected++; return nil }
						return nil
					}
					search[1] = func(*blockstorage.VolumeTypeSearchOpts) error {
						callbacks++
						callName = "mutation-after-first"
						supplied.Zone[0] = '!'
						return nil
					}
				} else {
					callName = "caller-mutated"
					callCloud = "caller-mutated-cloud"
					supplied.Zone[0] = '!'
					supplied.Project.ID[0] = '!'
				}
				cloud.Mux.HandleFunc("GET /connection-types/v3/project/types", func(w http.ResponseWriter, r *http.Request) {
					testcloud.JSON(w, 200, `{"volume_types":[{"id":"type"}]}`)
				})
				cloud.Mux.HandleFunc("GET /connection-types/v3/project/types/type", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"volume_type":{"id":"type"}}`) })
				result, err := connectionTypeContractCall(context.Background(), conn, operation, "type", read, search)
				if err != nil || result == nil {
					t.Fatal(result, err)
				}
				location := connectionReadFields(t, connectionTypeContractRow(t, operation, result.value)["location"])
				project := connectionReadFields(t, location["project"])
				if string(location["cloud"]) != `"call-cloud"` || string(location["zone"]) != `{"owned":true}` || string(project["id"]) != `"call"` || string(project["name"]) != `"call-project"` {
					t.Fatal(string(result.value))
				}
				if factory == "callbacks" && (callbacks != 2 || injected != 0) {
					t.Fatal(callbacks, injected)
				}
				current, err := conn.CurrentLocation()
				if err != nil || current.Project.Name == nil || *current.Project.Name != "default-project" || string(current.Project.ID) != `"default"` {
					t.Fatal(current, err)
				}
			})
		}
	}
}

func TestConnectionVolumeTypesExpressionFactoriesOwnFiltersAndDoNotInventRawRows(t *testing.T) {
	for _, operation := range []string{"SearchVolumeTypes", "GetVolumeType"} {
		for _, factory := range []string{"filters", "expression", "bulk", "callbacks"} {
			t.Run(operation+"/"+factory, func(t *testing.T) {
				cloud := testcloud.New(t)
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-types/v3/project/"))
				if err != nil {
					t.Fatal(err)
				}
				expression := "[].name"
				if operation == "GetVolumeType" {
					expression = "[0].name"
				}
				encoded, _ := json.Marshal(expression)
				filters := json.RawMessage(encoded)
				var options []blockstorage.VolumeTypeSearchOption
				switch factory {
				case "filters":
					options = []blockstorage.VolumeTypeSearchOption{blockstorage.WithVolumeTypeSearchFilters(filters)}
				case "expression":
					options = []blockstorage.VolumeTypeSearchOption{blockstorage.WithVolumeTypeSearchExpression(expression)}
				case "bulk":
					options = []blockstorage.VolumeTypeSearchOption{blockstorage.WithVolumeTypeSearchOptions(blockstorage.VolumeTypeSearchOpts{Filters: &filters})}
				default:
					options = make([]blockstorage.VolumeTypeSearchOption, 2)
					options[0] = func(o *blockstorage.VolumeTypeSearchOpts) error {
						o.Filters = &filters
						options[1] = func(*blockstorage.VolumeTypeSearchOpts) error { return errors.New("replaced option must not run") }
						return nil
					}
					options[1] = func(*blockstorage.VolumeTypeSearchOpts) error { filters[0] = '!'; return nil }
				}
				if factory != "callbacks" {
					filters[0] = '!'
				}
				requests := 0
				cloud.Mux.HandleFunc("GET /connection-types/v3/project/types", func(w http.ResponseWriter, r *http.Request) {
					requests++
					if r.URL.RawQuery != "" {
						t.Error(r.URL)
					}
					testcloud.JSON(w, 200, `{"volume_types":[{"id":"type","name":"界"}]}`)
				})
				result, err := connectionTypeContractCall(context.Background(), conn, operation, "", nil, options)
				expected := `["界"]`
				if operation == "GetVolumeType" {
					expected = `"界"`
				}
				if err != nil || result == nil || string(result.value) != expected || result.selected != nil || len(result.types) != 0 || len(result.pages) != 1 || result.observed != nil || requests != 1 {
					t.Fatal(result, err, requests)
				}
			})
		}
	}
}

func TestConnectionVolumeTypesPreflightAndGetterFailuresKeepExactOperationContext(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		for _, kind := range []string{"nil context", "nil connection", "already canceled", "nil option", "option error", "option cancellation", "getter error", "getter cancellation"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, optionError, getterError := errors.New("type operation canceled"), errors.New("type original failed"), errors.New("type locator failed")
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
				read, search := connectionTypeContractOptions(func() error {
					callbacks++
					if kind == "option cancellation" {
						cancel(cause)
					}
					if kind == "option error" || kind == "option cancellation" {
						return optionError
					}
					return nil
				})
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
					read = append(read, nil)
					search = append(search, nil)
					wantCallbacks = 1
				case "option error":
					want = optionError
					wantCallbacks = 1
				case "option cancellation":
					want = context.Canceled
					wantCallbacks = 1
				default:
					want = getterError
					wantCallbacks = 1
					wantLocates = 1
				}
				result, err := connectionTypeContractCall(ctx, conn, operation, "type", read, search)
				if result != nil || !errors.Is(err, want) || callbacks != wantCallbacks || locates != wantLocates {
					t.Fatal(result, err, callbacks, locates)
				}
				connectionTypeContractCheckOperation(t, operation, err)
				if strings.Contains(kind, "cancellation") || kind == "already canceled" {
					if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
						t.Fatal("lost context cause", err)
					}
				}
				if kind == "option cancellation" && !errors.Is(err, optionError) {
					t.Fatal("lost original failure", err)
				}
			})
		}
	}
}

func TestConnectionVolumeTypesAcceptedProofSurvivesReadCloseCancellationAndSourceDrift(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		for _, kind := range []string{"read close cancellation", "source change"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-types/v3/project/"))
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
				readError, closeError, cause := errors.New("type body read failed"), errors.New("type body close failed"), errors.New("type close canceled")
				payload := []byte(`{"volume_types":[{"id":"type"}]}`)
				path := "/connection-types/v3/project/types"
				if operation == "GetVolumeType" {
					payload = []byte(`{"volume_type":{"id":"type"}}`)
					path += "/type"
				}
				calls, callbacks := 0, 0
				cloud.Provider.HTTPClient.Transport = connectionReadRoundTripper(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.URL.Path != path {
						t.Error(r.URL)
					}
					stream := &connectionReadBody{content: bytes.Clone(payload), close: func() {
						if kind == "source change" {
							client.Endpoint = cloud.Server.URL + "/changed/v3/project/"
						} else {
							cancel(cause)
						}
					}}
					if kind == "read close cancellation" {
						stream.readError = readError
						stream.closeError = closeError
					}
					return &http.Response{StatusCode: 200, Header: http.Header{"X-Accepted-Type": {"retained"}}, Body: stream, Request: r}, nil
				})
				read, search := connectionTypeContractOptions(func() error { callbacks++; return nil })
				result, err := connectionTypeContractCall(ctx, conn, operation, "type", read, search)
				if err == nil || result == nil || result.value != nil || result.selected != nil || len(result.types) != 0 || calls != 1 || callbacks != 1 {
					t.Fatal(result, err, calls, callbacks)
				}
				connectionTypeContractCheckOperation(t, operation, err)
				var evidence *blockstorage.VolumeTypesPage
				if operation == "GetVolumeType" {
					evidence = result.observed
					if len(result.pages) != 0 {
						t.Fatal(result)
					}
				} else {
					if len(result.pages) != 1 {
						t.Fatal(result)
					}
					evidence = result.pages[0]
				}
				if evidence == nil || evidence.StatusCode != 200 || evidence.Header.Get("X-Accepted-Type") != "retained" || !bytes.Equal(evidence.Body, payload) {
					t.Fatal(evidence, string(payload))
				}
				if kind == "read close cancellation" {
					for _, want := range []error{readError, closeError, cause, context.Canceled} {
						if !errors.Is(err, want) {
							t.Fatal("lost accepted failure", want, err)
						}
					}
				} else if !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestConnectionVolumeTypesInvalidRecordedScopePreventsSelectionButOwnedLocationWorks(t *testing.T) {
	for _, operation := range []string{"ListVolumeTypes", "SearchVolumeTypes", "GetVolumeType"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			recorded := v3.CreateResult{}
			recorded.Header = http.Header{"X-Subject-Token": {"recorded-token"}}
			recorded.Body = map[string]any{"token": map[string]any{"project": map[string]any{"id": 123}}}
			if err := cloud.Provider.SetTokenAndAuthResult(recorded); err != nil {
				t.Fatal(err)
			}
			callbacks, locates := 0, 0
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates++
				if callbacks != 2 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(callbacks, opts)
				}
				return cloud.Server.URL + "/connection-types/v3/project/", nil
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			read, search := connectionTypeContractOptions(func() error { callbacks++; return nil })
			result, err := connectionTypeContractCall(context.Background(), conn, operation, "type", read, search)
			if result != nil || !errors.Is(err, resource.ErrInvalidOption) || callbacks != 1 || locates != 0 {
				t.Fatal(result, err, callbacks, locates)
			}
			connectionTypeContractCheckOperation(t, operation, err)
			owned := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"owned-project"`)}}
			read = append(read, blockstorage.WithVolumeTypeReadLocation(owned))
			search = append(search, blockstorage.WithVolumeTypeSearchLocation(owned))
			cloud.Mux.HandleFunc("GET /connection-types/v3/project/types", func(w http.ResponseWriter, r *http.Request) {
				testcloud.JSON(w, 200, `{"volume_types":[{"id":"type"}]}`)
			})
			cloud.Mux.HandleFunc("GET /connection-types/v3/project/types/type", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"volume_type":{"id":"type"}}`) })
			result, err = connectionTypeContractCall(context.Background(), conn, operation, "type", read, search)
			if err != nil || result == nil || callbacks != 2 || locates != 1 {
				t.Fatal(result, err, callbacks, locates)
			}
			location := connectionReadFields(t, connectionTypeContractRow(t, operation, result.value)["location"])
			if string(connectionReadFields(t, location["project"])["id"]) != `"owned-project"` {
				t.Fatal(string(result.value))
			}
		})
	}
}
