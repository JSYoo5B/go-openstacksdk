package gophercloudsdk_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync/atomic"
	"testing"

	sdk "github.com/JSYoo5B/gophercloudsdk"
	"github.com/JSYoo5B/gophercloudsdk/blockstorage"
	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionGetVolumeIDCachedCinderOriginalOptionsAndUntypedResponseIDs(t *testing.T) {
	cloud := testcloud.New(t)
	var callbacks, locates, gets atomic.Int32
	cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
		locates.Add(1)
		if callbacks.Load() != 1 || opts.Type != "block-storage" || opts.Version != 3 {
			t.Error("original options must precede Cinder-only selection", callbacks.Load(), opts)
			return "", errors.New("unexpected service selection")
		}
		return cloud.Server.URL + "/connection-volume-id/v3/project/", nil
	}
	conn, err := sdk.FromProvider(cloud.Provider)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{`"response-id"`, `null`, `false`, `0`, `""`, `[ false, 9007199254740993 ]`, `{"nested":{"id":9007199254740993}}`, "missing"}
	cloud.Mux.HandleFunc("GET /connection-volume-id/v3/project/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
		index := int(gets.Add(1)) - 1
		if index >= len(ids) {
			t.Error("unexpected workflow replay")
			w.WriteHeader(500)
			return
		}
		if r.URL.RawQuery != "" || r.Header.Get("X-Auth-Token") != "live-token" || (index > 0 && r.Header.Get("X-Original") != "after-callback") {
			t.Error(r.URL, r.Header)
		}
		w.Header().Set("X-Proof", "member")
		row := `{"name":"data","unknown_precision":9007199254740993}`
		if ids[index] != "missing" {
			row = `{"id":` + ids[index] + `,"name":"data","unknown_precision":9007199254740993}`
		}
		testcloud.JSON(w, http.StatusOK, `{"volume":`+row+`}`)
	})
	var cached *gophercloud.ServiceClient
	for _, id := range ids {
		input := blockstorage.GetVolumeIDRequest{NameOrID: "volume"}
		result, err := conn.GetVolumeID(context.Background(), input, func(*blockstorage.VolumeReadOpts) error {
			callbacks.Add(1)
			input.NameOrID = "caller-mutated"
			cloud.Provider.SetToken("live-token")
			if cached != nil {
				cached.MoreHeaders = map[string]string{"X-Original": "after-callback"}
			}
			return nil
		})
		want := id
		if id == "missing" {
			want = "null"
		}
		if err != nil || result == nil || string(result.ID) != want || result.Volume == nil || result.Value == nil || result.Observed == nil || len(result.Pages) != 0 {
			t.Fatal(id, result, err)
		}
		fields := connectionReadFields(t, result.Value)
		var normalized, wanted bytes.Buffer
		if err := json.Compact(&normalized, fields["id"]); err != nil {
			t.Fatal(err)
		}
		if err := json.Compact(&wanted, []byte(want)); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(normalized.Bytes(), wanted.Bytes()) || string(result.Volume.Body["unknown_precision"]) != "9007199254740993" || result.Volume.StatusCode != 200 || result.Observed.StatusCode != 200 || result.Observed.Header.Get("X-Proof") != "member" {
			t.Fatal(result, string(result.Value))
		}
		if id == "missing" {
			if _, present := result.Volume.Body["id"]; present {
				t.Fatal("request ID fabricated as response field", result.Volume.Body)
			}
		} else if string(result.Volume.Body["id"]) != id {
			t.Fatal("wire ID changed", result.Volume.Body)
		}
		beforeValue, beforeBody, beforeProof := bytes.Clone(result.Value), bytes.Clone(result.Volume.Body["id"]), bytes.Clone(result.Observed.Body)
		result.ID[0] = '!'
		if !bytes.Equal(beforeValue, result.Value) || !bytes.Equal(beforeBody, result.Volume.Body["id"]) || !bytes.Equal(beforeProof, result.Observed.Body) {
			t.Fatal("ID aliases lookup payloads")
		}
		result.Volume.Header.Set("X-Proof", "caller-mutated")
		if result.Observed.Header.Get("X-Proof") != "member" {
			t.Fatal("raw metadata aliases observed proof")
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
	if callbacks.Load() != int32(len(ids)) || gets.Load() != int32(len(ids)) || locates.Load() != 1 {
		t.Fatal(callbacks.Load(), gets.Load(), locates.Load())
	}
}

func TestConnectionGetVolumeIDLocationSnapshotsAndOwnedOverrides(t *testing.T) {
	cloud := testcloud.New(t)
	if err := connectionReadRecordScope(cloud.Provider, "before-options"); err != nil {
		t.Fatal(err)
	}
	conn, err := sdk.FromProvider(cloud.Provider, sdk.WithRegion("region"), sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-volume-id/v3/project/"))
	if err != nil {
		t.Fatal(err)
	}
	var gets atomic.Int32
	cloud.Mux.HandleFunc("GET /connection-volume-id/v3/project/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
		gets.Add(1)
		if err := connectionReadRecordScope(cloud.Provider, "in-flight"); err != nil {
			t.Error(err)
		}
		testcloud.JSON(w, 200, `{"volume":{"id":"response-id","project_id":false,"availability_zone":null}}`)
	})
	input := blockstorage.GetVolumeIDRequest{NameOrID: "volume"}
	result, err := conn.GetVolumeID(context.Background(), input, func(*blockstorage.VolumeReadOpts) error {
		return connectionReadRecordScope(cloud.Provider, "after-options")
	})
	if err != nil || result == nil {
		t.Fatal(result, err)
	}
	location := connectionReadLocation(t, "GetVolumeID", result.Value)
	project := connectionReadFields(t, location["project"])
	if string(project["id"]) != `"after-options"` || string(project["name"]) != "null" || string(location["region_name"]) != `"region"` || string(location["cloud"]) != "null" {
		t.Fatal(string(result.Value))
	}
	result, err = conn.GetVolumeID(context.Background(), input)
	if err != nil || result == nil {
		t.Fatal(result, err)
	}
	project = connectionReadFields(t, connectionReadLocation(t, "GetVolumeID", result.Value)["project"])
	if string(project["id"]) != `"in-flight"` {
		t.Fatal("next operation did not read current scope", string(result.Value))
	}

	for _, factory := range []string{"location", "bulk"} {
		name, cloudName := "owned-project", "owned-cloud"
		locationInput := resource.CloudLocation{Cloud: &cloudName, Project: resource.CloudProject{ID: json.RawMessage(`"owned-scope"`), Name: &name}}
		option := blockstorage.WithVolumeReadLocation(locationInput)
		if factory == "bulk" {
			option = blockstorage.WithVolumeReadOptions(blockstorage.VolumeReadOpts{Location: &locationInput})
		}
		name, cloudName = "caller-mutated", "caller-mutated"
		locationInput.Project.ID[0] = '!'
		for range 2 {
			result, err = conn.GetVolumeID(context.Background(), input, option)
			if err != nil || result == nil {
				t.Fatal(factory, result, err)
			}
			location = connectionReadLocation(t, "GetVolumeID", result.Value)
			project = connectionReadFields(t, location["project"])
			if string(location["cloud"]) != `"owned-cloud"` || string(project["id"]) != `"owned-scope"` || string(project["name"]) != `"owned-project"` {
				t.Fatal(factory, string(result.Value))
			}
			result.Value[0] = '!'
		}
	}
	current, err := conn.CurrentLocation()
	if err != nil || string(current.Project.ID) != `"in-flight"` || current.Cloud != nil || current.Project.Name != nil || gets.Load() != 6 {
		t.Fatal(current, err, gets.Load())
	}

	retainedName := "owned-callback"
	retained := resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`"callback-scope"`), Name: &retainedName}}
	first, second, replacement := 0, 0, 0
	var options []blockstorage.VolumeReadOption
	options = []blockstorage.VolumeReadOption{
		func(target *blockstorage.VolumeReadOpts) error {
			first++
			target.Location = &retained
			options[1] = func(*blockstorage.VolumeReadOpts) error { replacement++; return errors.New("caller replacement") }
			return nil
		},
		func(target *blockstorage.VolumeReadOpts) error {
			second++
			retainedName = "caller-mutated"
			retained.Project.ID[0] = '!'
			if target.Location == nil || target.Location.Project.Name == nil || *target.Location.Project.Name != "owned-callback" || string(target.Location.Project.ID) != `"callback-scope"` {
				t.Error(target)
			}
			return nil
		},
	}
	result, err = conn.GetVolumeID(context.Background(), input, options...)
	if err != nil || result == nil || first != 1 || second != 1 || replacement != 0 {
		t.Fatal(result, err, first, second, replacement)
	}
	project = connectionReadFields(t, connectionReadLocation(t, "GetVolumeID", result.Value)["project"])
	if string(project["name"]) != `"owned-callback"` || string(project["id"]) != `"callback-scope"` {
		t.Fatal(string(result.Value))
	}
}

func TestConnectionGetVolumeIDPreflightOptionAndGetterFailuresKeepOperationContext(t *testing.T) {
	for _, kind := range []string{"nil context", "nil connection", "already canceled", "nil option", "option error", "callback cancellation", "getter error", "getter cancellation"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause, rejected, getterError := errors.New("caller canceled ID lookup"), errors.New("ID option rejected"), errors.New("Cinder getter failed")
			calls, locates := 0, 0
			cloud.Provider.EndpointLocator = func(opts gophercloud.EndpointOpts) (string, error) {
				locates++
				if calls != 1 || opts.Type != "block-storage" || opts.Version != 3 {
					t.Error(calls, opts)
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
			current, activeContext := conn, context.Context(ctx)
			options := []blockstorage.VolumeReadOption{func(*blockstorage.VolumeReadOpts) error { calls++; return nil }}
			want, wantCalls, wantLocates := resource.ErrInvalidOption, 0, 0
			switch kind {
			case "nil context":
				activeContext = nil
			case "nil connection":
				current = nil
			case "already canceled":
				cancel(cause)
				want = context.Canceled
			case "nil option":
				options = append(options, nil)
				wantCalls = 1
			case "option error":
				options[0] = func(*blockstorage.VolumeReadOpts) error { calls++; return rejected }
				want, wantCalls = rejected, 1
			case "callback cancellation":
				options = []blockstorage.VolumeReadOption{func(*blockstorage.VolumeReadOpts) error { calls++; cancel(cause); return rejected }, func(*blockstorage.VolumeReadOpts) error { calls++; return nil }}
				want, wantCalls = context.Canceled, 1
			case "getter error", "getter cancellation":
				want, wantCalls, wantLocates = getterError, 1, 1
			}
			result, err := current.GetVolumeID(activeContext, blockstorage.GetVolumeIDRequest{NameOrID: "volume"}, options...)
			if result != nil || !errors.Is(err, want) || calls != wantCalls || locates != wantLocates {
				t.Fatal(result, err, calls, locates)
			}
			connectionReadCheckOperation(t, "GetVolumeID", err)
			if kind == "already canceled" || kind == "callback cancellation" || kind == "getter cancellation" {
				if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
					t.Fatal("cancellation cause lost", err)
				}
			}
			if kind == "callback cancellation" && !errors.Is(err, rejected) {
				t.Fatal("original callback error lost", err)
			}
		})
	}
}

func TestConnectionGetVolumeIDAcceptedProofSurvivesReadCloseCancellationAndSourceDrift(t *testing.T) {
	for _, kind := range []string{"read close cancellation", "source change"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-volume-id/v3/project/"))
			if err != nil {
				t.Fatal(err)
			}
			cinder, err := conn.BlockStorageV3(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause, readError, closeError := errors.New("member ID canceled"), errors.New("member ID read failed"), errors.New("member ID Close failed")
			body := []byte(`{"volume":{"id":"response-id"}}`)
			calls := 0
			cloud.Provider.HTTPClient.Transport = connectionReadRoundTripper(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != http.MethodGet || r.URL.Path != "/connection-volume-id/v3/project/volumes/volume" || r.URL.RawQuery != "" {
					t.Error(r.Method, r.URL)
				}
				stream := &connectionReadBody{content: bytes.Clone(body), readError: io.EOF, close: func() {}}
				if kind == "read close cancellation" {
					stream.readError, stream.closeError = readError, closeError
					stream.close = func() { cancel(cause) }
				} else {
					stream.close = func() { cinder.RawClient().Endpoint = cloud.Server.URL + "/changed/v3/project/" }
				}
				return &http.Response{StatusCode: 200, Status: "200 OK", Header: http.Header{"Content-Type": {"application/json"}, "X-Evidence": {"actual-member"}}, Body: stream, Request: r}, nil
			})
			callbacks := 0
			result, err := conn.GetVolumeID(ctx, blockstorage.GetVolumeIDRequest{NameOrID: "volume"}, func(*blockstorage.VolumeReadOpts) error { callbacks++; return nil })
			if result == nil || err == nil || result.ID != nil || result.Value != nil || result.Volume != nil || result.Observed == nil || len(result.Pages) != 0 || callbacks != 1 || calls != 1 {
				t.Fatal(result, err, callbacks, calls)
			}
			connectionReadCheckOperation(t, "GetVolumeID", err)
			if result.Observed.StatusCode != 200 || result.Observed.Header.Get("X-Evidence") != "actual-member" || !bytes.Equal(result.Observed.Body, body) {
				t.Fatal(result.Observed)
			}
			if kind == "read close cancellation" {
				for _, sentinel := range []error{readError, closeError, context.Canceled, cause} {
					if !errors.Is(err, sentinel) {
						t.Fatal("lost accepted cause", sentinel, err)
					}
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}

func TestConnectionGetVolumeIDAbsenceUnsafeNamesAndLookupFailuresRemainDistinct(t *testing.T) {
	for _, kind := range []string{"absent", "unsafe literal name", "ambiguous", "list forbidden", "invalid caller location"} {
		t.Run(kind, func(t *testing.T) {
			cloud := testcloud.New(t)
			conn, err := sdk.FromProvider(cloud.Provider, sdk.WithEndpoint(sdk.BlockStorage, cloud.Server.URL+"/connection-volume-id/v3/project/"))
			if err != nil {
				t.Fatal(err)
			}
			identity := "volume"
			if kind == "unsafe literal name" {
				identity = "folder/volume"
			}
			var gets, lists atomic.Int32
			cloud.Mux.HandleFunc("GET /connection-volume-id/v3/project/volumes/volume", func(w http.ResponseWriter, r *http.Request) {
				gets.Add(1)
				testcloud.JSON(w, 404, `{"itemNotFound":{"message":"member missing"}}`)
			})
			cloud.Mux.HandleFunc("GET /connection-volume-id/v3/project/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
				lists.Add(1)
				if r.URL.Query().Get("name") != identity {
					t.Error(r.URL)
				}
				w.Header().Set("X-Evidence", "actual-list")
				switch kind {
				case "unsafe literal name":
					testcloud.JSON(w, 200, `{"volumes":[{"id":"","name":"folder/volume"}]}`)
				case "ambiguous":
					testcloud.JSON(w, 200, `{"volumes":[{"id":"a","name":"volume"},{"id":"b","name":"volume"}]}`)
				case "list forbidden":
					testcloud.JSON(w, 403, `{"forbidden":{"message":"list denied"}}`)
				case "invalid caller location":
					testcloud.JSON(w, 200, `{"volumes":[{"id":"response-id","name":"volume"}]}`)
				default:
					testcloud.JSON(w, 200, `{"volumes":[]}`)
				}
			})
			options := []blockstorage.VolumeReadOption{}
			if kind == "invalid caller location" {
				options = append(options, blockstorage.WithVolumeReadLocation(resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`not-json`)}}))
			}
			result, err := conn.GetVolumeID(context.Background(), blockstorage.GetVolumeIDRequest{NameOrID: identity}, options...)
			wantGets := int32(1)
			if kind == "unsafe literal name" {
				wantGets = 0
			}
			if result == nil || result.Observed != nil || gets.Load() != wantGets || lists.Load() != 1 {
				t.Fatal(result, err, gets.Load(), lists.Load())
			}
			switch kind {
			case "absent":
				if err != nil || result.ID != nil || result.Value != nil || result.Volume != nil || len(result.Pages) != 1 {
					t.Fatal(result, err)
				}
			case "unsafe literal name":
				if err != nil || string(result.ID) != `""` || result.Value == nil || result.Volume == nil || len(result.Pages) != 1 {
					t.Fatal(result, err)
				}
			default:
				if err == nil || result.ID != nil || result.Value != nil || result.Volume != nil {
					t.Fatal(result, err)
				}
				connectionReadCheckOperation(t, "GetVolumeID", err)
				if kind == "ambiguous" {
					if !errors.Is(err, resource.ErrAmbiguous) || len(result.Pages) != 1 {
						t.Fatal(result, err)
					}
				} else if kind == "invalid caller location" {
					var response *resource.ResponseError
					if !errors.Is(err, resource.ErrInvalidOption) || errors.As(err, &response) || len(result.Pages) != 1 {
						t.Fatal(result, err, response)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 403 || native.ResponseHeader.Get("X-Evidence") != "actual-list" || len(result.Pages) != 0 {
						t.Fatal(result, err, native)
					}
				}
			}
			if len(result.Pages) == 1 && (result.Pages[0].StatusCode != 200 || result.Pages[0].Header.Get("X-Evidence") != "actual-list") {
				t.Fatal(result.Pages)
			}
		})
	}
}
