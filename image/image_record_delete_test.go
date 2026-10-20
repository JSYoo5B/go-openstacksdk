package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/serviceinfo"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	th "github.com/gophercloud/gophercloud/v2/testhelper"
)

func imageRecordDeleteStore(rawID string) *serviceinfo.StoreRecord {
	return &serviceinfo.StoreRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(rawID), "name": json.RawMessage(`"store name is passive"`)}}}}
}

func TestDeleteImageRecordLiteralWholeAndStoreRoutesOwnOpaqueAcknowledgements(t *testing.T) {
	for _, store := range []bool{false, true} {
		for _, id := range []string{"fixed", "a /한:%?\\b", "%2F/segment", " name-like identity ", "é", "e\u0301"} {
			t.Run(fmt.Sprintf("store=%v/id=%q", store, id), func(t *testing.T) {
				const storeID = "store /标签:%?\\x"
				raw := []byte{'o', 'p', 'a', 'q', 'u', 'e', 0xff}
				calls, locations := 0, 0
				cloud := "captured"
				body := &taskCoreBody{reader: bytes.NewReader(raw)}
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					path := "/reverse/glance/v2/images/" + url.PathEscape(id)
					if store {
						path = "/reverse/glance/v2/stores/" + url.PathEscape(storeID) + "/" + url.PathEscape(id)
					}
					if req.Method != http.MethodDelete || req.URL.EscapedPath() != path || req.URL.RawQuery != "" || req.Body != nil {
						t.Fatal(req.Method, req.URL, req.Body)
					}
					reply := taskCoreHTTP(req, 299, body)
					reply.Header.Set("OpenStack-image-import-methods", "a,, b")
					return reply, nil
				})
				service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
				options := []ImageRecordDeleteOption{}
				if store {
					options = append(options, WithImageRecordDeleteStoreID(storeID))
				}
				got, err := service.DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{ID: id}, options...)
				if got == nil || got.Acknowledgement == nil || err != nil || calls != 1 || locations != 1 || body.closes != 1 {
					t.Fatal(got, err, calls, locations, body.closes)
				}
				ack := got.Acknowledgement
				wantStore := ""
				if store {
					wantStore = storeID
				}
				if ack.ImageID != id || ack.StoreID != wantStore || ack.StatusCode != 299 || !bytes.Equal(ack.Body, raw) || ack.Header.Get("X-Task-Proof") != "actual" {
					t.Fatal(ack)
				}
				if store {
					if got.Record != nil {
						t.Fatal("store deletion invented image projection", got.Record)
					}
				} else {
					if got.Record == nil || got.Record.Wire != nil || got.Record.StatusCode != 0 || got.Record.Resource.StatusCode != 0 || len(got.Record.Header) != 0 || got.Record.Envelope != nil || len(got.Record.Resource.Body) != 65 {
						t.Fatal("literal deletion invented fetch receipt", got.Record)
					}
					encodedID, _ := json.Marshal(id)
					th.AssertEquals(t, string(encodedID), string(got.Record.Resource.Body["id"]))
					th.AssertEquals(t, `[]`, string(got.Record.Resource.Body["tags"]))
					th.CheckDeepEquals(t, []string{"a", "", " b"}, got.Record.ImportMethods)
					var location resource.CloudLocation
					if err := json.Unmarshal(got.Record.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "captured" {
						t.Fatal(location, err)
					}
				}
				ack.Body[0] = '!'
				if raw[0] != 'o' {
					t.Fatal("ack aliases source bytes", raw)
				}
			})
		}
	}
}

func TestDeleteImageRecordProjectsPrivateBodyAndPreservesFetchedReceipt(t *testing.T) {
	const fetched = `{"id":"fixed","protected":"false","size":"04","tags":"one","properties":{"exact":900719925474099312345}}`
	var handler taskCoreTransport
	calls, locations := 0, 0
	cloud := "before"
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) { locations++; return resource.CloudLocation{Cloud: &cloud}, nil }})
	seed := imageRecordUpdateFetched(t, service, fetched, &handler)
	seed.Resource.Body["id"] = json.RawMessage(`"public decoy"`)
	seed.Resource.Body["size"] = json.RawMessage(`"public invalid"`)
	seed.Wire.Body["id"] = json.RawMessage(`"wire decoy"`)
	created, updated := "passive created", "passive updated"
	seed.Resource.CreatedAt, seed.Resource.UpdatedAt = &created, &updated
	seed.Resource.Links = []resource.Link{{Href: "https://foreign.test/passive", Rel: "self"}}
	before := cloneImageRecord(seed)
	cloud = "during delete"
	handler = func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete || req.URL.EscapedPath() != "/reverse/glance/v2/images/fixed" || req.Body != nil {
			t.Fatal("public edits retargeted delete", req.Method, req.URL, req.Body)
		}
		response := taskCoreJSON(req, 201, `{"id":"response decoy","size":null}`)
		response.Header.Set("OpenStack-image-import-methods", "new, raw")
		return response, nil
	}
	got, err := service.DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{Record: seed})
	if got == nil || got.Record == nil || got.Acknowledgement == nil || err != nil || calls != 2 || locations != 2 || got.Record == seed || got.Record.Resource == seed.Resource || got.Record.Wire == seed.Wire {
		t.Fatal(got, err, calls, locations)
	}
	value := got.Record
	th.AssertEquals(t, `"fixed"`, string(value.Resource.Body["id"]))
	th.AssertEquals(t, `4`, string(value.Resource.Body["size"]))
	th.AssertEquals(t, `true`, string(value.Resource.Body["is_protected"]))
	th.AssertEquals(t, `["one"]`, string(value.Resource.Body["tags"]))
	if value.StatusCode != 203 || value.Resource.StatusCode != 203 || value.Header.Get("X-Task-Proof") != seed.Header.Get("X-Task-Proof") || !reflect.DeepEqual(value.Wire, seed.Wire) || string(value.Envelope) != fetched || !reflect.DeepEqual(value.bodyState, seed.bodyState) || !reflect.DeepEqual(before, seed) {
		t.Fatal("deletion overwrote receipt, Body or input", value, seed)
	}
	th.CheckDeepEquals(t, []string{"new", " raw"}, value.ImportMethods)
	if value.Resource.CreatedAt == seed.Resource.CreatedAt || value.Resource.UpdatedAt == seed.Resource.UpdatedAt || value.Resource.CreatedAt == nil || *value.Resource.CreatedAt != "passive created" || value.Resource.UpdatedAt == nil || *value.Resource.UpdatedAt != "passive updated" || !reflect.DeepEqual(value.Resource.Links, seed.Resource.Links) {
		t.Fatal("passive metadata was lost or aliased", value.Resource, seed.Resource)
	}
	var location resource.CloudLocation
	if err := json.Unmarshal(value.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "during delete" {
		t.Fatal(location, err)
	}
	value.bodyState.current["size"][1] = 'X'
	value.bodyState.original["size"][1] = 'X'
	value.Wire.Body["id"][1] = 'X'
	value.Header.Set("X-Task-Proof", "changed")
	value.Envelope[0] = '!'
	value.ImportMethods[0] = "changed"
	*value.Resource.CreatedAt = "changed"
	*value.Resource.UpdatedAt = "changed"
	value.Resource.Links[0].Href = "changed"
	if !reflect.DeepEqual(before, seed) {
		t.Fatal("returned deletion record aliases input", seed)
	}
}

func TestDeleteImageRecordKeepsPendingChangesForExplicitSubsequentCommit(t *testing.T) {
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			if req.Method != http.MethodPatch {
				t.Fatal(req.Method)
			}
			imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/name","value":"pending"}]`)
			return taskCoreJSON(req, 299, `pending opaque response`), nil
		case 2:
			if req.Method != http.MethodDelete || req.Body != nil {
				t.Fatal(req.Method, req.Body)
			}
			return taskCoreJSON(req, 203, `{"name":"must not overlay"}`), nil
		case 3:
			if req.Method != http.MethodPatch {
				t.Fatal("DELETE cleaned pending raw state", req.Method)
			}
			imageRecordUpdateCheck(t, imageRecordUpdatePayload(t, req), `[{"op":"add","path":"/name","value":"pending"}]`)
			return taskCoreJSON(req, 200, `{}`), nil
		default:
			t.Fatal("unexpected HTTP", calls)
			return nil, nil
		}
	})
	service := New(client)
	pending, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{ID: "fixed", Attributes: map[string]any{"name": "pending"}})
	if pending == nil || err != nil {
		t.Fatal(pending, err)
	}
	got, err := service.DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{Record: pending})
	if got == nil || got.Record == nil || err != nil || got.Record.StatusCode != 299 || got.Record.Wire != nil || string(got.Record.Envelope) != `pending opaque response` || !reflect.DeepEqual(got.Record.bodyState, pending.bodyState) || string(got.Record.Resource.Body["name"]) != `"pending"` {
		t.Fatal(got, err)
	}
	committed, err := service.UpdateImageRecord(context.Background(), ImageRecordUpdateRequest{Record: got.Record})
	if committed == nil || err != nil || calls != 3 {
		t.Fatal(committed, err, calls)
	}
}

func TestDeleteImageRecordStoreOnlyExtractsIdentitiesBeforeProjection(t *testing.T) {
	var handler taskCoreTransport
	calls := 0
	client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
	service := New(client)
	seed := imageRecordUpdateFetched(t, service, `{"id":"fixed","name":"seed"}`, &handler)
	// This branch consumes only the private identity. An unrelated raw descriptor
	// failure would surface if it incorrectly ran whole Image projection.
	seed.bodyState.current["hw_vif_multiqueue_enabled"] = json.RawMessage(`" invalid BoolStr "`)
	seed.bodyState.current["instance_type_rxtx_factor"] = json.RawMessage(`"not float"`)
	before := cloneImageRecord(seed)
	store := imageRecordDeleteStore(`"store /한:%?\\x"`)
	store.Resource.Body["properties"] = json.RawMessage(`{"broken":`)
	store.Resource.Body["description"] = json.RawMessage([]byte{0xff})
	handler = func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodDelete || req.URL.EscapedPath() != "/reverse/glance/v2/stores/"+url.PathEscape(`store /한:%?\x`)+"/fixed" || req.Body != nil {
			t.Fatal(req.Method, req.URL, req.Body)
		}
		return taskCoreJSON(req, 204, `opaque`), nil
	}
	got, err := service.DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{Record: seed}, WithImageRecordDeleteStoreRecord(store))
	if got == nil || got.Record != nil || got.Acknowledgement == nil || err != nil || calls != 2 || !reflect.DeepEqual(before, seed) {
		t.Fatal(got, err, calls)
	}
	got, err = service.DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{Record: seed})
	if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 2 {
		t.Fatal("whole deletion skipped required descriptor projection", got, err, calls)
	}
}

func TestDeleteImageRecordStoreSelectorsAndFactorySnapshots(t *testing.T) {
	for _, mode := range []string{"factory StoreID", "factory StoreRecord", "StoreID clears record", "StoreRecord clears ID", "whole config replacement", "nil record clears store"} {
		t.Run(mode, func(t *testing.T) {
			selected := "store factory"
			store := imageRecordDeleteStore(`"store record"`)
			var options []ImageRecordDeleteOption
			wantStore := ""
			switch mode {
			case "factory StoreID":
				options = []ImageRecordDeleteOption{WithImageRecordDeleteOpts(ImageRecordDeleteOpts{StoreID: &selected})}
				wantStore = selected
				selected = "caller changed"
			case "factory StoreRecord":
				options = []ImageRecordDeleteOption{WithImageRecordDeleteStoreRecord(store)}
				wantStore = "store record"
				store.Resource.Body["id"] = json.RawMessage(`"caller changed"`)
			case "StoreID clears record":
				options = []ImageRecordDeleteOption{WithImageRecordDeleteStoreRecord(store), WithImageRecordDeleteStoreID("store ID")}
				wantStore = "store ID"
			case "StoreRecord clears ID":
				options = []ImageRecordDeleteOption{WithImageRecordDeleteStoreID("store ID"), WithImageRecordDeleteStoreRecord(store)}
				wantStore = "store record"
			case "whole config replacement":
				options = []ImageRecordDeleteOption{WithImageRecordDeleteStoreID("discarded"), WithImageRecordDeleteOpts(ImageRecordDeleteOpts{})}
			case "nil record clears store":
				options = []ImageRecordDeleteOption{WithImageRecordDeleteStoreID("discarded"), WithImageRecordDeleteStoreRecord(nil)}
			}
			calls := 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
				calls++
				path := "/reverse/glance/v2/images/fixed"
				if wantStore != "" {
					path = "/reverse/glance/v2/stores/" + url.PathEscape(wantStore) + "/fixed"
				}
				if req.URL.EscapedPath() != path {
					t.Fatal(req.URL)
				}
				return taskCoreJSON(req, 204, ""), nil
			})
			got, err := New(client).DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{ID: "fixed"}, options...)
			if got == nil || got.Acknowledgement == nil || err != nil || got.Acknowledgement.StoreID != wantStore || (got.Record != nil) != (wantStore == "") || calls != 1 {
				t.Fatal(got, err, calls)
			}
		})
	}
}

func TestDeleteImageRecordSourceOptionsLocationAndChildSnapshots(t *testing.T) {
	for _, storeMode := range []bool{false, true} {
		t.Run(fmt.Sprintf("store=%v", storeMode), func(t *testing.T) {
			var handler taskCoreTransport
			calls, callbacks, locations := 0, 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return handler(req) })
			seed := imageRecordUpdateFetched(t, New(client), `{"id":"fixed","name":"seed"}`, &handler)
			store := imageRecordDeleteStore(`"store fixed"`)
			headers := map[string]string{"X-Option": "factory"}
			ignore := false
			factory := WithImageRecordDeleteOpts(ImageRecordDeleteOpts{Headers: headers, IgnoreMissing: &ignore})
			headers["X-Option"], ignore = "caller changed", true
			cloud := "captured"
			facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"token project"`)}}
			var retained *ImageRecordDeleteOpts
			var options []ImageRecordDeleteOption
			handler = func(req *http.Request) (*http.Response, error) {
				path := "/reverse/glance/v2/images/fixed"
				if storeMode {
					path = "/reverse/glance/v2/stores/store%20fixed/fixed"
				}
				if req.URL.EscapedPath() != path || req.Header.Get("X-Source") != "captured" || req.Header.Get("X-Option") != "factory" || req.Header.Get("X-Final") != "yes" || req.Header.Get("X-Auth-Token") != "live" || req.Header.Get("OpenStack-API-Version") != "image 2.10" {
					t.Fatal(req.URL, req.Header)
				}
				retained.Headers["X-Option"] = "retained mutation"
				if retained.StoreRecord != nil {
					retained.StoreRecord.Resource.Body["id"] = json.RawMessage(`"retained changed"`)
				}
				return taskCoreJSON(req, 204, ""), nil
			}
			client.MoreHeaders, client.Microversion = map[string]string{"X-Source": "captured"}, "2.10"
			service := NewWithDependencies(client, Dependencies{CloudLocation: func() (resource.CloudLocation, error) {
				locations++
				seed.bodyState.current["id"] = json.RawMessage(`"location changed ID"`)
				client.MoreHeaders["X-Source"] = "changed by location"
				options[0] = WithImageRecordDeleteHeader("X-Option", "location changed option")
				return facts, nil
			}})
			options = []ImageRecordDeleteOption{factory}
			if storeMode {
				options = append(options, WithImageRecordDeleteStoreRecord(store))
			}
			options = append(options, func(config *ImageRecordDeleteOpts) error {
				callbacks++
				retained = config
				cloud = "after option"
				facts.Project.ID[1] = 'X'
				client.SetToken("live")
				return WithImageRecordDeleteHeader("X-Final", "yes")(config)
			})
			got, err := service.DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{Record: seed}, options...)
			if got == nil || err != nil || calls != 2 || callbacks != 1 || locations != 1 {
				t.Fatal(got, err, calls, callbacks, locations)
			}
			if !storeMode {
				var location resource.CloudLocation
				if err := json.Unmarshal(got.Record.Resource.Body["location"], &location); err != nil || location.Cloud == nil || *location.Cloud != "captured" || string(location.Project.ID) != `"token project"` {
					t.Fatal(location, err)
				}
			}
		})
	}
}

func TestDeleteImageRecordAcceptedStatusesKeepOpaqueBodyWithoutTranslation(t *testing.T) {
	for _, store := range []bool{false, true} {
		for _, test := range []struct {
			code int
			raw  string
		}{{200, `null`}, {201, `[]`}, {204, ""}, {299, `{"id":"response decoy"}`}, {300, "not JSON"}, {304, `false`}, {399, "opaque\xff"}} {
			t.Run(fmt.Sprintf("store=%v/%d", store, test.code), func(t *testing.T) {
				calls, retries := 0, 0
				body := &taskCoreBody{reader: strings.NewReader(test.raw)}
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					return taskCoreHTTP(req, test.code, body), nil
				})
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries++
					return err
				}
				options := []ImageRecordDeleteOption{}
				if store {
					options = append(options, WithImageRecordDeleteStoreID("store"))
				}
				got, err := New(client).DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{ID: "fixed"}, options...)
				if got == nil || got.Acknowledgement == nil || err != nil || got.Acknowledgement.StatusCode != test.code || string(got.Acknowledgement.Body) != test.raw || calls != 1 || retries != 0 || body.closes != 1 || (got.Record != nil) != !store {
					t.Fatal(got, err, calls, retries, body.closes)
				}
			})
		}
	}
}

func TestDeleteImageRecordAcceptedPhysicalFailuresReturnOnlyAcknowledgement(t *testing.T) {
	for _, store := range []bool{false, true} {
		for _, mode := range []string{"read", "close", "cancel", "source drift", "read drift restored on Close", "outer drift restored on Close"} {
			t.Run(fmt.Sprintf("store=%v/%s", store, mode), func(t *testing.T) {
				const raw = `already accepted`
				marker := errors.New("accepted deletion failure")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				outerInvalid := false
				ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
					if outerInvalid {
						return marker
					}
					return nil
				})
				calls, retries := 0, 0
				var client *gophercloud.ServiceClient
				body := &taskCoreBody{reader: strings.NewReader(raw)}
				action := func() {}
				switch mode {
				case "read":
					body.reader = &taskCoreReader{body: raw, err: marker}
				case "close":
					body.closeErr = errors.Join(marker, gophercloud.ErrUnexpectedResponseCode{Actual: 404})
				case "cancel":
					action = func() { cancel(marker) }
				case "source drift", "read drift restored on Close":
					action = func() { client.Endpoint = "https://foreign.test/" }
				case "outer drift restored on Close":
					action = func() { outerInvalid = true }
				}
				if mode != "read" {
					body.reader = &taskCoreReader{body: raw, err: io.EOF, action: action}
				}
				selected := io.ReadCloser(body)
				if strings.Contains(mode, "restored") {
					selected = &imageRecordCloseBody{taskCoreBody: body, after: func() { client.Endpoint = "https://glance.example/reverse/glance/v2/"; outerInvalid = false }}
				}
				client = taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return taskCoreHTTP(req, 201, selected), nil })
				client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, err error, _ uint) error {
					retries++
					return err
				}
				options := []ImageRecordDeleteOption{}
				if store {
					options = append(options, WithImageRecordDeleteStoreID("store"))
				}
				got, err := New(client).DeleteImageRecord(ctx, ImageRecordDeleteRequest{ID: "fixed"}, options...)
				if got == nil || got.Record != nil || got.Acknowledgement == nil || got.Acknowledgement.StatusCode != 201 || string(got.Acknowledgement.Body) != raw || err == nil || calls != 1 || retries != 0 || body.closes != 1 {
					t.Fatal(got, err, calls, retries, body.closes)
				}
				if strings.Contains(mode, "source") || mode == "read drift restored on Close" {
					if !errors.Is(err, resource.ErrInvalidOption) {
						t.Fatal(err)
					}
				} else if !errors.Is(err, marker) {
					t.Fatal("cause lost", err)
				}
				if mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if errors.Is(err, resource.ErrNotFound) {
					t.Fatal("accepted close404 became absence", err)
				}
				taskCoreProof(t, err, 201, raw)
			})
		}
	}
}

func TestDeleteImageRecordMissingPolicySuppressesOnlyCleanFinalNative404(t *testing.T) {
	for _, store := range []bool{false, true} {
		for _, mode := range []string{"default", "explicit true", "strict", "retry clean final", "retry hook cause", "read", "close", "cancel", "source", "nested transport404"} {
			t.Run(fmt.Sprintf("store=%v/%s", store, mode), func(t *testing.T) {
				marker := errors.New("delete missing handling")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				calls, retries := 0, 0
				var client *gophercloud.ServiceClient
				body := &taskCoreBody{reader: strings.NewReader(`missing response`)}
				switch mode {
				case "read":
					body.reader = &taskCoreReader{body: `missing response`, err: marker}
				case "close":
					body.closeErr = marker
				case "cancel":
					body.reader = &taskCoreReader{body: `missing response`, err: io.EOF, action: func() { cancel(marker) }}
				case "source":
					body.reader = &taskCoreReader{body: `missing response`, err: io.EOF, action: func() { client.Endpoint = "https://foreign.test/" }}
				}
				client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if mode == "nested transport404" {
						return nil, errors.Join(marker, gophercloud.ErrUnexpectedResponseCode{Method: req.Method, URL: "https://decoy.test/", Actual: 404})
					}
					if mode == "retry clean final" && calls == 1 {
						return taskCoreJSON(req, 503, `retry`), nil
					}
					return taskCoreHTTP(req, 404, body), nil
				})
				if strings.HasPrefix(mode, "retry") {
					client.RetryFunc = func(_ context.Context, _, _ string, _ *gophercloud.RequestOpts, original error, count uint) error {
						retries++
						if mode == "retry hook cause" {
							return errors.Join(original, marker)
						}
						if count == 1 {
							return nil
						}
						return original
					}
				}
				options := []ImageRecordDeleteOption{}
				if store {
					options = append(options, WithImageRecordDeleteStoreID("store"))
				}
				if mode == "strict" {
					options = append(options, WithImageRecordDeleteIgnoreMissing(false))
				}
				if mode == "explicit true" {
					options = append(options, WithImageRecordDeleteIgnoreMissing(true))
				}
				got, err := New(client).DeleteImageRecord(ctx, ImageRecordDeleteRequest{ID: "fixed"}, options...)
				clean := mode == "default" || mode == "explicit true" || mode == "retry clean final"
				if got != nil || clean && err != nil || !clean && err == nil {
					t.Fatal(got, err, calls, retries)
				}
				wantCalls := 1
				if mode == "retry clean final" {
					wantCalls = 2
					if retries != 2 {
						t.Fatal("404 bypassed native hook", retries)
					}
				}
				if calls != wantCalls || mode != "nested transport404" && body.closes != 1 {
					t.Fatal(calls, body.closes)
				}
				if !clean && mode != "strict" && mode != "source" && !errors.Is(err, marker) {
					t.Fatal("handling cause lost", err)
				}
				if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if mode == "cancel" && !errors.Is(err, context.Canceled) {
					t.Fatal(err)
				}
				if mode == "strict" {
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != 404 || string(native.Body) != `missing response` {
						t.Fatal(native, err)
					}
				}
			})
		}
	}
}

func TestDeleteImageRecordNativeRejectionsNeverCreateAcknowledgement(t *testing.T) {
	for _, store := range []bool{false, true} {
		for _, code := range []int{400, 403, 404, 409, 500, 599} {
			t.Run(fmt.Sprintf("store=%v/%d", store, code), func(t *testing.T) {
				calls := 0
				client := taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					return taskCoreJSON(req, code, `actual rejection`), nil
				})
				options := []ImageRecordDeleteOption{WithImageRecordDeleteIgnoreMissing(false)}
				if store {
					options = append(options, WithImageRecordDeleteStoreID("store"))
				}
				got, err := New(client).DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{ID: "fixed"}, options...)
				var native gophercloud.ErrUnexpectedResponseCode
				var accepted *resource.ResponseError
				if got != nil || err == nil || !errors.As(err, &native) || native.Actual != code || string(native.Body) != `actual rejection` || native.ResponseHeader.Get("X-Task-Proof") != "actual" || errors.As(err, &accepted) || calls != 1 {
					t.Fatal(got, err, native, calls)
				}
			})
		}
	}
}

func TestDeleteImageRecordNativeRetryKeepsFixedScopeAndActualStatusPolicy(t *testing.T) {
	for _, store := range []bool{false, true} {
		for _, mode := range []string{"successful retry", "expanded OkCodes rejection", "changed body ownership"} {
			t.Run(fmt.Sprintf("store=%v/%s", store, mode), func(t *testing.T) {
				calls, retries := 0, 0
				var client *gophercloud.ServiceClient
				client = taskCoreClient(func(req *http.Request) (*http.Response, error) {
					calls++
					if req.Method != http.MethodDelete || req.Body != nil {
						t.Fatal(req.Method, req.Body)
					}
					if calls == 1 {
						return taskCoreJSON(req, 503, `initial rejection`), nil
					}
					if calls != 2 {
						t.Fatal("unexpected replay", calls)
					}
					if req.Header.Get("X-Auth-Token") != "retry token" || req.Header.Get("X-Retry") != "ordinary" {
						t.Fatal(req.Header)
					}
					if mode == "expanded OkCodes rejection" {
						return taskCoreJSON(req, 418, `actual rejected status`), nil
					}
					return taskCoreJSON(req, 203, `actual accepted status`), nil
				})
				client.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, original error, count uint) error {
					retries++
					if count > 1 {
						return original
					}
					client.SetToken("retry token")
					options.MoreHeaders = map[string]string{"X-Retry": "ordinary"}
					if mode == "expanded OkCodes rejection" {
						options.OkCodes = append(options.OkCodes, 418)
					}
					if mode == "changed body ownership" {
						options.JSONBody = map[string]bool{"must_not_send": true}
					}
					return nil
				}
				originalHook := reflect.ValueOf(client.RetryFunc).Pointer()
				options := []ImageRecordDeleteOption{WithImageRecordDeleteIgnoreMissing(false)}
				if store {
					options = append(options, WithImageRecordDeleteStoreID("store"))
				}
				got, err := New(client).DeleteImageRecord(context.Background(), ImageRecordDeleteRequest{ID: "fixed"}, options...)
				if mode == "successful retry" {
					if got == nil || got.Acknowledgement == nil || err != nil || got.Acknowledgement.StatusCode != 203 || string(got.Acknowledgement.Body) != `actual accepted status` || calls != 2 {
						t.Fatal(got, err, calls)
					}
				} else {
					var native gophercloud.ErrUnexpectedResponseCode
					wantCode, wantCalls, wantBody := 418, 2, `actual rejected status`
					if mode == "changed body ownership" {
						wantCode, wantCalls, wantBody = 503, 1, `initial rejection`
						if !errors.Is(err, resource.ErrInvalidOption) {
							t.Fatal(err)
						}
					}
					if got != nil || err == nil || !errors.As(err, &native) || native.Actual != wantCode || string(native.Body) != wantBody || calls != wantCalls {
						t.Fatal(got, err, native, calls)
					}
				}
				if retries != 1 || reflect.ValueOf(client.RetryFunc).Pointer() != originalHook {
					t.Fatal(retries)
				}
			})
		}
	}
}

func TestDeleteImageRecordCompletePreflightRejectsAmbiguousOrUnsafeSelectors(t *testing.T) {
	id, empty := "store", ""
	cases := []struct {
		name    string
		request ImageRecordDeleteRequest
		options []ImageRecordDeleteOption
	}{
		{"missing image", ImageRecordDeleteRequest{}, nil}, {"both image forms", ImageRecordDeleteRequest{ID: "fixed", Record: &ImageRecord{}}, nil},
		{"empty Record", ImageRecordDeleteRequest{Record: &ImageRecord{}}, nil},
		{"private null identity", ImageRecordDeleteRequest{Record: &ImageRecord{bodyState: newImageRecordBodyState(map[string]json.RawMessage{"id": json.RawMessage(`null`)})}}, nil},
		{"private untyped identity", ImageRecordDeleteRequest{Record: &ImageRecord{bodyState: newImageRecordBodyState(map[string]json.RawMessage{"id": json.RawMessage(`false`)})}}, nil},
		{"handcrafted Record no private state", ImageRecordDeleteRequest{Record: &ImageRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"id": json.RawMessage(`"fixed"`)}}}}}, nil},
		{"both store selectors", ImageRecordDeleteRequest{ID: "fixed"}, []ImageRecordDeleteOption{WithImageRecordDeleteOpts(ImageRecordDeleteOpts{StoreID: &id, StoreRecord: imageRecordDeleteStore(`"other"`)})}},
		{"explicit empty store", ImageRecordDeleteRequest{ID: "fixed"}, []ImageRecordDeleteOption{WithImageRecordDeleteStoreID("")}},
		{"empty store pointer", ImageRecordDeleteRequest{ID: "fixed"}, []ImageRecordDeleteOption{WithImageRecordDeleteOpts(ImageRecordDeleteOpts{StoreID: &empty})}},
		{"store missing Resource", ImageRecordDeleteRequest{ID: "fixed"}, []ImageRecordDeleteOption{WithImageRecordDeleteStoreRecord(&serviceinfo.StoreRecord{})}},
		{"store name is not fallback", ImageRecordDeleteRequest{ID: "fixed"}, []ImageRecordDeleteOption{WithImageRecordDeleteStoreRecord(&serviceinfo.StoreRecord{Resource: &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{"name": json.RawMessage(`"store name"`)}}}})}},
		{"store null id", ImageRecordDeleteRequest{ID: "fixed"}, []ImageRecordDeleteOption{WithImageRecordDeleteStoreRecord(imageRecordDeleteStore(`null`))}},
		{"store untyped id", ImageRecordDeleteRequest{ID: "fixed"}, []ImageRecordDeleteOption{WithImageRecordDeleteStoreRecord(imageRecordDeleteStore(`false`))}},
		{"nil option", ImageRecordDeleteRequest{ID: "fixed"}, []ImageRecordDeleteOption{nil}},
		{"owned auth header", ImageRecordDeleteRequest{ID: "fixed"}, []ImageRecordDeleteOption{WithImageRecordDeleteHeader("X-Auth-Token", "foreign")}},
		{"invalid header", ImageRecordDeleteRequest{ID: "fixed"}, []ImageRecordDeleteOption{WithImageRecordDeleteHeaders(map[string]string{"X-Extra": "\n"})}},
	}
	for _, value := range []string{" ", ".", "..", "bad\n", string([]byte{0xff})} {
		cases = append(cases, struct {
			name    string
			request ImageRecordDeleteRequest
			options []ImageRecordDeleteOption
		}{fmt.Sprintf("image %q", value), ImageRecordDeleteRequest{ID: value}, nil})
		cases = append(cases, struct {
			name    string
			request ImageRecordDeleteRequest
			options []ImageRecordDeleteOption
		}{fmt.Sprintf("store %q", value), ImageRecordDeleteRequest{ID: "fixed"}, []ImageRecordDeleteOption{WithImageRecordDeleteStoreID(value)}})
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil }))
			got, err := service.DeleteImageRecord(context.Background(), test.request, test.options...)
			if got != nil || !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
				t.Fatal(got, err, calls)
			}
		})
	}
	for _, mode := range []string{"nil context", "canceled", "nil service"} {
		t.Run(mode, func(t *testing.T) {
			calls, callbacks := 0, 0
			ctx := context.Background()
			service := New(taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil }))
			if mode == "nil context" {
				ctx = nil
			}
			if mode == "canceled" {
				canceled, cancel := context.WithCancel(context.Background())
				cancel()
				ctx = canceled
			}
			if mode == "nil service" {
				service = nil
			}
			got, err := service.DeleteImageRecord(ctx, ImageRecordDeleteRequest{ID: "fixed"}, func(*ImageRecordDeleteOpts) error { callbacks++; return nil })
			if got != nil || err == nil || calls != 0 || callbacks != 0 {
				t.Fatal(got, err, calls, callbacks)
			}
		})
	}
}

func TestDeleteImageRecordStopsAfterGuardFailureInOptions(t *testing.T) {
	for _, mode := range []string{"cancel", "source", "binding", "outer"} {
		t.Run(mode, func(t *testing.T) {
			marker := errors.New("image delete option guard")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			outerInvalid := false
			ctx = rest.WithOperationGuard(ctx, func(context.Context) error {
				if outerInvalid {
					return marker
				}
				return nil
			})
			calls, callbacks, later := 0, 0, 0
			client := taskCoreClient(func(req *http.Request) (*http.Response, error) { calls++; return nil, nil })
			service := New(client)
			got, err := service.DeleteImageRecord(ctx, ImageRecordDeleteRequest{ID: "fixed"}, func(*ImageRecordDeleteOpts) error {
				callbacks++
				switch mode {
				case "cancel":
					cancel(marker)
				case "source":
					client.Endpoint = "https://foreign.test/"
				case "binding":
					service.API = nil
				case "outer":
					outerInvalid = true
				}
				return nil
			}, func(*ImageRecordDeleteOpts) error { later++; return nil })
			if got != nil || err == nil || calls != 0 || callbacks != 1 || later != 0 {
				t.Fatal(got, err, calls, callbacks, later)
			}
			if mode == "cancel" || mode == "outer" {
				if !errors.Is(err, marker) {
					t.Fatal("cause lost", err)
				}
			} else if !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal(err)
			}
		})
	}
}
