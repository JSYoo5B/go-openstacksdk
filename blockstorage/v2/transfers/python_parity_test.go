package transfers_test

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/blockstorage/v2/transfers"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// pythonTransferCall records the wire request openstacksdk block_storage v2 would also send.
type pythonTransferCall struct{ method, path, query, body string }

func pythonTransferWire(code int, body string) *http.Response {
	return testcloud.Response(code, body, http.Header{"Content-Type": {"application/json"}})
}

func pythonTransferAPI(t *testing.T, reply func(*http.Request) *http.Response) (*transfers.API, *testcloud.Recorder) {
	t.Helper()
	calls := testcloud.NewRecorder(func(req *http.Request) (*http.Response, error) { return reply(req), nil })
	cloud := testcloud.New(t)
	cloud.SetTransport(calls)
	client := cloud.Client("block-storage", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/cinder/v2/project/"
	return transfers.New(client), calls
}

// pythonTransferCalls projects recorded calls onto the compared fields.
func pythonTransferCalls(rec *testcloud.Recorder) []pythonTransferCall {
	calls := []pythonTransferCall{}
	for _, call := range rec.Calls() {
		calls = append(calls, pythonTransferCall{call.Method, call.Path, call.RawQuery, call.Body})
	}
	return calls
}

const pythonTransferBase = "/cinder/v2/project/os-volume-transfer"

const pythonTransferRow = `{"id":"tr-1","name":"handoff","volume_id":"vol-1","auth_key":"secret","links":[{"rel":"self","href":"x"}],"created_at":"2026-10-11T01:02:03.000000"}`

func TestPythonTransferCreateGetAcceptMatchProxyRequests(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonTransferAPI(t, func(req *http.Request) *http.Response {
		if req.Method == http.MethodPost {
			return pythonTransferWire(202, `{"transfer":`+pythonTransferRow+`}`)
		}
		return pythonTransferWire(200, `{"transfer":`+pythonTransferRow+`}`)
	})
	// create_transfer(volume_id="vol-1", name="handoff")
	created, err := api.Create(ctx, transfers.CreateOpts{VolumeID: "vol-1", Name: "handoff"})
	if err != nil || created.ID != "tr-1" || created.AuthKey != "secret" || created.VolumeID != "vol-1" || created.Links[0]["rel"] != "self" || created.CreatedAt.Second() != 3 {
		t.Fatal(created, err)
	}
	// get_transfer(transfer)
	got, err := api.Get(ctx, "tr-1")
	if err != nil || got.Name != "handoff" {
		t.Fatal(got, err)
	}
	// accept_transfer(transfer, auth_key="secret")
	accepted, err := api.Accept(ctx, "tr-1", transfers.AcceptOpts{AuthKey: "secret"})
	if err != nil || accepted.ID != "tr-1" || accepted.VolumeID != "vol-1" {
		t.Fatal(accepted, err)
	}
	want := []pythonTransferCall{
		{http.MethodPost, pythonTransferBase, "", `{"transfer":{"name":"handoff","volume_id":"vol-1"}}`},
		{http.MethodGet, pythonTransferBase + "/tr-1", "", ""},
		{http.MethodPost, pythonTransferBase + "/tr-1/accept", "", `{"accept":{"auth_key":"secret"}}`},
	}
	if !reflect.DeepEqual(pythonTransferCalls(calls), want) {
		t.Fatalf("%+v", pythonTransferCalls(calls))
	}
}

func TestPythonTransferDeleteIgnoresMissing(t *testing.T) {
	ctx := context.Background()
	for _, code := range []int{http.StatusAccepted, http.StatusNotFound, http.StatusConflict} {
		t.Run(http.StatusText(code), func(t *testing.T) {
			api, calls := pythonTransferAPI(t, func(*http.Request) *http.Response {
				return pythonTransferWire(code, `{}`)
			})
			// delete_transfer(transfer) defaults to ignore_missing=True.
			err := api.Remove(ctx, resource.ID("tr-1"))
			if code == http.StatusConflict {
				if !gophercloud.ResponseCodeIs(err, code) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			// delete_transfer(transfer, ignore_missing=False)
			strict := api.Remove(ctx, resource.ID("tr-1"), resource.WithMissingError())
			if code == http.StatusNotFound && !errors.Is(strict, resource.ErrNotFound) {
				t.Fatal(strict)
			}
			if calls.Len() != 2 || pythonTransferCalls(calls)[0] != (pythonTransferCall{http.MethodDelete, pythonTransferBase + "/tr-1", "", ""}) {
				t.Fatalf("%+v", pythonTransferCalls(calls))
			}
		})
	}
}

func TestPythonTransferListAndFindUseDifferentRoutes(t *testing.T) {
	ctx := context.Background()
	api, calls := pythonTransferAPI(t, func(req *http.Request) *http.Response {
		if req.URL.Path == pythonTransferBase+"/detail" {
			return pythonTransferWire(200, `{"transfers":[`+pythonTransferRow+`]}`)
		}
		return pythonTransferWire(http.StatusNotFound, `{}`)
	})
	// transfers() lists /volume-transfers/detail; Go lists the legacy os-volume-transfer/detail route.
	for _, err := range api.List(ctx) {
		if err != nil {
			t.Fatal(err)
		}
	}
	// find_transfer("handoff") falls back to the summary /os-volume-transfer list; Go lists detail.
	if value, err := api.Find(ctx, resource.ID("handoff"), resource.WithIgnoreMissing()); err != nil || value != nil {
		t.Fatal(value, err)
	}
	if value, err := api.Find(ctx, resource.Name("handoff")); err != nil || value.ID != "tr-1" {
		t.Fatal(value, err)
	}
	want := []pythonTransferCall{
		{http.MethodGet, pythonTransferBase + "/detail", "", ""},
		{http.MethodGet, pythonTransferBase + "/handoff", "", ""},
		{http.MethodGet, pythonTransferBase + "/detail", "", ""},
	}
	if !reflect.DeepEqual(pythonTransferCalls(calls), want) {
		t.Fatalf("%+v", pythonTransferCalls(calls))
	}
}
