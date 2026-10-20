package transfers_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v3/transfers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// pythonTransferCall records the wire request openstacksdk would also send.
type pythonTransferCall struct{ method, path, query, body string }

type pythonTransferTransport func(*http.Request) (*http.Response, error)

func (transport pythonTransferTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func pythonTransferWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

func pythonTransferAPI(t *testing.T, reply func(*http.Request) *http.Response) (*transfers.API, *[]pythonTransferCall) {
	t.Helper()
	calls := &[]pythonTransferCall{}
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonTransferTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, pythonTransferCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v3/project/"
	return transfers.New(client), calls
}

// Python uses this legacy route only when microversion 3.55 is unavailable;
// otherwise it uses /volume-transfers, which Go never sends.
const pythonTransferLegacy = "/cinder/v3/project/os-volume-transfer"

const pythonTransferRow = `{"id":"tr-1","name":"handoff","volume_id":"vol-1","auth_key":"secret","links":[{"rel":"self","href":"x"}],"created_at":"2026-10-11T01:02:03.000000"}`

func TestPythonTransferGetAcceptDeleteUseLegacyRoute(t *testing.T) {
	ctx := context.Background()
	status := http.StatusAccepted
	api, calls := pythonTransferAPI(t, func(req *http.Request) *http.Response {
		switch req.Method {
		case http.MethodDelete:
			return pythonTransferWire(status, `{"itemNotFound":{"message":"gone"}}`)
		case http.MethodPost:
			return pythonTransferWire(202, `{"transfer":{"id":"tr-1","name":"handoff","volume_id":"vol-1","links":[]}}`)
		}
		return pythonTransferWire(200, `{"transfer":`+pythonTransferRow+`}`)
	})
	// get_transfer(transfer)
	got, err := api.Get(ctx, "tr-1")
	if err != nil || got.ID != "tr-1" || got.Name != "handoff" || got.VolumeID != "vol-1" || got.CreatedAt.Second() != 3 {
		t.Fatal(got, err)
	}
	// accept_transfer(transfer, auth_key) returns the transfer from the response.
	accepted, err := api.Accept(ctx, "tr-1", transfers.AcceptOpts{AuthKey: "secret"})
	if err != nil || accepted.ID != "tr-1" || accepted.VolumeID != "vol-1" {
		t.Fatal(accepted, err)
	}
	// delete_transfer(transfer) ignores 404 by default; ignore_missing=False raises.
	if err := api.Remove(ctx, resource.ID("tr-1")); err != nil {
		t.Fatal(err)
	}
	status = http.StatusNotFound
	if err := api.Remove(ctx, resource.ID("tr-1")); err != nil {
		t.Fatal(err)
	}
	if err := api.Remove(ctx, resource.ID("tr-1"), resource.WithMissingError()); !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(err)
	}
	want := []pythonTransferCall{
		{http.MethodGet, pythonTransferLegacy + "/tr-1", "", ""},
		{http.MethodPost, pythonTransferLegacy + "/tr-1/accept", "", `{"accept":{"auth_key":"secret"}}`},
		{http.MethodDelete, pythonTransferLegacy + "/tr-1", "", ""},
		{http.MethodDelete, pythonTransferLegacy + "/tr-1", "", ""},
		{http.MethodDelete, pythonTransferLegacy + "/tr-1", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Fatalf("%+v", *calls)
	}
}

func TestPythonTransferCreateListAndFindGaps(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonTransferAPI(t, func(req *http.Request) *http.Response {
		switch {
		case req.Method == http.MethodPost:
			return pythonTransferWire(202, `{"transfer":`+pythonTransferRow+`}`)
		case req.URL.Path == pythonTransferLegacy+"/detail":
			return pythonTransferWire(200, `{"transfers":[`+pythonTransferRow+`]}`)
		}
		return pythonTransferWire(404, `{"itemNotFound":{"message":"gone"}}`)
	})
	// create_transfer(volume_id=..., name=..., no_snapshots=True) needs /volume-transfers at 3.55;
	// Go can only add the key to the legacy route.
	created, err := api.Create(ctx, transfers.CreateOpts{VolumeID: "vol-1", Name: "handoff"}, transfers.WithCreateField("no_snapshots", true))
	if err != nil || created.AuthKey != "secret" {
		t.Fatal(created, err)
	}
	// transfers(all_projects=True) with the default details=True.
	var ids []string
	for value, err := range api.List(ctx, transfers.WithListQuery("all_tenants", "True")) {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, value.ID)
	}
	// find_transfer("handoff") would GET by ID and then list /volume-transfers and match locally.
	byID, err := api.Find(ctx, resource.ID("handoff"), resource.WithIgnoreMissing())
	if err != nil || byID != nil {
		t.Fatal(byID, err)
	}
	byName, err := api.Find(ctx, resource.Name("handoff"))
	if err != nil || byName.ID != "tr-1" || !reflect.DeepEqual(ids, []string{"tr-1"}) {
		t.Fatal(byName, err, ids)
	}
	listQuery, _ := url.ParseQuery((*calls)[1].query)
	want := []pythonTransferCall{
		{http.MethodPost, pythonTransferLegacy, "", `{"transfer":{"name":"handoff","no_snapshots":true,"volume_id":"vol-1"}}`},
		{http.MethodGet, pythonTransferLegacy + "/detail", (*calls)[1].query, ""},
		{http.MethodGet, pythonTransferLegacy + "/handoff", "", ""},
		{http.MethodGet, pythonTransferLegacy + "/detail", "", ""},
	}
	if !reflect.DeepEqual(*calls, want) || !reflect.DeepEqual(listQuery, url.Values{"all_tenants": {"True"}}) {
		t.Fatalf("%+v", *calls)
	}
}
