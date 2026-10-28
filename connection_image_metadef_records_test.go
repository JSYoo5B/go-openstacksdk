package openstack_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageMetadefRecordsShareClientAndCurrentLocation(t *testing.T) {
	const endpoint = "https://cloud.test/reverse/glance/v2/"
	cloud := "configured-cloud"
	facts := resource.CloudLocation{Cloud: &cloud, Project: resource.CloudProject{ID: json.RawMessage(`"current-project"`)}}
	expected, err := facts.ForResource(nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	provider := &gophercloud.ProviderClient{}
	calls := 0
	provider.HTTPClient.Transport = serviceInfoTransport(func(req *http.Request) (*http.Response, error) {
		calls++
		path, body := "metadefs/resource_types", `{"resource_types":[{"name":"global"}]}`
		if calls == 2 {
			path, body = "metadefs/namespaces/OS::Nova/resource_types", `{"resource_type_associations":[{"name":"scoped","location":null}]}`
		}
		if calls > 3 || req.Method != http.MethodGet || req.URL.String() != endpoint+path || req.Body != nil {
			t.Fatal("unexpected request", req.Method, req.URL)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"X-Proof": {"owned"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	conn, err := sdk.FromProvider(provider, sdk.WithEndpointFor(sdk.Image, "v2", endpoint), sdk.WithCloudLocation(facts))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cloud = "caller mutation"
	service, err := conn.Image(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	versioned, err := conn.ImageV2(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if service.API.MetadefResourceTypes.RawClient() != versioned.RawClient() || versioned.RawClient().ProviderClient != provider || calls != 0 {
		t.Fatal("binding made a request or lost the shared client")
	}
	rows, err := service.API.MetadefResourceTypes.AllRecords(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 || string(rows[0].Resource.Body["location"]) != string(expected) || rows[0].Namespace != nil {
		t.Fatal("missing owned current location", rows)
	}
	scope, err := service.API.MetadefResourceTypes.InNamespace(context.Background(), "OS::Nova")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rows, err = scope.AllRecords(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 || string(rows[0].Wire.Body["location"]) != "null" || string(rows[0].Resource.Body["location"]) != string(expected) || string(rows[0].Resource.Body["namespace_name"]) != `"OS::Nova"` || rows[0].Namespace == nil || *rows[0].Namespace != "OS::Nova" || calls != 2 {
		t.Fatal("scoped URI or explicit location lost", rows, calls)
	}
	versionedRows, err := versioned.MetadefResourceTypes.AllRecords(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versionedRows) != 1 || string(versionedRows[0].Resource.Body["location"]) != string(expected) || calls != 3 {
		t.Fatal("versioned entry point lost Connection location", versionedRows, calls)
	}
}
