package flavors_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/flavors"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type flavorTransport func(*http.Request) (*http.Response, error)

func (transport flavorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type flavorCall struct{ method, path, query, body, accept string }

const flavorBase = "/neutron/v2.0/flavors"

func flavorAPI(t *testing.T, calls *[]flavorCall, reply func(*http.Request) (int, string)) *flavors.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = flavorTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, flavorCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return flavors.New(client)
}

func flavorFailure(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "flavors" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
	if code == 0 {
		return
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if !errors.As(err, &native) || native.Actual != code {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const flavorRow = `{"id":"f1","name":"gold","description":"fast","enabled":true,"service_type":"L3_ROUTER_NAT","service_profiles":["sp1","sp2"],"x_ext":7}`

func flavorCheck(t *testing.T, v *flavors.Flavor) {
	t.Helper()
	if v == nil || v.ID != "f1" || *v.Name != "gold" || *v.Description != "fast" || !*v.IsEnabled || *v.ServiceType != "L3_ROUTER_NAT" || strings.Join(v.ServiceProfileIDs, ",") != "sp1,sp2" {
		t.Fatalf("decoded flavor: %#v", v)
	}
	if string(v.Resource.Body["is_enabled"]) != "true" || string(v.Resource.Body["service_profile_ids"]) != `["sp1","sp2"]` || string(v.Resource.Body["x_ext"]) != "7" || string(v.Wire.Body["enabled"]) != "true" {
		t.Fatalf("raw views: %#v %#v", v.Resource, v.Wire)
	}
}

func TestPythonFlavorCreatePostsAttrsWithWireNames(t *testing.T) {
	var calls []flavorCall
	api := flavorAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"flavor":` + flavorRow + `}` })
	v, err := api.Create(context.Background(),
		flavors.WithCreateField("name", "gold"),
		flavors.WithCreateField("is_enabled", true),
		flavors.WithCreateField("service_type", "L3_ROUTER_NAT"),
		flavors.WithCreateField("service_profile_ids", []string{"sp1", "sp2"}),
		flavors.WithCreateField("x_ext", 7))
	if err != nil {
		t.Fatal(err)
	}
	want := `{"flavor":{"enabled":true,"name":"gold","service_profiles":["sp1","sp2"],"service_type":"L3_ROUTER_NAT","x_ext":7}}`
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != flavorBase || calls[0].query != "" || calls[0].body != want {
		t.Fatal(calls)
	}
	flavorCheck(t, v)
	if v.StatusCode != 201 {
		t.Fatal(v.StatusCode)
	}

	calls = nil
	api = flavorAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"flavor":{}}` })
	if _, err := api.Create(context.Background()); err != nil || calls[0].body != `{"flavor":{}}` {
		t.Fatal(err, calls)
	}
	for _, code := range []int{400, 404, 409, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []flavorCall
			api := flavorAPI(t, &calls, func(*http.Request) (int, string) { return code, `{"NeutronError":{}}` })
			v, err := api.Create(context.Background(), flavors.WithCreateField("name", "x"))
			if v != nil || len(calls) != 1 {
				t.Fatal(v, calls)
			}
			flavorFailure(t, err, "Create", code)
		})
	}
	calls = nil
	api = flavorAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{}` })
	_, err = api.Create(context.Background(), flavors.WithCreateField("base_path", "/other"))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
		t.Fatal(err, calls)
	}
	flavorFailure(t, err, "Create", 0)
}

func TestPythonFlavorGetMapsFieldsAndReportsMissing(t *testing.T) {
	var calls []flavorCall
	api := flavorAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"flavor":` + flavorRow + `}` })
	v, err := api.Get(context.Background(), "f1", flavors.WithGetFields("is_enabled", "service_profile_ids", "name"))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != flavorBase+"/f1" || calls[0].query != "fields=enabled&fields=service_profiles&fields=name" {
		t.Fatal(calls)
	}
	flavorCheck(t, v)

	calls = nil
	api = flavorAPI(t, &calls, func(*http.Request) (int, string) { return 404, `{"NeutronError":{"type":"FlavorNotFound"}}` })
	v, err = api.Get(context.Background(), "a/b")
	if v != nil || len(calls) != 1 || calls[0].path != flavorBase+"/a%2Fb" || calls[0].query != "" {
		t.Fatal(v, calls)
	}
	flavorFailure(t, err, "Get", 404)
}

func TestPythonFlavorUpdatePutsOnlyGivenAttrs(t *testing.T) {
	var calls []flavorCall
	api := flavorAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"flavor":` + flavorRow + `}` })
	v, err := api.Update(context.Background(), "f1", flavors.WithUpdateField("description", "fast"), flavors.WithUpdateField("enabled", false))
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPut || calls[0].path != flavorBase+"/f1" || calls[0].body != `{"flavor":{"description":"fast","enabled":false}}` {
		t.Fatal(calls)
	}
	flavorCheck(t, v)

	calls = nil
	v, err = api.Update(context.Background(), "f1")
	if err != nil || len(calls) != 0 || v.ID != "f1" || v.Name != nil || v.Wire != nil {
		t.Fatal(v, err, calls)
	}
	for _, key := range []string{"id", "flavor"} {
		_, err = api.Update(context.Background(), "f1", flavors.WithUpdateField(key, "other"))
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(key, err, calls)
		}
	}
	api = flavorAPI(t, &calls, func(*http.Request) (int, string) { return 409, `{}` })
	_, err = api.Update(context.Background(), "f1", flavors.WithUpdateField("name", "x"))
	flavorFailure(t, err, "Update", 409)
}

func TestPythonFlavorDeleteIgnoresMissingByDefault(t *testing.T) {
	for _, tc := range []struct {
		code    int
		options []flavors.DeleteOption
		ignored bool
		fails   bool
	}{
		{204, nil, false, false},
		{404, nil, true, false},
		{404, []flavors.DeleteOption{flavors.WithDeleteIgnoreMissing(true)}, true, false},
		{404, []flavors.DeleteOption{flavors.WithDeleteIgnoreMissing(false)}, false, true},
		{409, nil, false, true},
		{500, nil, false, true},
	} {
		t.Run(fmt.Sprint(tc.code, tc.ignored, tc.fails), func(t *testing.T) {
			var calls []flavorCall
			api := flavorAPI(t, &calls, func(*http.Request) (int, string) { return tc.code, "" })
			result, err := api.Delete(context.Background(), "f1", tc.options...)
			if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != flavorBase+"/f1" || calls[0].body != "" {
				t.Fatal(calls)
			}
			if tc.fails {
				flavorFailure(t, err, "Delete", tc.code)
				return
			}
			if err != nil || result == nil || result.StatusCode != tc.code || result.IgnoredMissing != tc.ignored {
				t.Fatal(result, err)
			}
		})
	}
}

func TestPythonFlavorListMapsQueryFiltersLocallyAndFollowsLinks(t *testing.T) {
	var calls []flavorCall
	api := flavorAPI(t, &calls, func(req *http.Request) (int, string) {
		switch req.URL.Query().Get("marker") {
		case "b":
			return 200, `{"flavors":[{"id":"c","service_profiles":["sp1"]},{"id":"d","service_profiles":"sp1"}]}`
		case "d":
			return 200, `{"flavors":[]}`
		}
		return 200, `{"flavors":[{"id":"a","service_profiles":["sp1"]},{"id":"b","service_profiles":[]}],"flavors_links":[{"rel":"next","href":"?enabled=True&limit=2&marker=b&service_type=L3"}]}`
	})
	values, err := api.All(context.Background(),
		flavors.WithListQuery("is_enabled", true),
		flavors.WithListQuery("service_type", "L3"),
		flavors.WithListQuery("service_profile_ids", []string{"sp1"}),
		flavors.WithListQuery("unknown", make(chan int)),
		flavors.WithListLimit(2))
	if err != nil {
		t.Fatal(err)
	}
	// A full last page without links continues with its last ID as marker.
	if len(calls) != 3 || calls[0].path != flavorBase || calls[0].query != "enabled=True&limit=2&service_type=L3" || calls[0].accept != "application/json" || calls[1].query != "enabled=True&limit=2&marker=b&service_type=L3" || calls[2].query != "enabled=True&limit=2&marker=d&service_type=L3" {
		t.Fatal(calls)
	}
	// service_profiles is a list descriptor, so the scalar "sp1" becomes ["sp1"].
	if len(values) != 3 || values[0].ID != "a" || values[1].ID != "c" || values[2].ID != "d" || strings.Join(values[2].ServiceProfileIDs, ",") != "sp1" {
		t.Fatalf("%#v", values)
	}

	calls = nil
	api = flavorAPI(t, &calls, func(*http.Request) (int, string) { return 503, `{}` })
	_, err = api.All(context.Background())
	flavorFailure(t, err, "List", 503)
}

func TestPythonFlavorFindFallsBackToNameList(t *testing.T) {
	var calls []flavorCall
	api := flavorAPI(t, &calls, func(req *http.Request) (int, string) {
		if req.URL.Path != flavorBase {
			return 404, ""
		}
		return 200, `{"flavors":[` + flavorRow + `]}`
	})
	v, err := api.Find(context.Background(), "gold")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0].path != flavorBase+"/gold" || calls[1].path != flavorBase || calls[1].query != "name=gold" {
		t.Fatal(calls)
	}
	flavorCheck(t, v)

	calls = nil
	api = flavorAPI(t, &calls, func(*http.Request) (int, string) { return 200, `{"flavor":` + flavorRow + `}` })
	if v, err = api.Find(context.Background(), "f1"); err != nil || len(calls) != 1 || calls[0].path != flavorBase+"/f1" {
		t.Fatal(v, err, calls)
	}
	for _, strict := range []bool{false, true} {
		calls = nil
		api = flavorAPI(t, &calls, func(req *http.Request) (int, string) {
			if req.URL.Path != flavorBase {
				return 400, ""
			}
			return 200, `{"flavors":[]}`
		})
		v, err = api.Find(context.Background(), "none", flavors.WithFindIgnoreMissing(!strict))
		if v != nil || len(calls) != 2 || strict != errors.Is(err, resource.ErrNotFound) {
			t.Fatal(strict, v, err, calls)
		}
	}
	calls = nil
	api = flavorAPI(t, &calls, func(*http.Request) (int, string) { return 500, "" })
	_, err = api.Find(context.Background(), "gold")
	if len(calls) != 1 {
		t.Fatal(calls)
	}
	flavorFailure(t, err, "Find", 500)
}

func TestPythonFlavorAssociateServiceProfileReturnsResponseJSON(t *testing.T) {
	var calls []flavorCall
	api := flavorAPI(t, &calls, func(*http.Request) (int, string) { return 201, `{"service_profile":{"id":"sp1"}}` })
	v, err := api.AssociateServiceProfile(context.Background(), "f/1", "sp1")
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodPost || calls[0].path != flavorBase+"/f%2F1/service_profiles" || calls[0].query != "" || calls[0].body != `{"service_profile":{"id":"sp1"}}` {
		t.Fatal(calls)
	}
	if v.ServiceProfileID == nil || *v.ServiceProfileID != "sp1" || string(v.Body) != `{"service_profile":{"id":"sp1"}}` || v.StatusCode != 201 {
		t.Fatalf("%#v", v)
	}

	// Python returns the error JSON for raise_exc=False; Go reports the status.
	for _, code := range []int{400, 404, 409} {
		calls = nil
		api = flavorAPI(t, &calls, func(*http.Request) (int, string) { return code, `{"NeutronError":{"type":"Conflict"}}` })
		v, err = api.AssociateServiceProfile(context.Background(), "f1", "sp1")
		if v != nil || len(calls) != 1 {
			t.Fatal(v, calls)
		}
		flavorFailure(t, err, "AssociateServiceProfile", code)
		var native gophercloud.ErrUnexpectedResponseCode
		if errors.As(err, &native); !strings.Contains(string(native.Body), "NeutronError") {
			t.Fatal(string(native.Body))
		}
	}
	calls = nil
	api = flavorAPI(t, &calls, func(*http.Request) (int, string) { return 204, "" })
	_, err = api.AssociateServiceProfile(context.Background(), "f1", "sp1")
	var receipt *resource.ResponseError
	if !errors.As(err, &receipt) || receipt.StatusCode != 204 {
		t.Fatal(err)
	}
	flavorFailure(t, err, "AssociateServiceProfile", 0)
	calls = nil
	for _, ids := range [][2]string{{"", "sp1"}, {"f1", " "}, {"..", "sp1"}} {
		_, err = api.AssociateServiceProfile(context.Background(), ids[0], ids[1])
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatal(ids, err, calls)
		}
	}
}

func TestPythonFlavorDisassociateServiceProfileDeletesNestedMember(t *testing.T) {
	var calls []flavorCall
	api := flavorAPI(t, &calls, func(*http.Request) (int, string) { return 204, "" })
	if err := api.DisassociateServiceProfile(context.Background(), "f1", "sp/1"); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || calls[0].method != http.MethodDelete || calls[0].path != flavorBase+"/f1/service_profiles/sp%2F1" || calls[0].body != "" {
		t.Fatal(calls)
	}
	// Python ignores every status here; Go reports 400 and above, even 404.
	for _, code := range []int{404, 409, 500} {
		calls = nil
		api = flavorAPI(t, &calls, func(*http.Request) (int, string) { return code, `{}` })
		err := api.DisassociateServiceProfile(context.Background(), "f1", "sp1")
		if len(calls) != 1 {
			t.Fatal(calls)
		}
		flavorFailure(t, err, "DisassociateServiceProfile", code)
	}
}
