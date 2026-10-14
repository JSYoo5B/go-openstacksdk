package federation_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/identity/v3/federation"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pyMappingRecordTransport func(*http.Request) (*http.Response, error)

func (transport pyMappingRecordTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pyMappingRecordCall struct{ method, path, query, body, accept string }

func pyMappingRecordAPI(t *testing.T, calls *[]pyMappingRecordCall, reply func(*http.Request) (int, string)) *federation.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pyMappingRecordTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pyMappingRecordCall{req.Method, req.URL.EscapedPath(), req.URL.RawQuery, raw, req.Header.Get("Accept")})
		code, body := reply(req)
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("identity", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/keystone/v3/"
	return federation.New(client)
}

func pyMappingRecordOperation(t *testing.T, err error, operation string, code int) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "mappings" {
		t.Fatalf("missing %s operation context: %v", operation, err)
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if code != 0 && (!errors.As(err, &native) || native.Actual != code) {
		t.Fatalf("want HTTP %d, got %v", code, err)
	}
}

const pyMappingRecordBase = "/keystone/v3/OS-FEDERATION/mappings"
const pyMappingRecordRules = `[{"local":[{"user":{"name":"{0}","type":"ephemeral"},"projects":[{"name":"p","roles":[{"name":"member"}]}]}],"remote":[{"type":"REMOTE_USER","any_one_of":["a"]}]}]`
const pyMappingRecordRow = `{"id":"m1","schema_version":"2.0","rules":` + pyMappingRecordRules + `,"links":{"self":"x"}}`

func pyMappingRecordCheck(t *testing.T, v *federation.MappingRecord) {
	t.Helper()
	if v == nil || v.ID != "m1" || v.SchemaVersion == nil || *v.SchemaVersion != "2.0" || string(v.Rules) != pyMappingRecordRules {
		t.Fatalf("decoded mapping: %#v", v)
	}
	if len(v.Body) != 3 || string(v.Body["schema_version"]) != `"2.0"` || string(v.Wire["links"]) != `{"self":"x"}` {
		t.Fatalf("raw views: %#v", v)
	}
}

func TestPythonMappingRecordGetKeepsSchemaVersionAndRawRules(t *testing.T) {
	for _, body := range []string{`{"mapping":` + pyMappingRecordRow + `}`, pyMappingRecordRow} {
		var calls []pyMappingRecordCall
		api := pyMappingRecordAPI(t, &calls, func(*http.Request) (int, string) { return 200, body })
		v, err := api.GetMappingRecord(context.Background(), "m1")
		if err != nil || len(calls) != 1 || calls[0].method != http.MethodGet || calls[0].path != pyMappingRecordBase+"/m1" || calls[0].query != "" || calls[0].body != "" {
			t.Fatal(err, calls)
		}
		pyMappingRecordCheck(t, v)
	}
	for _, code := range []int{203, 399, 400, 404, 500} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			var calls []pyMappingRecordCall
			api := pyMappingRecordAPI(t, &calls, func(*http.Request) (int, string) { return code, `{"mapping":{"id":"m1"}}` })
			v, err := api.GetMappingRecord(context.Background(), "a/b")
			if calls[0].path != pyMappingRecordBase+"/a%2Fb" {
				t.Fatal(calls)
			}
			if code < 400 {
				if err != nil || v.ID != "m1" || v.SchemaVersion != nil || v.Rules != nil || string(v.Body["rules"]) != "null" {
					t.Fatal(v, err)
				}
				return
			}
			pyMappingRecordOperation(t, err, "Get", code)
		})
	}
}

func TestPythonMappingRecordListFiltersPagingFindUpdateDelete(t *testing.T) {
	var calls []pyMappingRecordCall
	api := pyMappingRecordAPI(t, &calls, func(*http.Request) (int, string) {
		return 200, `{"mappings":[` + pyMappingRecordRow + `,{"id":"m2","schema_version":"1.0","rules":[]}],"links":{"next":null}}`
	})
	values, err := api.AllMappingRecords(context.Background())
	if err != nil || len(values) != 2 || len(calls) != 1 || calls[0].path != pyMappingRecordBase || calls[0].query != "" || calls[0].accept != "application/json" {
		t.Fatal(err, values, calls)
	}
	pyMappingRecordCheck(t, values[0])

	calls = nil
	values, err = api.AllMappingRecords(context.Background(), federation.WithMappingRecordListQuery("schema_version", "1.0"), federation.WithMappingRecordListQuery("unknown", 1))
	if err != nil || len(values) != 1 || values[0].ID != "m2" || calls[0].query != "" {
		t.Fatal(err, values, calls)
	}
	calls = nil
	values, err = api.AllMappingRecords(context.Background(), federation.WithMappingRecordListQuery("name", "m1"))
	if err != nil || len(values) != 1 || values[0].ID != "m1" || calls[0].query != "" {
		t.Fatal(err, values, calls)
	}

	calls = nil
	pages := []string{`{"mappings":[{"id":"a"}]}`, `{"mappings":[]}`}
	api = pyMappingRecordAPI(t, &calls, func(*http.Request) (int, string) {
		page := pages[0]
		pages = pages[1:]
		return 200, page
	})
	values, err = api.AllMappingRecords(context.Background(), federation.WithMappingRecordListLimit(1), federation.WithMappingRecordListMarker("start"))
	if err != nil || len(values) != 1 || len(calls) != 2 || calls[0].query != "limit=1&marker=start" || calls[1].query != "limit=1&marker=a" {
		t.Fatal(err, values, calls)
	}

	calls = nil
	api = pyMappingRecordAPI(t, &calls, func(req *http.Request) (int, string) {
		switch {
		case req.Method == http.MethodPatch:
			return 200, `{"mapping":` + pyMappingRecordRow + `}`
		case req.URL.Path == pyMappingRecordBase+"/m1":
			return 404, `{}`
		case req.Method == http.MethodDelete:
			return 404, `{}`
		}
		return 200, `{"mappings":[` + pyMappingRecordRow + `]}`
	})
	v, err := api.FindMappingRecord(context.Background(), "m1")
	if err != nil || len(calls) != 2 || calls[1].path != pyMappingRecordBase || calls[1].query != "" {
		t.Fatal(err, calls)
	}
	pyMappingRecordCheck(t, v)

	calls = nil
	v, err = api.UpdateMappingRecord(context.Background(), "m1", federation.WithMappingRecordField("name", "x"), federation.WithMappingRecordField("schema_version", "2.0"), federation.WithMappingRecordField("rules", []any{}))
	if err != nil || len(calls) != 1 || calls[0].method != http.MethodPatch || calls[0].body != `{"mapping":{"rules":[],"schema_version":"2.0"}}` {
		t.Fatal(err, calls)
	}
	pyMappingRecordCheck(t, v)

	calls = nil
	if err := api.DeleteMappingRecord(context.Background(), "m1"); err != nil || calls[0].method != http.MethodDelete {
		t.Fatal(err, calls)
	}
	err = api.DeleteMappingRecord(context.Background(), "m1", federation.WithMappingRecordDeleteIgnoreMissing(false))
	pyMappingRecordOperation(t, err, "Delete", 404)

	calls = nil
	api = pyMappingRecordAPI(t, &calls, func(*http.Request) (int, string) { return 403, `{}` })
	_, err = api.AllMappingRecords(context.Background())
	pyMappingRecordOperation(t, err, "List", 403)
}
