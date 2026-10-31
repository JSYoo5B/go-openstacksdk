package openstack

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/image"
	"github.com/gophercloud/gophercloud/v2"
)

func TestConnectionImageCreatePolicyCapturesExplicitOptions(t *testing.T) {
	raw := json.RawMessage(`"raw"`)
	vendor := map[string]any{"owned": "value"}
	option := WithImageCreatePolicy(image.WithImageCreatePolicyOpts(image.ImageCreatePolicy{ImageFormat: raw, DisableVendorAgent: vendor}))
	raw[1] = 'x'
	vendor["owned"] = "changed"
	connection, err := FromProvider(&gophercloud.ProviderClient{}, option, WithEndpointFor(Image, "v2", "https://cloud.test/v2/"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := connection.Image(context.Background())
	if err != nil || service == nil {
		t.Fatal(service, err)
	}
	if string(connection.options.imageCreatePolicy.ImageFormat) != `"raw"` || string(connection.options.imageCreatePolicy.DisableVendorAgent["owned"].(json.RawMessage)) != `"value"` {
		t.Fatal(connection.options.imageCreatePolicy)
	}
}
