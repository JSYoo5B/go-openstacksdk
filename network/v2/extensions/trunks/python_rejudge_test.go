package trunks_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/trunks"
)

// update_trunk(trunk, **attrs) PUTs the passed attributes under their wire
// names. Trunk does not inherit NetworkResource, so no If-Match is sent.
func TestPythonRejudgeTrunkUpdate(t *testing.T) {
	type call struct {
		method, path, ifMatch string
		body                  any
	}
	var calls []call
	cloud := testcloud.New(t)
	cloud.SetTransport(testcloud.LinkRequest(func(req *http.Request) (*http.Response, error) {
		var body any
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			if err := json.Unmarshal(data, &body); err != nil {
				t.Errorf("body %q: %v", data, err)
			}
		}
		calls = append(calls, call{req.Method, req.URL.Path, req.Header.Get("If-Match"), body})
		reply := `{"trunk":{"id":"t-1","name":"t2","description":"","admin_state_up":false,"port_id":"p-1","status":"DOWN","sub_ports":[],"project_id":"proj","tenant_id":"proj","revision_number":4}}`
		return testcloud.Response(200, reply, http.Header{"Content-Type": {"application/json"}}), nil
	}))
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	api := trunks.New(client)

	name, description, down := "t2", "", false
	got, err := api.Update(context.Background(), "t-1", trunks.UpdateOpts{Name: &name, Description: &description, AdminStateUp: &down}, trunks.WithUpdateField("x_vendor", 1))
	if err != nil || got.ID != "t-1" || got.Name != "t2" || got.Description != "" || got.AdminStateUp || got.PortID != "p-1" || got.Status != "DOWN" || got.ProjectID != "proj" {
		t.Fatal(got, err)
	}
	if _, err := api.Update(context.Background(), "t-1", trunks.UpdateOpts{Name: &name}); err != nil {
		t.Fatal(err)
	}
	var want1, want2 any
	_ = json.Unmarshal([]byte(`{"trunk":{"name":"t2","description":"","admin_state_up":false,"x_vendor":1}}`), &want1)
	_ = json.Unmarshal([]byte(`{"trunk":{"name":"t2"}}`), &want2)
	want := []call{
		{"PUT", "/neutron/v2.0/trunks/t-1", "", want1},
		{"PUT", "/neutron/v2.0/trunks/t-1", "", want2},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls\n got %+v\nwant %+v", calls, want)
	}
}
