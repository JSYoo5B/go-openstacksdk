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
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type connectionMutationContractOutcome struct {
	resolved *blockstorage.GetVolumeResult
	volumeID string
	applied  *blockstorage.VolumeMutationPage
	volume   *resource.RawResource
	value    json.RawMessage
}

func connectionMutationContractCall(ctx context.Context, conn *sdk.Connection, operation, identity string, update []blockstorage.UpdateVolumeOption, bootable []blockstorage.SetVolumeBootableOption) (*connectionMutationContractOutcome, error) {
	if operation == "UpdateVolume" {
		result, err := conn.UpdateVolume(ctx, blockstorage.UpdateVolumeRequest{NameOrID: identity}, update...)
		if result == nil {
			return nil, err
		}
		return &connectionMutationContractOutcome{resolved: result.Resolved, volumeID: result.VolumeID, applied: result.Applied, volume: result.Volume, value: result.Value}, err
	}
	result, err := conn.SetVolumeBootable(ctx, blockstorage.SetVolumeBootableRequest{NameOrID: identity}, bootable...)
	if result == nil {
		return nil, err
	}
	return &connectionMutationContractOutcome{resolved: result.Resolved, volumeID: result.VolumeID, applied: result.Applied}, err
}

func connectionMutationContractOptions(callback func() error) ([]blockstorage.UpdateVolumeOption, []blockstorage.SetVolumeBootableOption) {
	return []blockstorage.UpdateVolumeOption{func(opts *blockstorage.UpdateVolumeOpts) error {
		name := "changed"
		opts.Attributes.Name = &name
		return callback()
	}}, []blockstorage.SetVolumeBootableOption{func(*blockstorage.SetVolumeBootableOpts) error { return callback() }}
}

func connectionMutationContractResolved(t *testing.T, result *connectionMutationContractOutcome) {
	t.Helper()
	if result == nil || result.resolved == nil || result.resolved.Volume == nil || result.resolved.Observed == nil || result.resolved.Observed.StatusCode != 200 || len(result.resolved.Pages) != 0 || string(result.resolved.Volume.Body["id"]) != `"resolved-volume"` {
		t.Fatal("lookup proof was not preserved independently", result)
	}
}

func TestConnectionVolumeMutationsOriginalsPrecedeCachedCinderAndBindActualResponseID(t *testing.T) {
	for _, operation := range []string{"UpdateVolume", "SetVolumeBootable"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			var callbacks, locates, requests atomic.Int32
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates.Add(1)
				if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error("originals must precede only Cinder v3 selection", callbacks.Load(), opts)
					return "", errors.New("unexpected service selection")
				}
				return cloud.Server.URL + "/connection-mutations/v3/project/", nil
			}
			conn, err := sdk.FromProvider(cloud.Provider)
			if err != nil {
				t.Fatal(err)
			}
			var cached *gophercloud.ServiceClient
			cloud.Mux.HandleFunc("GET /connection-mutations/v3/project/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
				index := requests.Add(1)
				if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live-token" || index == 3 && r.Header.Get("X-Original") != "after-original" {
					t.Error(r.URL, r.Header)
				}
				w.Header().Set("X-Lookup", "actual")
				testcloud.JSON(w, 200, `{"volume":{"id":"resolved-volume","name":"old","description":"prior","metadata":{"kept":"prior"},"unknown_prior":9007199254740993}}`)
			})
			mutationMethod, mutationPath := http.MethodPut, "/connection-mutations/v3/project/volumes/resolved-volume"
			if operation == "SetVolumeBootable" {
				mutationMethod, mutationPath = http.MethodPost, mutationPath+"/action"
			}
			cloud.Mux.HandleFunc(mutationMethod+" "+mutationPath, func(w http.ResponseWriter, r *http.Request) {
				index := requests.Add(1)
				if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live-token" || index == 4 && r.Header.Get("X-Original") != "after-original" {
					t.Error(r.URL, r.Header)
				}
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					return
				}
				fields := connectionReadFields(t, raw)
				w.Header().Set("X-Mutation", "actual")
				if operation == "UpdateVolume" {
					body := connectionReadFields(t, fields["volume"])
					if len(fields) != 1 || len(body) != 1 || string(body["name"]) != `"changed"` {
						t.Error("only dirty fields should be sent", string(raw))
					}
					testcloud.JSON(w, 203, `{"volume":{"description":null,"server_only":"ignored"}}`)
				} else {
					body := connectionReadFields(t, fields["os-set_bootable"])
					want := "true"
					if index == 4 {
						want = "false"
					}
					if len(fields) != 1 || len(body) != 1 || string(body["bootable"]) != want {
						t.Error(string(raw), want)
					}
					w.WriteHeader(203)
					_, _ = w.Write([]byte{0xff, 0x00, 'a'})
				}
			})
			for index := range 2 {
				identity := "volume"
				update, boot := connectionMutationContractOptions(func() error {
					callbacks.Add(1)
					cloud.Provider.SetToken("live-token")
					identity = "callback-mutated"
					if cached != nil {
						cached.MoreHeaders = map[string]string{"X-Original": "after-original"}
					}
					return nil
				})
				if index == 1 {
					boot = append(boot, blockstorage.WithSetVolumeBootable(false))
				}
				result, err := connectionMutationContractCall(context.Background(), conn, operation, identity, update, boot)
				if err != nil {
					t.Fatal(result, err)
				}
				connectionMutationContractResolved(t, result)
				if result.volumeID != "resolved-volume" || result.applied == nil || result.applied.StatusCode != 203 || result.applied.Header.Get("X-Mutation") != "actual" || result.resolved.Observed.Header.Get("X-Lookup") != "actual" {
					t.Fatal(result)
				}
				if operation == "UpdateVolume" {
					if result.volume == nil || string(result.volume.Body["name"]) != `"changed"` || string(result.volume.Body["description"]) != "null" || string(result.volume.Body["unknown_prior"]) != "9007199254740993" || result.volume.Body["server_only"] != nil || string(result.resolved.Volume.Body["name"]) != `"old"` || result.volume.StatusCode != 203 {
						t.Fatal(result)
					}
					view := connectionReadFields(t, result.value)
					if string(view["name"]) != `"changed"` || string(view["description"]) != "null" || string(view["metadata"]) != `{"kept":"prior"}` {
						t.Fatal(string(result.value))
					}
					proof := bytes.Clone(result.applied.Body)
					result.value[0] = '!'
					result.volume.Body["name"][0] = '!'
					result.volume.Header.Set("X-Mutation", "caller-mutated")
					if !bytes.Equal(result.applied.Body, proof) || result.applied.Header.Get("X-Mutation") != "actual" || string(result.resolved.Volume.Body["name"]) != `"old"` {
						t.Fatal("merged outputs alias raw phase proof")
					}
				} else if result.value != nil || result.volume != nil || !bytes.Equal(result.applied.Body, []byte{0xff, 0x00, 'a'}) {
					t.Fatal(result)
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

func TestConnectionUpdateVolumeNoOpKeepsLookupAndSkipsMutation(t *testing.T) {
	for _, kind := range []string{"empty", "equal name", "equal bool number", "ignored unknown"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-mutations/v3/project/"))
			if err != nil {
				t.Fatal(err)
			}
			var requests atomic.Int32
			cloud.Mux.HandleFunc("GET /connection-mutations/v3/project/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				testcloud.JSON(w, 200, `{"volume":{"id":"resolved-volume","name":"old","multiattach":true,"unknown_prior":{"kept":1}}}`)
			})
			var callbacks int
			options := []blockstorage.UpdateVolumeOption{func(*blockstorage.UpdateVolumeOpts) error { callbacks++; return nil }}
			switch kind {
			case "equal name":
				options = append(options, blockstorage.WithUpdateVolumeName("old"))
			case "equal bool number":
				options = append(options, blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{"is_multiattach": json.RawMessage("1")}))
			case "ignored unknown":
				options = append(options, blockstorage.WithUpdateVolumeFields(map[string]json.RawMessage{"unknown": json.RawMessage(`{"new":2}`), "source_replica": json.RawMessage(`"ignored"`)}))
			}
			result, err := conn.UpdateVolume(context.Background(), blockstorage.UpdateVolumeRequest{NameOrID: "volume"}, options...)
			if err != nil || result == nil || result.VolumeID != "resolved-volume" || result.Applied != nil || result.Volume == nil || len(result.Value) == 0 || callbacks != 1 || requests.Load() != 1 {
				t.Fatal(result, err, callbacks, requests.Load())
			}
			if string(result.Volume.Body["multiattach"]) != "true" || result.Volume.Body["unknown"] != nil || string(result.Volume.Body["unknown_prior"]) != `{"kept":1}` {
				t.Fatal("equal raw state or prior unknown field changed", result.Volume)
			}
			view := connectionReadFields(t, result.Value)
			if string(view["is_multiattach"]) != "true" || string(view["name"]) != `"old"` {
				t.Fatal(string(result.Value))
			}
		})
	}
}

func TestConnectionVolumeMutationsOwnFactoryAndRetainedCallbackState(t *testing.T) {
	for _, operation := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, factory := range []string{"bulk", "individual", "callbacks"} {
			t.Run(operation+"/"+factory, func(t *testing.T) {
				cloud := testcloud.New(t)
				defaultName, name, callCloud := "connection-default", "owned-name", "owned-cloud"
				defaults := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &defaultName}}
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithCloudLocation(defaults), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-mutations/v3/project/"))
				if err != nil {
					t.Fatal(err)
				}
				location := resource.CloudLocation{Cloud: &callCloud, Project: resource.CloudProject{ID: json.RawMessage(`"scope"`), Name: &name}}
				metadata := map[string]string{"kept": "owned"}
				fields := map[string]json.RawMessage{"size": json.RawMessage("4"), "image_id": json.RawMessage(`"owned-image"`)}
				attrs := blockstorage.UpdateVolumeAttributes{Name: &name, Metadata: metadata, Fields: fields}
				enabled := false
				var update []blockstorage.UpdateVolumeOption
				var boot []blockstorage.SetVolumeBootableOption
				first, second, injected := 0, 0, 0
				mutateCaller := func() {
					name, callCloud = "caller-mutated", "caller-mutated-cloud"
					metadata["kept"] = "caller-mutated"
					fields["size"][0] = '!'
					location.Project.ID[0] = '!'
					enabled = true
				}
				switch factory {
				case "bulk":
					update = []blockstorage.UpdateVolumeOption{blockstorage.WithUpdateVolumeOptions(blockstorage.UpdateVolumeOpts{Attributes: attrs, Location: &location})}
					boot = []blockstorage.SetVolumeBootableOption{blockstorage.WithSetVolumeBootableOptions(blockstorage.SetVolumeBootableOpts{Bootable: &enabled, Location: &location})}
				case "individual":
					update = []blockstorage.UpdateVolumeOption{blockstorage.WithUpdateVolumeAttributes(attrs), blockstorage.WithUpdateVolumeLocation(location)}
					boot = []blockstorage.SetVolumeBootableOption{blockstorage.WithSetVolumeBootable(enabled), blockstorage.WithSetVolumeBootableLocation(location)}
				default:
					update = make([]blockstorage.UpdateVolumeOption, 2)
					boot = make([]blockstorage.SetVolumeBootableOption, 2)
					update[0] = func(opts *blockstorage.UpdateVolumeOpts) error {
						first++
						opts.Attributes, opts.Location = attrs, &location
						update[1] = func(*blockstorage.UpdateVolumeOpts) error { injected++; return errors.New("caller replacement ran") }
						return nil
					}
					update[1] = func(opts *blockstorage.UpdateVolumeOpts) error {
						second++
						mutateCaller()
						if opts.Attributes.Name == nil || *opts.Attributes.Name != "owned-name" || opts.Attributes.Metadata["kept"] != "owned" || string(opts.Attributes.Fields["size"]) != "4" || opts.Location == nil || string(opts.Location.Project.ID) != `"scope"` {
							t.Error("prior callback state is not owned", opts)
						}
						return nil
					}
					boot[0] = func(opts *blockstorage.SetVolumeBootableOpts) error {
						first++
						opts.Bootable, opts.Location = &enabled, &location
						boot[1] = func(*blockstorage.SetVolumeBootableOpts) error {
							injected++
							return errors.New("caller replacement ran")
						}
						return nil
					}
					boot[1] = func(opts *blockstorage.SetVolumeBootableOpts) error {
						second++
						mutateCaller()
						if opts.Bootable == nil || *opts.Bootable || opts.Location == nil || string(opts.Location.Project.ID) != `"scope"` || opts.Location.Cloud == nil || *opts.Location.Cloud != "owned-cloud" {
							t.Error("prior callback state is not owned", opts)
						}
						return nil
					}
				}
				if factory != "callbacks" {
					mutateCaller()
				}
				cloud.Mux.HandleFunc("GET /connection-mutations/v3/project/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
					testcloud.JSON(w, 200, `{"volume":{"id":"resolved-volume","name":"old","project_id":"scope","availability_zone":"wire-zone"}}`)
				})
				method, path := http.MethodPut, "/connection-mutations/v3/project/volumes/resolved-volume"
				if operation == "SetVolumeBootable" {
					method, path = http.MethodPost, path+"/action"
				}
				cloud.Mux.HandleFunc(method+" "+path, func(w http.ResponseWriter, r *http.Request) {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Error(err)
						return
					}
					top := connectionReadFields(t, body)
					if operation == "UpdateVolume" {
						volume := connectionReadFields(t, top["volume"])
						if len(top) != 1 || len(volume) != 4 || string(volume["name"]) != `"owned-name"` || string(volume["size"]) != "4" || string(volume["imageRef"]) != `"owned-image"` || string(connectionReadFields(t, volume["metadata"])["kept"]) != `"owned"` {
							t.Error(string(body))
						}
						testcloud.JSON(w, 200, `{"volume":{}}`)
					} else {
						if string(connectionReadFields(t, top["os-set_bootable"])["bootable"]) != "false" {
							t.Error(string(body))
						}
						w.WriteHeader(204)
					}
				})
				result, err := connectionMutationContractCall(context.Background(), conn, operation, "volume", update, boot)
				if err != nil {
					t.Fatal(result, err)
				}
				connectionMutationContractResolved(t, result)
				view := result.resolved.Value
				if operation == "UpdateVolume" {
					view = result.value
				}
				computed := connectionReadFields(t, connectionReadFields(t, view)["location"])
				project := connectionReadFields(t, computed["project"])
				if string(computed["cloud"]) != `"owned-cloud"` || string(computed["zone"]) != `"wire-zone"` || string(project["id"]) != `"scope"` || string(project["name"]) != `"owned-name"` {
					t.Fatal(string(view))
				}
				if factory == "callbacks" && (first != 1 || second != 1 || injected != 0) {
					t.Fatal(first, second, injected)
				}
				current, err := conn.CurrentLocation()
				if err != nil || current.Cloud != nil || current.Project.Name == nil || *current.Project.Name != "connection-default" || string(current.Project.ID) != `"scope"` {
					t.Fatal("operation changed defaults", current, err)
				}
			})
		}
	}
}

func TestConnectionVolumeMutationsRecordedScopeIsCapturedAfterOriginalsThroughMutation(t *testing.T) {
	for _, operation := range []string{"UpdateVolume", "SetVolumeBootable"} {
		t.Run(operation, func(t *testing.T) {
			cloud := testcloud.New(t)
			if err := connectionReadRecordScope(cloud.Provider, "before-original"); err != nil {
				t.Fatal(err)
			}
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("owned-region"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-mutations/v3/project/"))
			if err != nil {
				t.Fatal(err)
			}
			var requests atomic.Int32
			cloud.Mux.HandleFunc("GET /connection-mutations/v3/project/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if err := connectionReadRecordScope(cloud.Provider, "between-stages"); err != nil {
					t.Error(err)
				}
				testcloud.JSON(w, 200, `{"volume":{"id":"resolved-volume","name":"old"}}`)
			})
			method, path := http.MethodPut, "/connection-mutations/v3/project/volumes/resolved-volume"
			if operation == "SetVolumeBootable" {
				method, path = http.MethodPost, path+"/action"
			}
			cloud.Mux.HandleFunc(method+" "+path, func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(204) })
			callbacks := 0
			for index := range 2 {
				update, boot := connectionMutationContractOptions(func() error {
					callbacks++
					if index == 0 {
						return connectionReadRecordScope(cloud.Provider, "after-original")
					}
					return nil
				})
				result, err := connectionMutationContractCall(context.Background(), conn, operation, "volume", update, boot)
				if err != nil {
					t.Fatal(result, err)
				}
				connectionMutationContractResolved(t, result)
				want := `"after-original"`
				if index == 1 {
					want = `"between-stages"`
				}
				views := []json.RawMessage{result.resolved.Value}
				if operation == "UpdateVolume" {
					views = append(views, result.value)
				}
				for _, view := range views {
					location := connectionReadFields(t, connectionReadFields(t, view)["location"])
					project := connectionReadFields(t, location["project"])
					if string(project["id"]) != want || string(project["name"]) != "null" || string(location["region_name"]) != `"owned-region"` {
						t.Fatal("scope reread between stages", string(view))
					}
				}
			}
			if callbacks != 2 || requests.Load() != 4 {
				t.Fatal(callbacks, requests.Load())
			}
		})
	}
}

func TestConnectionVolumeMutationsPreflightAndGetterFailuresKeepOuterOperationAndCause(t *testing.T) {
	for _, operation := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, kind := range []string{"nil context", "nil connection", "already canceled", "nil option", "option error", "option cancellation", "getter error", "getter cancellation"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				cause, optionError, getterError := errors.New("mutation canceled"), errors.New("mutation original failed"), errors.New("mutation locator failed")
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
				update, boot := connectionMutationContractOptions(func() error {
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
					update = append(update, nil)
					boot = append(boot, nil)
					wantCallbacks = 1
				case "option error":
					want = optionError
					wantCallbacks = 1
				case "option cancellation":
					want = context.Canceled
					wantCallbacks = 1
				default:
					want = getterError
					wantCallbacks, wantLocates = 1, 1
				}
				result, err := connectionMutationContractCall(ctx, conn, operation, "volume", update, boot)
				if result != nil || !errors.Is(err, want) || callbacks != wantCallbacks || locates != wantLocates {
					t.Fatal(result, err, callbacks, locates)
				}
				connectionReadCheckOperation(t, operation, err)
				if strings.Contains(kind, "cancellation") || kind == "already canceled" {
					if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) {
						t.Fatal("lost cancel cause", err)
					}
				}
				if kind == "option cancellation" && !errors.Is(err, optionError) {
					t.Fatal("lost original failure", err)
				}
			})
		}
	}
}

func TestConnectionVolumeMutationsAcceptedAndRejectedSecondStageProofStaysDistinct(t *testing.T) {
	for _, operation := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, kind := range []string{"read close cancellation", "source change", "native forbidden"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-mutations/v3/project/"))
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
				readError, closeError, cause := errors.New("mutation read failed"), errors.New("mutation close failed"), errors.New("mutation close canceled")
				payload := []byte(`{"volume":{"description":"accepted"}}`)
				if operation == "SetVolumeBootable" {
					payload = []byte{0xff, 0x00, 'a'}
				}
				calls, callbacks := 0, 0
				cloud.Provider.HTTPClient.Transport = connectionReadRoundTripper(func(r *http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						if r.Method != http.MethodGet || r.URL.Path != "/connection-mutations/v3/project/volumes/volume" || r.URL.RawQuery != "" {
							t.Error(r.Method, r.URL)
						}
						return &http.Response{StatusCode: 200, Header: http.Header{"X-Lookup": {"complete"}}, Body: io.NopCloser(strings.NewReader(`{"volume":{"id":"resolved-volume","name":"old"}}`)), Request: r}, nil
					}
					method, path := http.MethodPut, "/connection-mutations/v3/project/volumes/resolved-volume"
					if operation == "SetVolumeBootable" {
						method, path = http.MethodPost, path+"/action"
					}
					if calls != 2 || r.Method != method || r.URL.Path != path || r.URL.RawQuery != "" {
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
					return &http.Response{StatusCode: 203, Header: http.Header{"X-Mutation": {"actual"}}, Body: stream, Request: r}, nil
				})
				update, boot := connectionMutationContractOptions(func() error { callbacks++; return nil })
				result, err := connectionMutationContractCall(ctx, conn, operation, "volume", update, boot)
				if err == nil || calls != 2 || callbacks != 1 {
					t.Fatal(result, err, calls, callbacks)
				}
				connectionMutationContractResolved(t, result)
				connectionReadCheckOperation(t, operation, err)
				if result.volume != nil || result.value != nil || result.resolved.Observed.Header.Get("X-Lookup") != "complete" {
					t.Fatal("logical result or lookup proof changed after failure", result)
				}
				var accepted *resource.ResponseError
				if kind == "native forbidden" {
					var native gophercloud.ErrUnexpectedResponseCode
					if result.applied != nil || !errors.As(err, &native) || native.Actual != 403 || string(native.Body) != `{"error":"denied"}` || native.ResponseHeader.Get("X-Rejected") != "actual" || errors.As(err, &accepted) {
						t.Fatal("rejected mutation borrowed lookup proof", result, err)
					}
				} else {
					if result.applied == nil || result.applied.StatusCode != 203 || result.applied.Header.Get("X-Mutation") != "actual" || !bytes.Equal(result.applied.Body, payload) || !errors.As(err, &accepted) || accepted.StatusCode != 203 {
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

func TestConnectionVolumeMutationsLocalTargetAndLocationFailuresKeepLookupWithoutMutation(t *testing.T) {
	for _, operation := range []string{"UpdateVolume", "SetVolumeBootable"} {
		for _, kind := range []string{"absent", "missing response ID", "unsafe response ID", "invalid owned location"} {
			t.Run(operation+"/"+kind, func(t *testing.T) {
				cloud := testcloud.New(t)
				conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-mutations/v3/project/"))
				if err != nil {
					t.Fatal(err)
				}
				var requests atomic.Int32
				cloud.Mux.HandleFunc("GET /connection-mutations/v3/project/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if kind == "absent" {
						testcloud.JSON(w, 404, `{"error":"not found"}`)
						return
					}
					row := `{"id":"resolved-volume","name":"old"}`
					if kind == "missing response ID" {
						row = `{}`
					}
					if kind == "unsafe response ID" {
						row = `{"id":"../escape","name":"old"}`
					}
					testcloud.JSON(w, 200, `{"volume":`+row+`}`)
				})
				cloud.Mux.HandleFunc("GET /connection-mutations/v3/project/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					if kind != "absent" {
						t.Error(kind, r.URL)
					}
					testcloud.JSON(w, 200, `{"volumes":[]}`)
				})
				update, boot := connectionMutationContractOptions(func() error { return nil })
				if kind == "invalid owned location" {
					location := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage("!")}}
					update = append(update, blockstorage.WithUpdateVolumeLocation(location))
					boot = append(boot, blockstorage.WithSetVolumeBootableLocation(location))
				}
				result, err := connectionMutationContractCall(context.Background(), conn, operation, "volume", update, boot)
				want := resource.ErrInvalidOption
				count := int32(1)
				if kind == "absent" {
					want, count = resource.ErrNotFound, 2
				}
				if result == nil || result.resolved == nil || !errors.Is(err, want) || requests.Load() != count || result.applied != nil || result.volume != nil || result.value != nil {
					t.Fatal(result, err, requests.Load())
				}
				connectionReadCheckOperation(t, operation, err)
				var accepted *resource.ResponseError
				if errors.As(err, &accepted) {
					t.Fatal("local error borrowed accepted lookup response", err, accepted)
				}
				if kind == "absent" {
					if result.resolved.Volume != nil || result.resolved.Observed != nil || len(result.resolved.Pages) != 1 || result.resolved.Pages[0].StatusCode != 200 {
						t.Fatal(result)
					}
				} else if result.resolved.Observed == nil || result.resolved.Observed.StatusCode != 200 {
					t.Fatal("lookup proof lost", result)
				}
			})
		}
	}
}
