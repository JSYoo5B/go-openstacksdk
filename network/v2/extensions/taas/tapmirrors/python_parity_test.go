package tapmirrors_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pythonTapMirrorCall struct {
	method, path string
	query        url.Values
	body         string
}

func pythonTapMirrorWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}})
}

func pythonTapMirrorAPI(t *testing.T, calls *[]pythonTapMirrorCall, reply func(*http.Request) *http.Response) (*tapmirrors.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonTapMirrorCall{req.Method, req.URL.Path, req.URL.Query(), raw})
		return reply(req), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return tapmirrors.New(client), cloud
}

func pythonTapMirrorJSON(t *testing.T, raw string) any {
	t.Helper()
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatal(raw, err)
	}
	return value
}

const (
	pythonTapMirrorBase = "/neutron/v2.0/taas/tap_mirrors"
	pythonTapMirrorRow  = `{"id":"tm-1","name":"mirror","description":"d","project_id":"proj","tenant_id":"proj","port_id":"p1","mirror_type":"gre","remote_ip":"192.0.2.9","directions":{"IN":"101","OUT":"102"}}`
)

// create_tap_mirror(name=..., description=..., port_id=..., mirror_type=...,
// remote_ip=..., directions=..., project_id=..., x_extra=...)
func TestPythonTapMirrorCreateBody(t *testing.T) {
	var calls []pythonTapMirrorCall
	api, _ := pythonTapMirrorAPI(t, &calls, func(*http.Request) *http.Response {
		return pythonTapMirrorWire(201, `{"tap_mirror":`+pythonTapMirrorRow+`}`)
	})
	created, err := api.Create(context.Background(), tapmirrors.CreateOpts{
		Name: "mirror", Description: "d", PortID: "p1", MirrorType: tapmirrors.MirrorTypeGre,
		RemoteIP: "192.0.2.9", Directions: tapmirrors.Directions{In: 101, Out: 102},
	}, tapmirrors.WithCreateField("project_id", "proj"), tapmirrors.WithCreateField("x_extra", true))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != pythonTapMirrorBase || len(calls[0].query) != 0 {
		t.Fatalf("%+v", calls)
	}
	want := pythonTapMirrorJSON(t, `{"tap_mirror":{"name":"mirror","description":"d","port_id":"p1","mirror_type":"gre","remote_ip":"192.0.2.9","directions":{"IN":"101","OUT":"102"},"project_id":"proj","x_extra":true}}`)
	if got := pythonTapMirrorJSON(t, calls[0].body); !reflect.DeepEqual(got, want) {
		t.Fatalf("body %s", calls[0].body)
	}
	if !(created.ID == "tm-1" && created.Name == "mirror" && created.Description == "d" && created.ProjectID == "proj" && created.PortID == "p1" &&
		created.MirrorType == "gre" && created.RemoteIP == "192.0.2.9" && created.Directions == (tapmirrors.Directions{In: 101, Out: 102})) {
		t.Fatalf("%+v", created)
	}

	// Python omits attributes the caller did not pass; Go always sends the
	// five non-omitempty core keys.
	calls = nil
	if _, err := api.Create(context.Background(), tapmirrors.CreateOpts{PortID: "p1", MirrorType: tapmirrors.MirrorTypeErspanv1, RemoteIP: "192.0.2.9", Directions: tapmirrors.Directions{In: 7}}); err != nil {
		t.Fatal(err)
	}
	want = pythonTapMirrorJSON(t, `{"tap_mirror":{"name":"","port_id":"p1","mirror_type":"erspanv1","remote_ip":"192.0.2.9","directions":{"IN":"7"}}}`)
	if got := pythonTapMirrorJSON(t, calls[0].body); !reflect.DeepEqual(got, want) {
		t.Fatalf("body %s", calls[0].body)
	}
	// A core key cannot be replaced by an extension field.
	if _, err := api.Create(context.Background(), tapmirrors.CreateOpts{}, tapmirrors.WithCreateField("directions", map[string]int{"IN": 1})); !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
		t.Fatal(err, len(calls))
	}
}

// get_tap_mirror(tap_mirror) with the default fields=None.
func TestPythonTapMirrorGetWithoutFields(t *testing.T) {
	var calls []pythonTapMirrorCall
	api, _ := pythonTapMirrorAPI(t, &calls, func(*http.Request) *http.Response {
		return pythonTapMirrorWire(200, `{"tap_mirror":{"id":"tm-1","name":"mirror","tenant_id":"legacy","directions":{"IN":"5"}}}`)
	})
	got, err := api.Get(context.Background(), "tm-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pythonTapMirrorBase+"/tm-1" || len(calls[0].query) != 0 || calls[0].body != "" {
		t.Fatalf("%+v", calls)
	}
	// Python's project_id falls back to tenant_id; Go keeps the two fields apart.
	if !(got.ID == "tm-1" && got.Name == "mirror" && got.TenantID == "legacy" && got.ProjectID == "" && got.Directions.In == 5) {
		t.Fatalf("%+v", got)
	}
	calls = nil
	api, _ = pythonTapMirrorAPI(t, &calls, func(*http.Request) *http.Response { return pythonTapMirrorWire(404, `{}`) })
	if _, err := api.Get(context.Background(), "tm-1"); !gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		t.Fatal(err)
	}
}

// update_tap_mirror(tap_mirror, name=..., description=..., x_extra=...)
func TestPythonTapMirrorUpdateBody(t *testing.T) {
	var calls []pythonTapMirrorCall
	api, _ := pythonTapMirrorAPI(t, &calls, func(*http.Request) *http.Response {
		return pythonTapMirrorWire(200, `{"tap_mirror":`+pythonTapMirrorRow+`}`)
	})
	name, description := "mirror", ""
	updated, err := api.Update(context.Background(), "tm-1", tapmirrors.UpdateOpts{Name: &name, Description: &description}, tapmirrors.WithUpdateField("x_extra", 1))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != pythonTapMirrorBase+"/tm-1" || len(calls[0].query) != 0 {
		t.Fatalf("%+v", calls)
	}
	want := pythonTapMirrorJSON(t, `{"tap_mirror":{"name":"mirror","description":"","x_extra":1}}`)
	if got := pythonTapMirrorJSON(t, calls[0].body); !reflect.DeepEqual(got, want) {
		t.Fatalf("body %s", calls[0].body)
	}
	if updated.ID != "tm-1" || updated.RemoteIP != "192.0.2.9" {
		t.Fatalf("%+v", updated)
	}
	// Python skips the request when no attribute is dirty; Go still sends PUT.
	calls = nil
	if _, err := api.Update(context.Background(), "tm-1", tapmirrors.UpdateOpts{}); err != nil || len(calls) != 1 || calls[0].body != `{"tap_mirror":{}}` {
		t.Fatalf("%v %+v", err, calls)
	}
}

// delete_tap_mirror(tap_mirror, ignore_missing=True|False)
func TestPythonTapMirrorDeleteIgnoreMissing(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		code    int
		call    func(*tapmirrors.API) error
		wantErr func(error) bool
	}{
		{"remove default 204", 204, func(a *tapmirrors.API) error { return a.Remove(ctx, resource.ID("tm-1")) }, func(err error) bool { return err == nil }},
		{"remove default 404", 404, func(a *tapmirrors.API) error { return a.Remove(ctx, resource.ID("tm-1")) }, func(err error) bool { return err == nil }},
		{"remove default 409", 409, func(a *tapmirrors.API) error { return a.Remove(ctx, resource.ID("tm-1")) }, func(err error) bool {
			return gophercloud.ResponseCodeIs(err, http.StatusConflict)
		}},
		{"remove strict 404", 404, func(a *tapmirrors.API) error { return a.Remove(ctx, resource.ID("tm-1"), resource.WithMissingError()) }, func(err error) bool {
			return errors.Is(err, resource.ErrNotFound) && gophercloud.ResponseCodeIs(err, http.StatusNotFound)
		}},
		{"ignore 204", 204, func(a *tapmirrors.API) error { return resource.IgnoreMissing(a.Delete(ctx, "tm-1")) }, func(err error) bool { return err == nil }},
		{"ignore 404", 404, func(a *tapmirrors.API) error { return resource.IgnoreMissing(a.Delete(ctx, "tm-1")) }, func(err error) bool { return err == nil }},
		{"ignore 409", 409, func(a *tapmirrors.API) error { return resource.IgnoreMissing(a.Delete(ctx, "tm-1")) }, func(err error) bool {
			return gophercloud.ResponseCodeIs(err, http.StatusConflict)
		}},
		{"plain 404", 404, func(a *tapmirrors.API) error { return a.Delete(ctx, "tm-1") }, func(err error) bool {
			return gophercloud.ResponseCodeIs(err, http.StatusNotFound)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonTapMirrorCall
			api, _ := pythonTapMirrorAPI(t, &calls, func(*http.Request) *http.Response { return pythonTapMirrorWire(tc.code, "") })
			if err := tc.call(api); !tc.wantErr(err) {
				t.Fatal(err)
			}
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != pythonTapMirrorBase+"/tm-1" || len(calls[0].query) != 0 {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

// find_tap_mirror(name_or_id): Python GETs the ID, then lists ?name= after
// 400/403/404. The closest Go chain is Find(ID, ignore) then Find(Name).
func TestPythonTapMirrorFindChain(t *testing.T) {
	ctx := context.Background()
	find := func(api *tapmirrors.API, value string, options ...resource.LookupOption) (*tapmirrors.TapMirror, error) {
		found, err := api.Find(ctx, resource.ID(value), resource.WithIgnoreMissing())
		if err != nil || found != nil {
			return found, err
		}
		return api.Find(ctx, resource.Name(value), options...)
	}
	for _, tc := range []struct {
		name    string
		getCode int
		list    string
		options []resource.LookupOption
		calls   int
		check   func(*tapmirrors.TapMirror, error) bool
	}{
		{"id hit", 200, "", nil, 1, func(v *tapmirrors.TapMirror, err error) bool { return err == nil && v.ID == "tm-1" }},
		{"name fallback", 404, `{"tap_mirrors":[` + pythonTapMirrorRow + `]}`, nil, 2, func(v *tapmirrors.TapMirror, err error) bool { return err == nil && v.ID == "tm-1" }},
		{"missing ignored", 404, `{"tap_mirrors":[]}`, []resource.LookupOption{resource.WithIgnoreMissing()}, 2, func(v *tapmirrors.TapMirror, err error) bool { return err == nil && v == nil }},
		{"missing strict", 404, `{"tap_mirrors":[]}`, nil, 2, func(v *tapmirrors.TapMirror, err error) bool { return v == nil && errors.Is(err, resource.ErrNotFound) }},
		{"duplicate", 404, `{"tap_mirrors":[` + pythonTapMirrorRow + `,{"id":"tm-2","name":"mirror","directions":{}}]}`, nil, 2, func(v *tapmirrors.TapMirror, err error) bool {
			return errors.Is(err, resource.ErrAmbiguous)
		}},
		// Python would continue with the name list after 400 or 403.
		{"400 stops", 400, "", nil, 1, func(v *tapmirrors.TapMirror, err error) bool {
			return gophercloud.ResponseCodeIs(err, http.StatusBadRequest)
		}},
		{"403 stops", 403, "", nil, 1, func(v *tapmirrors.TapMirror, err error) bool {
			return gophercloud.ResponseCodeIs(err, http.StatusForbidden)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []pythonTapMirrorCall
			api, _ := pythonTapMirrorAPI(t, &calls, func(req *http.Request) *http.Response {
				if req.URL.Path == pythonTapMirrorBase {
					return pythonTapMirrorWire(200, tc.list)
				}
				if tc.getCode == 200 {
					return pythonTapMirrorWire(200, `{"tap_mirror":`+pythonTapMirrorRow+`}`)
				}
				return pythonTapMirrorWire(tc.getCode, `{}`)
			})
			value := "mirror"
			if tc.getCode == 200 {
				value = "tm-1"
			}
			found, err := find(api, value, tc.options...)
			if !tc.check(found, err) {
				t.Fatal(found, err)
			}
			if len(calls) != tc.calls || calls[0].method != http.MethodGet || calls[0].path != pythonTapMirrorBase+"/"+value || len(calls[0].query) != 0 {
				t.Fatalf("%+v", calls)
			}
			if tc.calls == 2 && (calls[1].path != pythonTapMirrorBase || !reflect.DeepEqual(calls[1].query, url.Values{"name": {"mirror"}})) {
				t.Fatalf("%+v", calls[1])
			}
		})
	}
}

// tap_mirrors(name=..., project_id=..., limit=..., fields=[...])
func TestPythonTapMirrorListQueryAndPaging(t *testing.T) {
	ctx := context.Background()
	t.Run("query and links", func(t *testing.T) {
		var calls []pythonTapMirrorCall
		var cloud *testcloud.Cloud
		api, cloud := pythonTapMirrorAPI(t, &calls, func(req *http.Request) *http.Response {
			if req.URL.Query().Get("marker") == "" {
				return pythonTapMirrorWire(200, `{"tap_mirrors":[`+pythonTapMirrorRow+`],"tap_mirrors_links":[{"rel":"next","href":"`+cloud.Server.URL+pythonTapMirrorBase+`?limit=1&marker=tm-1&name=mirror&project_id=proj"}]}`)
			}
			return pythonTapMirrorWire(200, `{"tap_mirrors":[]}`)
		})
		rows, err := api.All(ctx, resource.WithQuery("name", "mirror"), resource.WithQuery("project_id", "proj"), resource.WithPageSize(1))
		if err != nil || len(rows) != 1 || rows[0].ID != "tm-1" || rows[0].Directions.Out != 102 {
			t.Fatal(rows, err)
		}
		want := []url.Values{
			{"name": {"mirror"}, "project_id": {"proj"}, "limit": {"1"}},
			{"name": {"mirror"}, "project_id": {"proj"}, "limit": {"1"}, "marker": {"tm-1"}},
		}
		if len(calls) != 2 || !reflect.DeepEqual([]url.Values{calls[0].query, calls[1].query}, want) || calls[0].path != pythonTapMirrorBase {
			t.Fatalf("%+v", calls)
		}
	})
	t.Run("full page without links stops", func(t *testing.T) {
		var calls []pythonTapMirrorCall
		api, _ := pythonTapMirrorAPI(t, &calls, func(*http.Request) *http.Response {
			return pythonTapMirrorWire(200, `{"tap_mirrors":[`+pythonTapMirrorRow+`]}`)
		})
		rows, err := api.All(ctx, resource.WithPageSize(1))
		if err != nil || len(rows) != 1 || len(calls) != 1 {
			t.Fatal(rows, err, calls)
		}
	})
	t.Run("repeated fields collapse", func(t *testing.T) {
		var calls []pythonTapMirrorCall
		api, _ := pythonTapMirrorAPI(t, &calls, func(*http.Request) *http.Response { return pythonTapMirrorWire(200, `{"tap_mirrors":[]}`) })
		if _, err := api.All(ctx, resource.WithQuery("fields", "id"), resource.WithQuery("fields", "name")); err != nil {
			t.Fatal(err)
		}
		for range api.List(ctx, tapmirrors.WithListQuery("sort_key", "name"), tapmirrors.WithListQuery("sort_key", "id")) {
		}
		want := []url.Values{{"fields": {"name"}}, {"sort_key": {"id"}}}
		if len(calls) != 2 || !reflect.DeepEqual([]url.Values{calls[0].query, calls[1].query}, want) {
			t.Fatalf("%+v", calls)
		}
	})
}
