package defaulttypes_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/defaulttypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type dtCall struct {
	method, path, query, body string
	header                    http.Header
}

const dtBase = "/cinder/v3/project/default-types"

func dtDiscovery(maximum string) string {
	return `{"version":{"id":"v3.0","status":"CURRENT","version":"` + maximum + `","min_version":"3.0"}}`
}

func dtAPI(t *testing.T, version, maximum string, reply func(*http.Request) (int, string)) (*defaulttypes.API, *testcloud.Recorder) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := testcloud.NewRecorder(func(r *http.Request) (*http.Response, error) {
		code, text := 200, dtDiscovery(maximum)
		if r.URL.Path != "/cinder/v3/" {
			code, text = reply(r)
		}
		return testcloud.Response(code, text, http.Header{"Content-Type": {"application/json"}}), nil
	})
	cloud.SetTransport(calls)
	client := cloud.Client("volumev3", "/cinder/v3/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = version
	return defaulttypes.New(client), calls
}

// dtCalls projects recorded calls onto the local call fields.
func dtCalls(rec *testcloud.Recorder) []dtCall {
	calls := []dtCall{}
	for _, call := range rec.Calls() {
		calls = append(calls, dtCall{call.Method, call.EscapedPath, call.RawQuery, call.Body, call.Header})
	}
	return calls
}

func dtOperation(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "default_types" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if code != 0 && (!errors.As(err, &native) || native.Actual != code) {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

func dtVersion(t *testing.T, call dtCall, want string) {
	t.Helper()
	if call.header.Get("OpenStack-API-Version") != "volume "+want || call.header.Get("X-OpenStack-Volume-API-Version") != want {
		t.Fatalf("want microversion %s, got %v", want, call.header)
	}
}

const dtRow = `{"project_id":"p1","volume_type_id":"vt1"}`

func TestDefaultTypeListRequiresMicroversion367(t *testing.T) {
	api, calls := dtAPI(t, "", "3.70", func(*http.Request) (int, string) {
		return 200, `{"default_types":[` + dtRow + `,{"project_id":"p2","volume_type_id":"vt2"}]}`
	})
	values, err := api.All(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if calls.Len() != 2 || dtCalls(calls)[0].path != "/cinder/v3/" || dtCalls(calls)[1].method != http.MethodGet || dtCalls(calls)[1].path != dtBase || dtCalls(calls)[1].query != "" {
		t.Fatal(dtCalls(calls))
	}
	dtVersion(t, dtCalls(calls)[1], "3.67")
	if len(values) != 2 || *values[0].ProjectID != "p1" || *values[0].VolumeTypeID != "vt1" || values[0].ID != "" || len(values[1].Discovery) != 1 {
		t.Fatalf("rows: %#v", values)
	}

	for _, item := range []struct{ version, maximum string }{{"", "3.66"}, {"3.62", "3.70"}} {
		api, calls = dtAPI(t, item.version, item.maximum, func(*http.Request) (int, string) {
			t.Fatal("request after unsupported microversion")
			return 0, ""
		})
		_, err = api.All(context.Background())
		if !errors.Is(err, resource.ErrUnsupported) || calls.Len() != 1 {
			t.Fatal(item, err, dtCalls(calls))
		}
		dtOperation(t, err, "List", 0)
	}

	api, calls = dtAPI(t, "3.70", "3.70", func(*http.Request) (int, string) { return 200, `{"default_types":[]}` })
	if _, err = api.All(context.Background()); err != nil || calls.Len() != 2 {
		t.Fatal(err, dtCalls(calls))
	}
	dtVersion(t, dtCalls(calls)[1], "3.70")
}

func TestDefaultTypeGetSetUnset(t *testing.T) {
	api, calls := dtAPI(t, "", "3.70", func(*http.Request) (int, string) { return 200, `{"default_type":` + dtRow + `}` })
	v, err := api.Get(context.Background(), "p1")
	if err != nil || dtCalls(calls)[1].method != http.MethodGet || dtCalls(calls)[1].path != dtBase+"/p1" {
		t.Fatal(err, dtCalls(calls))
	}
	dtVersion(t, dtCalls(calls)[1], "3.67")
	if v.ID != "p1" || *v.ProjectID != "p1" || *v.VolumeTypeID != "vt1" || v.StatusCode != 200 {
		t.Fatalf("get: %#v", v)
	}

	api, calls = dtAPI(t, "", "3.70", func(*http.Request) (int, string) { return 200, `{"default_type":` + dtRow + `}` })
	v, err = api.Set(context.Background(), "p1", "vt1")
	if err != nil || dtCalls(calls)[1].method != http.MethodPut || dtCalls(calls)[1].path != dtBase+"/p1" || dtCalls(calls)[1].body != `{"default_type":{"volume_type":"vt1"}}` {
		t.Fatal(err, dtCalls(calls))
	}
	if v.ID != "p1" || *v.VolumeTypeID != "vt1" || *v.ProjectID != "p1" {
		t.Fatalf("set: %#v", v)
	}
	api, calls = dtAPI(t, "3.67", "3.70", func(*http.Request) (int, string) { return 200, `` })
	v, err = api.Set(context.Background(), "p1", "")
	if err != nil || dtCalls(calls)[1].body != `{"default_type":{"volume_type_id":""}}` || v.ID != "p1" || *v.VolumeTypeID != "" {
		t.Fatal(err, dtCalls(calls), v)
	}
	api, _ = dtAPI(t, "", "3.70", func(*http.Request) (int, string) { return 404, `{}` })
	_, err = api.Set(context.Background(), "p1", "vt1")
	dtOperation(t, err, "Set", 404)

	for _, item := range []struct {
		code    int
		options []defaulttypes.DeleteOption
		fail    bool
	}{{204, nil, false}, {404, nil, false}, {404, []defaulttypes.DeleteOption{defaulttypes.WithDeleteIgnoreMissing(false)}, true}, {403, nil, true}} {
		api, calls = dtAPI(t, "", "3.70", func(*http.Request) (int, string) { return item.code, `` })
		err = api.Unset(context.Background(), "p1", item.options...)
		if dtCalls(calls)[1].method != http.MethodDelete || dtCalls(calls)[1].path != dtBase+"/p1" {
			t.Fatal(dtCalls(calls))
		}
		dtVersion(t, dtCalls(calls)[1], "3.67")
		if item.fail {
			dtOperation(t, err, "Unset", item.code)
		} else if err != nil {
			t.Fatal(err)
		}
	}
	if _, err := api.Get(context.Background(), ""); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}
