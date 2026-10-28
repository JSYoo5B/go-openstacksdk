package v3_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"testing"

	v3 "github.com/JSYoo5B/go-openstacksdk/blockstorage/v3"
	"github.com/JSYoo5B/go-openstacksdk/common/extensions"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
)

// pythonCinderExtensionCall records the wire request openstacksdk would also send.
type pythonCinderExtensionCall struct{ method, path, query, body string }

// extensions() lists the Extension resource from the Cinder endpoint; Go reuses the service client with common/extensions.
func TestPythonCinderExtensionsList(t *testing.T) {
	ctx := context.Background()
	var calls []pythonCinderExtensionCall
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		calls = append(calls, pythonCinderExtensionCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		body := `{"extensions":[{"alias":"os-hosts","name":"Hosts","description":"Admin-only host administration.","namespace":"","updated":"2011-06-29T00:00:00+00:00","links":[]},{"alias":"backups","name":"Backups","description":"Backups support.","updated":"2012-12-12T00:00:00+00:00","links":[{"rel":"describedby","href":"https://example.invalid"}]}]}`
		return testcloud.Response(200, body, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	api := extensions.New(v3.New(client).RawClient())
	var rows []extensions.Extension
	for value, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, *value)
	}
	if want := []pythonCinderExtensionCall{{http.MethodGet, "/cinder/v3/project/extensions", "", ""}}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("%+v", calls)
	}
	// Python exposes alias, name, description, links and updated_at (wire key updated).
	if len(rows) != 2 || rows[0].Alias != "os-hosts" || rows[0].Name != "Hosts" || rows[0].Description != "Admin-only host administration." ||
		rows[0].Updated != "2011-06-29T00:00:00+00:00" || len(rows[0].Links) != 0 || rows[1].Alias != "backups" || len(rows[1].Links) != 1 {
		t.Fatalf("%+v", rows)
	}
}
