package v2_test

import (
	"context"
	"fmt"
	"log"

	"encoding/json"
	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/security/addressgroups"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func networkV2(ctx context.Context) *v2.Service {
	conn, err := openstack.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	service, err := conn.NetworkV2(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return service
}

func Example_findPortRecord() {
	ctx := context.Background()
	port, err := networkV2(ctx).Ports.FindRecord(ctx, "app-port",
		request.Query("binding_profile", map[string]any{"nested": map[string]any{"enabled": true}}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(port)
}

func Example_deleteBandwidthLimitRule() {
	ctx := context.Background()
	net := networkV2(ctx)
	if err := resource.IgnoreMissing(net.QoSRules.DeleteBandwidthLimitRule(ctx, "qos-policy-id", "rule-id")); err != nil {
		log.Fatal(err)
	}
	v := "edge-firewall"
	found, err := net.FirewallGroups.Find(ctx, resource.ID(v), resource.WithIgnoreMissing())
	if err == nil && found == nil {
		found, err = net.FirewallGroups.Find(ctx, resource.Name(v), resource.WithIgnoreMissing())
	}
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(found)
}

func Example_deleteWithOptions() {
	ctx := context.Background()
	api := addressgroups.New(networkV2(ctx).RawClient())
	id := "address-group-id"
	if err := api.DeleteWithOptions(ctx, id); err != nil {
		log.Fatal(err)
	}
	if err := api.DeleteWithOptions(ctx, id, request.IgnoreMissing(false)); err != nil {
		log.Fatal(err)
	}
	strict := false
	if err := api.DeleteWithOptions(ctx, id, addressgroups.WithDeleteOptions(addressgroups.DeleteOpts{IgnoreMissing: &strict})); err != nil {
		log.Fatal(err)
	}
}

func Example_addExtraRoutesRecord() {
	ctx := context.Background()
	api := routers.New(networkV2(ctx).RawClient())
	cached := &resource.RawResource{Metadata: resource.Metadata{Body: map[string]json.RawMessage{
		"name":   json.RawMessage(`"cached-edge"`),
		"routes": json.RawMessage(`[]`),
	}}}
	row, err := api.AddExtraRoutesRecord(ctx, "ROUTER_ID", json.RawMessage(`{"router":{"routes":[]}}`),
		routers.WithRouterActionSeed(cached),
		request.Headers(map[string]string{"X-Client-Trace": "router-action"}))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(row)
}
