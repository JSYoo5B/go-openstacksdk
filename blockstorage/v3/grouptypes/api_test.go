package grouptypes_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/grouptypes"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type gtTransport func(*http.Request) (*http.Response, error)

func (f gtTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, err := f(r)
	if v != nil && v.Request == nil {
		v.Request = r
	}
	return v, err
}

type gtCall struct {
	method, path, query, body string
	header                    http.Header
}

const (
	gtBase      = "/cinder/v3/project/group_types"
	gtDiscovery = `{"version":{"id":"v3.0","status":"CURRENT","version":"3.70","min_version":"3.0"}}`
	gtRow       = `{"id":"gt1","name":"gold","description":"d","is_public":true,"group_specs":{"consistent_group_snapshot_enabled":"<is> True"},"extra":1}`
)

// gtAPI answers version discovery at /cinder/v3/ and every other request
// through reply.
func gtAPI(t *testing.T, version string, reply func(*http.Request) (int, string)) (*grouptypes.API, *[]gtCall) {
	t.Helper()
	cloud := testcloud.New(t)
	calls := &[]gtCall{}
	cloud.Provider.HTTPClient.Transport = gtTransport(func(r *http.Request) (*http.Response, error) {
		body := ""
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			body = string(raw)
		}
		*calls = append(*calls, gtCall{r.Method, r.URL.EscapedPath(), r.URL.RawQuery, body, r.Header.Clone()})
		code, text := 200, gtDiscovery
		if r.URL.Path != "/cinder/v3/" {
			code, text = reply(r)
		}
		return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(text))}, nil
	})
	client := cloud.Client("volumev3", "/cinder/v3/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	client.Microversion = version
	return grouptypes.New(client), calls
}

func gtOperation(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "group_types" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if code != 0 && (!errors.As(err, &native) || native.Actual != code) {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

func gtVersion(t *testing.T, call gtCall, want string) {
	t.Helper()
	if want == "" {
		if call.header.Get("OpenStack-API-Version") != "" || call.header.Get("X-OpenStack-Volume-API-Version") != "" {
			t.Fatalf("unexpected microversion headers: %v", call.header)
		}
		return
	}
	if call.header.Get("OpenStack-API-Version") != "volume "+want || call.header.Get("X-OpenStack-Volume-API-Version") != want {
		t.Fatalf("want microversion %s, got %v", want, call.header)
	}
}

func gtCheck(t *testing.T, v *grouptypes.GroupType) {
	t.Helper()
	if v == nil || v.ID != "gt1" || v.Name != "gold" || *v.Description != "d" || !*v.IsPublic ||
		!reflect.DeepEqual(v.GroupSpecs, map[string]string{"consistent_group_snapshot_enabled": "<is> True"}) {
		t.Fatalf("decoded group type: %#v", v)
	}
	if len(v.Body) != 5 || string(v.Wire["extra"]) != "1" {
		t.Fatalf("raw views: %#v", v)
	}
}

func TestGroupTypeGetNegotiatesMicroversion311(t *testing.T) {
	api, calls := gtAPI(t, "", func(*http.Request) (int, string) { return 200, `{"group_type":` + gtRow + `}` })
	v, err := api.Get(context.Background(), "gt1")
	if err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 2 || (*calls)[0].path != "/cinder/v3/" || (*calls)[1].method != http.MethodGet || (*calls)[1].path != gtBase+"/gt1" {
		t.Fatal(*calls)
	}
	gtVersion(t, (*calls)[0], "")
	gtVersion(t, (*calls)[1], "3.11")
	gtCheck(t, v)
	if v.StatusCode != 200 || len(v.Discovery) != 1 || v.Discovery[0].StatusCode != 200 {
		t.Fatalf("evidence: %#v", v)
	}

	// A configured microversion is sent as is, without discovery.
	api, calls = gtAPI(t, "3.5", func(*http.Request) (int, string) { return 200, gtRow })
	v, err = api.Get(context.Background(), "a/b")
	if err != nil || len(*calls) != 1 || (*calls)[0].path != gtBase+"/a%2Fb" {
		t.Fatal(err, *calls)
	}
	gtVersion(t, (*calls)[0], "3.5")
	if v.ID != "gt1" {
		t.Fatalf("root object decode: %#v", v)
	}

	// An unknown group_specs keeps the Python default {}.
	api, _ = gtAPI(t, "3.11", func(*http.Request) (int, string) { return 200, `{"group_type":{"id":"gt2"}}` })
	v, err = api.Get(context.Background(), "gt2")
	if err != nil || string(v.Body["group_specs"]) != "{}" || v.GroupSpecs == nil || string(v.Body["name"]) != "null" {
		t.Fatal(err, v)
	}
}

func TestGroupTypeStatusPolicyAndRejections(t *testing.T) {
	for _, code := range []int{200, 203, 299, 399, 400, 403, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			api, _ := gtAPI(t, "3.11", func(*http.Request) (int, string) { return code, `{"group_type":` + gtRow + `}` })
			v, err := api.Get(context.Background(), "gt1")
			if code < 400 {
				if err != nil || v.StatusCode != code {
					t.Fatal(err, v)
				}
				return
			}
			gtOperation(t, err, "Get", code)
		})
	}
	for _, body := range []string{`[]`, `{"group_type":null}`, `"x"`} {
		api, _ := gtAPI(t, "3.11", func(*http.Request) (int, string) { return 200, body })
		v, err := api.Get(context.Background(), "gt1")
		var accepted *resource.ResponseError
		if !errors.As(err, &accepted) || v == nil || string(v.Envelope) != body {
			t.Fatalf("%s: %v %#v", body, err, v)
		}
		gtOperation(t, err, "Get", 0)
	}
	// Invalid JSON keeps the seeded id, as _translate_response ignores it.
	api, _ := gtAPI(t, "3.11", func(*http.Request) (int, string) { return 202, `not json` })
	v, err := api.Get(context.Background(), "gt1")
	if err != nil || v.ID != "gt1" || v.Wire != nil {
		t.Fatal(err, v)
	}
	if _, err := api.Get(context.Background(), " "); !errors.Is(err, resource.ErrInvalidOption) {
		t.Fatal(err)
	}
}

func TestGroupTypeCreatePostsEnvelope(t *testing.T) {
	api, calls := gtAPI(t, "", func(*http.Request) (int, string) { return 202, `{"group_type":` + gtRow + `}` })
	v, err := api.Create(context.Background(),
		grouptypes.WithCreateField("name", "gold"),
		grouptypes.WithCreateField("is_public", true),
		grouptypes.WithCreateField("group_specs", map[string]string{"k": "v"}),
		grouptypes.WithCreateField("unknown", make(chan int)))
	if err != nil {
		t.Fatal(err)
	}
	call := (*calls)[1]
	if call.method != http.MethodPost || call.path != gtBase || call.header.Get("Content-Type") != "application/json" {
		t.Fatal(call)
	}
	gtVersion(t, call, "3.11")
	var got, want any
	_ = json.Unmarshal([]byte(call.body), &got)
	_ = json.Unmarshal([]byte(`{"group_type":{"name":"gold","is_public":true,"group_specs":{"k":"v"}}}`), &want)
	if !reflect.DeepEqual(got, want) {
		t.Fatal(call.body)
	}
	gtCheck(t, v)

	api, _ = gtAPI(t, "3.11", func(*http.Request) (int, string) { return 400, `{"badRequest":{}}` })
	_, err = api.Create(context.Background(), grouptypes.WithCreateField("name", "x"))
	gtOperation(t, err, "Create", 400)
}

func TestGroupTypeUpdateSendsDirtyAttributesOnly(t *testing.T) {
	api, calls := gtAPI(t, "", func(*http.Request) (int, string) { return 200, `{"group_type":` + gtRow + `}` })
	v, err := api.Update(context.Background(), "gt1", grouptypes.WithUpdateField("description", "d"), grouptypes.WithUpdateField("id", "other"))
	if err != nil {
		t.Fatal(err)
	}
	call := (*calls)[1]
	if call.method != http.MethodPut || call.path != gtBase+"/gt1" || call.body != `{"group_type":{"description":"d"}}` {
		t.Fatal(call)
	}
	gtVersion(t, call, "3.11")
	gtCheck(t, v)

	// Nothing dirty: no discovery and no update request.
	api, calls = gtAPI(t, "", func(*http.Request) (int, string) { t.Fatal("unexpected request"); return 0, "" })
	v, err = api.Update(context.Background(), "gt1", grouptypes.WithUpdateField("id", "x"), grouptypes.WithUpdateField("unknown", 1))
	if err != nil || len(*calls) != 0 || v.ID != "gt1" || v.StatusCode != 0 {
		t.Fatal(err, *calls, v)
	}
}

func TestGroupTypeDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, item := range []struct {
		code    int
		options []grouptypes.DeleteOption
		fail    bool
	}{{202, nil, false}, {204, nil, false}, {404, nil, false}, {404, []grouptypes.DeleteOption{grouptypes.WithDeleteIgnoreMissing(false)}, true}, {409, nil, true}} {
		api, calls := gtAPI(t, "3.11", func(*http.Request) (int, string) { return item.code, `{}` })
		err := api.Delete(context.Background(), "gt1", item.options...)
		if (*calls)[0].method != http.MethodDelete || (*calls)[0].path != gtBase+"/gt1" {
			t.Fatal(*calls)
		}
		if item.fail {
			gtOperation(t, err, "Delete", item.code)
		} else if err != nil {
			t.Fatal(item.code, err)
		}
	}
}

func TestGroupTypeListQueryFiltersAndPaging(t *testing.T) {
	api, calls := gtAPI(t, "3.11", func(r *http.Request) (int, string) {
		if r.URL.Query().Get("marker") == "" {
			return 200, `{"group_types":[` + gtRow + `,{"id":"gt9","name":"other"}],"group_types_links":[{"rel":"next","href":"http://` + r.Host + `/cinder/v3/project/group_types?marker=gt9&is_public=True"}]}`
		}
		return 200, `{"group_types":[{"id":"gt3","name":"gold","description":null}]}`
	})
	values, err := api.All(context.Background(),
		grouptypes.WithListQuery("is_public", true),
		grouptypes.WithListQuery("all_projects", true),
		grouptypes.WithListQuery("name", "gold"),
		grouptypes.WithListQuery("unknown", "x"))
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 2 || values[0].ID != "gt1" || values[1].ID != "gt3" {
		t.Fatalf("filtered rows: %#v", values)
	}
	if (*calls)[0].query != "all_tenants=True&is_public=True" || (*calls)[0].header.Get("Accept") != "application/json" {
		t.Fatal((*calls)[0])
	}
	if len(*calls) != 2 || (*calls)[1].path != gtBase {
		t.Fatal(*calls)
	}
	gtVersion(t, (*calls)[1], "3.11")

	api, calls = gtAPI(t, "3.11", func(*http.Request) (int, string) { return 200, `{"group_types":[` + gtRow + `]}` })
	values, err = api.All(context.Background(), grouptypes.WithListLimit(1), grouptypes.WithListMarker("m"), grouptypes.WithListPaginated(false))
	if err != nil || len(values) != 1 || (*calls)[0].query != "limit=1&marker=m" || len(*calls) != 1 {
		t.Fatal(err, *calls)
	}

	api, _ = gtAPI(t, "3.11", func(*http.Request) (int, string) { return 500, `{}` })
	_, err = api.All(context.Background())
	gtOperation(t, err, "List", 500)
}

func TestGroupTypeFindUsesIsPublicNone(t *testing.T) {
	list := `{"group_types":[` + gtRow + `,{"id":"gt2","name":"silver"}]}`
	api, calls := gtAPI(t, "3.11", func(r *http.Request) (int, string) {
		if r.URL.Path == gtBase+"/gold" {
			return 404, `{"itemNotFound":{}}`
		}
		return 200, list
	})
	v, err := api.Find(context.Background(), "gold")
	if err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 2 || (*calls)[0].path != gtBase+"/gold" || (*calls)[0].query != "is_public=none" || (*calls)[1].path != gtBase || (*calls)[1].query != "is_public=none" {
		t.Fatal(*calls)
	}
	gtCheck(t, v)

	api, calls = gtAPI(t, "3.11", func(*http.Request) (int, string) { return 200, `{"group_type":` + gtRow + `}` })
	v, err = api.Find(context.Background(), "gt1")
	if err != nil || len(*calls) != 1 || v.ID != "gt1" {
		t.Fatal(err, *calls)
	}

	api, _ = gtAPI(t, "3.11", func(r *http.Request) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/missing") {
			return 404, `{}`
		}
		return 200, list
	})
	v, err = api.Find(context.Background(), "missing")
	if err != nil || v != nil {
		t.Fatal(err, v)
	}
	_, err = api.Find(context.Background(), "missing", grouptypes.WithFindIgnoreMissing(false))
	if !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	gtOperation(t, err, "Find", 0)

	api, _ = gtAPI(t, "3.11", func(r *http.Request) (int, string) {
		if strings.HasSuffix(r.URL.Path, "/gold") {
			return 400, `{}`
		}
		return 200, `{"group_types":[{"id":"a","name":"gold"},{"id":"b","name":"gold"}]}`
	})
	_, err = api.Find(context.Background(), "gold")
	var ambiguous *resource.AmbiguousError
	if !errors.As(err, &ambiguous) {
		t.Fatal(err)
	}
}

func TestGroupTypeGroupSpecs(t *testing.T) {
	api, calls := gtAPI(t, "", func(r *http.Request) (int, string) {
		return 200, `{"group_specs":{"a":"1"}}`
	})
	v, err := api.FetchGroupSpecs(context.Background(), "gt1")
	if err != nil || (*calls)[1].method != http.MethodGet || (*calls)[1].path != gtBase+"/gt1/group_specs" {
		t.Fatal(err, *calls)
	}
	gtVersion(t, (*calls)[1], "3.11")
	if v.ID != "gt1" || !reflect.DeepEqual(v.GroupSpecs, map[string]string{"a": "1"}) || v.Wire != nil || v.StatusCode != 200 {
		t.Fatalf("fetched specs: %#v", v)
	}

	api, calls = gtAPI(t, "3.11", func(*http.Request) (int, string) { return 200, `{}` })
	v, err = api.CreateGroupSpecs(context.Background(), "gt1", map[string]any{"a": "1"})
	if err != nil || (*calls)[0].method != http.MethodPost || (*calls)[0].path != gtBase+"/gt1/group_specs" || (*calls)[0].body != `{"group_specs":{"a":"1"}}` {
		t.Fatal(err, *calls)
	}
	if string(v.Body["group_specs"]) != "{}" {
		t.Fatalf("absent group_specs must default to {}: %#v", v)
	}
	for _, body := range []string{`[]`, `not json`} {
		api, _ = gtAPI(t, "3.11", func(*http.Request) (int, string) { return 200, body })
		_, err = api.FetchGroupSpecs(context.Background(), "gt1")
		var accepted *resource.ResponseError
		if !errors.As(err, &accepted) {
			t.Fatal(body, err)
		}
		gtOperation(t, err, "FetchGroupSpecs", 0)
	}

	api, calls = gtAPI(t, "3.11", func(*http.Request) (int, string) { return 200, `{"a":"1"}` })
	p, err := api.GetGroupSpecsProperty(context.Background(), "gt1", "a")
	if err != nil || (*calls)[0].path != gtBase+"/gt1/group_specs/a" || string(p.Value) != `"1"` || p.Key != "a" || p.GroupTypeID != "gt1" {
		t.Fatal(err, *calls, p)
	}
	api, _ = gtAPI(t, "3.11", func(*http.Request) (int, string) { return 200, `{"b":"1"}` })
	p, err = api.GetGroupSpecsProperty(context.Background(), "gt1", "a")
	if err != nil || string(p.Value) != "null" {
		t.Fatal(err, p)
	}

	api, calls = gtAPI(t, "3.11", func(*http.Request) (int, string) { return 200, `{"a":"2"}` })
	p, err = api.UpdateGroupSpecsProperty(context.Background(), "gt1", "a", "2")
	if err != nil || (*calls)[0].method != http.MethodPut || (*calls)[0].body != `{"a":"2"}` || string(p.Value) != `"2"` {
		t.Fatal(err, *calls, p)
	}
	api, _ = gtAPI(t, "3.11", func(*http.Request) (int, string) { return 200, `{"b":"2"}` })
	_, err = api.UpdateGroupSpecsProperty(context.Background(), "gt1", "a", "2")
	gtOperation(t, err, "UpdateGroupSpecsProperty", 0)

	for _, item := range []struct {
		code int
		keep bool
		fail bool
	}{{200, false, false}, {404, false, false}, {404, true, true}, {403, false, true}} {
		api, calls = gtAPI(t, "3.11", func(*http.Request) (int, string) { return item.code, `` })
		var options []grouptypes.DeleteOption
		if item.keep {
			options = append(options, grouptypes.WithDeleteIgnoreMissing(false))
		}
		err = api.DeleteGroupSpecsProperty(context.Background(), "gt1", "a", options...)
		if (*calls)[0].method != http.MethodDelete || (*calls)[0].path != gtBase+"/gt1/group_specs/a" {
			t.Fatal(*calls)
		}
		if item.fail {
			gtOperation(t, err, "DeleteGroupSpecsProperty", item.code)
		} else if err != nil {
			t.Fatal(err)
		}
	}
}
