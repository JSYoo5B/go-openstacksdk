package usages_test

import (
	"context"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/placement/v1/usages"
)

type pythonUsageTransport func(*http.Request) (*http.Response, error)

func (transport pythonUsageTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

type pythonUsageCall struct{ method, path, query, version string }

// Usage declares _max_microversion 1.38, so the helper pins that version.
func pythonUsageAPI(t *testing.T, calls *[]pythonUsageCall, body string) *usages.API {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = pythonUsageTransport(func(req *http.Request) (*http.Response, error) {
		*calls = append(*calls, pythonUsageCall{req.Method, req.URL.Path, req.URL.RawQuery, req.Header.Get("OpenStack-API-Version")})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}, nil
	})
	client := cloud.Client("placement", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/placement/"
	client.Microversion = "1.38"
	return usages.New(client)
}

func TestPythonUsagesProxyCall(t *testing.T) {
	ctx := context.Background()
	var calls []pythonUsageCall
	api := pythonUsageAPI(t, &calls, `{"usages":{"INSTANCE":{"consumer_count":2,"VCPU":4,"MEMORY_MB":1024},"MIGRATION":{"consumer_count":1,"VCPU":2}}}`)
	// usages("p-1", user_id="u-1", consumer_type="INSTANCE")
	got, err := api.Get(ctx, usages.WithGetOptions(usages.GetOpts{ProjectID: "p-1", UserID: "u-1", ConsumerType: "INSTANCE"}))
	if err != nil {
		t.Fatal(err)
	}
	// Python yields one Usage per consumer type with consumer_count split from resources.
	want := map[string]usages.ConsumerTypeUsage{"INSTANCE": {"consumer_count": 2, "VCPU": 4, "MEMORY_MB": 1024}, "MIGRATION": {"consumer_count": 1, "VCPU": 2}}
	if !reflect.DeepEqual(got.Usages, want) {
		t.Fatal(got.Usages)
	}
	// usages("p-1") drops the None filters.
	if _, err := api.Get(ctx, usages.WithGetOptions(usages.GetOpts{ProjectID: "p-1"})); err != nil {
		t.Fatal(err)
	}
	wantCalls := []pythonUsageCall{
		{http.MethodGet, "/placement/usages", "consumer_type=INSTANCE&project_id=p-1&user_id=u-1", "placement 1.38"},
		{http.MethodGet, "/placement/usages", "project_id=p-1", "placement 1.38"},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("%+v", calls)
	}
}

// Python accepts the flat pre-1.38 shape as one Usage; Go Get rejects it.
func TestPythonUsagesPre138ShapeIsNotDecoded(t *testing.T) {
	var calls []pythonUsageCall
	api := pythonUsageAPI(t, &calls, `{"usages":{"VCPU":2}}`)
	if _, err := api.Get(context.Background(), usages.WithGetOptions(usages.GetOpts{ProjectID: "p-1"})); err == nil {
		t.Fatal("decoded flat usages")
	}
}
