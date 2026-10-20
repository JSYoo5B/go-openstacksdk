package resourceclasses_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/resourceclasses"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonRCTransport func(*http.Request) (*http.Response, error)

func (transport pythonRCTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonRCWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type pythonRCCall struct{ method, path, query, body, version string }

type pythonRCRecorder struct {
	mu    sync.Mutex
	calls []pythonRCCall
}

func (r *pythonRCRecorder) snapshot() []pythonRCCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]pythonRCCall(nil), r.calls...)
}

// ResourceClass declares _max_microversion 1.2, so the helper pins that version.
func pythonRCAPI(t *testing.T, recorder *pythonRCRecorder, reply func(*http.Request) *http.Response) *resourceclasses.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonRCTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		recorder.mu.Lock()
		recorder.calls = append(recorder.calls, pythonRCCall{req.Method, req.URL.Path, req.URL.RawQuery, raw, req.Header.Get("OpenStack-API-Version")})
		recorder.mu.Unlock()
		return reply(req), nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = "1.2"
	return resourceclasses.New(client)
}

const pythonRCBase = "/placement/resource_classes"

func TestPythonResourceClassProxyCalls(t *testing.T) {
	ctx := context.Background()
	t.Run("create get list", func(t *testing.T) {
		var recorder pythonRCRecorder
		api := pythonRCAPI(t, &recorder, func(req *http.Request) *http.Response {
			switch {
			case req.Method == http.MethodPost:
				return pythonRCWire(201, "")
			case req.URL.Path == pythonRCBase:
				return pythonRCWire(200, `{"resource_classes":[{"name":"VCPU","links":[]},{"name":"CUSTOM_X","links":[{"href":"/resource_classes/CUSTOM_X","rel":"self"}]}]}`)
			}
			return pythonRCWire(200, `{"name":"CUSTOM_X","links":[{"href":"/resource_classes/CUSTOM_X","rel":"self"}]}`)
		})
		// create_resource_class(name="CUSTOM_X") returns the attrs it sent; the 201 has no body.
		if err := api.Create(ctx, resourceclasses.CreateOpts{Name: "CUSTOM_X"}); err != nil {
			t.Fatal(err)
		}
		// get_resource_class("CUSTOM_X")
		got, err := api.Get(ctx, "CUSTOM_X")
		if err != nil || got.Name != "CUSTOM_X" {
			t.Fatal(got, err)
		}
		// resource_classes() yields every class.
		all, err := api.All(ctx)
		if err != nil || len(all) != 2 || all[1].Name != "CUSTOM_X" {
			t.Fatal(all, err)
		}
		// resource_classes(name="CUSTOM_X") filters the body attribute locally after the same request.
		named, err := api.All(ctx, resource.WithName("CUSTOM_X"))
		if err != nil || len(named) != 1 || named[0].Name != "CUSTOM_X" {
			t.Fatal(named, err)
		}
		want := []pythonRCCall{
			{http.MethodPost, pythonRCBase, "", `{"name":"CUSTOM_X"}`, "placement 1.2"},
			{http.MethodGet, pythonRCBase + "/CUSTOM_X", "", "", "placement 1.2"},
			{http.MethodGet, pythonRCBase, "", "", "placement 1.2"},
			{http.MethodGet, pythonRCBase, "", "", "placement 1.2"},
		}
		if got := recorder.snapshot(); !reflect.DeepEqual(got, want) {
			t.Fatalf("%+v", got)
		}
	})
	t.Run("delete_resource_class ignore_missing", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			code    int
			options []resource.LookupOption
			missing bool
		}{
			{"default ignores 404", 404, nil, false},
			{"ignore_missing=False", 404, []resource.LookupOption{resource.WithMissingError()}, true},
			{"deleted", 204, nil, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var recorder pythonRCRecorder
				api := pythonRCAPI(t, &recorder, func(*http.Request) *http.Response { return pythonRCWire(tc.code, "") })
				err := api.Remove(ctx, resource.ID("CUSTOM_X"), tc.options...)
				if tc.missing != errors.Is(err, resource.ErrNotFound) || (!tc.missing && err != nil) {
					t.Fatal(err)
				}
				if got := recorder.snapshot(); !reflect.DeepEqual(got, []pythonRCCall{{http.MethodDelete, pythonRCBase + "/CUSTOM_X", "", "", "placement 1.2"}}) {
					t.Fatalf("%+v", got)
				}
			})
		}
	})
	t.Run("get_resource_class missing", func(t *testing.T) {
		var recorder pythonRCRecorder
		api := pythonRCAPI(t, &recorder, func(*http.Request) *http.Response { return pythonRCWire(404, `{}`) })
		if _, err := api.Get(ctx, "CUSTOM_X"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			t.Fatal(err)
		}
	})
}

// At 1.2 update_resource_class sends PUT with a JSON body and gets a 200 resource class.
// Go Update sends no body and accepts only the 1.7+ statuses 201 and 204.
func TestPythonResourceClassUpdateIsNotTheRenameCall(t *testing.T) {
	var recorder pythonRCRecorder
	api := pythonRCAPI(t, &recorder, func(*http.Request) *http.Response { return pythonRCWire(200, `{"name":"CUSTOM_Y"}`) })
	err := api.Update(context.Background(), "CUSTOM_X")
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != 200 || !reflect.DeepEqual(native.Expected, []int{201, 204}) {
		t.Fatal(err)
	}
	if got := recorder.snapshot(); !reflect.DeepEqual(got, []pythonRCCall{{http.MethodPut, pythonRCBase + "/CUSTOM_X", "", "", "placement 1.2"}}) {
		t.Fatalf("%+v", got)
	}
}

func TestPythonResourceClassWaitForDelete(t *testing.T) {
	var recorder pythonRCRecorder
	var mu sync.Mutex
	codes := []int{200, 200, 404}
	api := pythonRCAPI(t, &recorder, func(*http.Request) *http.Response {
		mu.Lock()
		defer mu.Unlock()
		code := codes[0]
		if len(codes) > 1 {
			codes = codes[1:]
		}
		return pythonRCWire(code, `{"name":"CUSTOM_X"}`)
	})
	err := api.WaitForDeletion(context.Background(), resource.ID("CUSTOM_X"), resource.WithPollInterval(time.Millisecond), resource.WithTimeout(120*time.Second))
	if err != nil || len(recorder.snapshot()) != 3 {
		t.Fatal(err, recorder.snapshot())
	}
}
