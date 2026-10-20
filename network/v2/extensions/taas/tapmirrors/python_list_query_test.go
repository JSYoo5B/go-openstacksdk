package tapmirrors_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/taas/tapmirrors"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// tap_mirrors(name="mirror", fields=["id", "name"], sort_key=["name", "id"],
// sort_dir=["asc", "desc"], limit=1)
func TestPythonTapMirrorListRepeatedQueryValues(t *testing.T) {
	ctx := context.Background()
	var calls []pythonTapMirrorCall
	var cloud *testcloud.Cloud
	api, cloud := pythonTapMirrorAPI(t, &calls, func(req *http.Request) *http.Response {
		if req.URL.Query().Get("marker") == "" {
			next := cloud.Server.URL + pythonTapMirrorBase + "?" + req.URL.RawQuery + "&marker=tm-1"
			return pythonTapMirrorWire(200, `{"tap_mirrors":[`+pythonTapMirrorRow+`],"tap_mirrors_links":[{"rel":"next","href":"`+next+`"}]}`)
		}
		return pythonTapMirrorWire(200, `{"tap_mirrors":[]}`)
	})
	var rows []*tapmirrors.TapMirror
	for row, err := range api.List(ctx,
		tapmirrors.WithListQuery("name", "mirror"),
		tapmirrors.WithListQueryValues("fields", "id", "name"),
		tapmirrors.WithListQueryValues("sort_key", "project_id"),
		tapmirrors.WithListQueryValues("sort_key", "name", "id"),
		tapmirrors.WithListQueryValues("sort_dir", "asc", "desc"),
		tapmirrors.WithListQueryValues("dropped", "x"),
		tapmirrors.WithListQueryValues("dropped"),
		tapmirrors.WithListQuery("limit", "1"),
	) {
		if err != nil {
			t.Fatal(err)
		}
		rows = append(rows, row)
	}
	if len(rows) != 1 || rows[0].ID != "tm-1" || rows[0].PortID != "p1" {
		t.Fatal(rows)
	}
	first := url.Values{"name": {"mirror"}, "fields": {"id", "name"}, "sort_key": {"name", "id"}, "sort_dir": {"asc", "desc"}, "limit": {"1"}}
	second := url.Values{"name": {"mirror"}, "fields": {"id", "name"}, "sort_key": {"name", "id"}, "sort_dir": {"asc", "desc"}, "limit": {"1"}, "marker": {"tm-1"}}
	if len(calls) != 2 || calls[0].method != http.MethodGet || calls[0].path != pythonTapMirrorBase || !reflect.DeepEqual(calls[0].query, first) || !reflect.DeepEqual(calls[1].query, second) {
		t.Fatalf("%+v", calls)
	}
	// A later single-value option for the key wins.
	calls = nil
	for range api.List(ctx, tapmirrors.WithListQueryValues("fields", "id", "name"), tapmirrors.WithListQuery("fields", "id")) {
	}
	if len(calls) != 2 || !reflect.DeepEqual(calls[0].query, url.Values{"fields": {"id"}}) || !reflect.DeepEqual(calls[1].query, url.Values{"fields": {"id"}, "marker": {"tm-1"}}) {
		t.Fatalf("%+v", calls)
	}
}

func TestPythonTapMirrorListRepeatedQueryRejectsInputAndStatus(t *testing.T) {
	ctx := context.Background()
	var calls []pythonTapMirrorCall
	api, _ := pythonTapMirrorAPI(t, &calls, func(*http.Request) *http.Response {
		return pythonTapMirrorWire(http.StatusBadRequest, `{"NeutronError":{"type":"HTTPBadRequest","message":"bad sort key"}}`)
	})
	for _, err := range api.List(ctx, tapmirrors.WithListQueryValues(" ", "id")) {
		var operation *resource.OperationError
		if !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "List" {
			t.Fatal(err)
		}
	}
	if len(calls) != 0 {
		t.Fatalf("%+v", calls)
	}
	seen := false
	for _, err := range api.List(ctx, tapmirrors.WithListQueryValues("sort_key", "bogus", "id")) {
		seen = true
		if !gophercloud.ResponseCodeIs(err, http.StatusBadRequest) {
			t.Fatal(err)
		}
	}
	if !seen || len(calls) != 1 || !reflect.DeepEqual(calls[0].query, url.Values{"sort_key": {"bogus", "id"}}) {
		t.Fatalf("%+v", calls)
	}
}
