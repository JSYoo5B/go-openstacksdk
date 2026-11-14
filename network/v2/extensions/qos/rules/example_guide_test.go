package rules_test

import (
	"context"
	"fmt"
	"log"

	"encoding/json"
	openstack "github.com/JSYoo5B/go-openstacksdk"
	v2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/request"
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

func Example_minimumPacketRate() {
	ctx := context.Background()
	network := networkV2(ctx)
	policyID := "qos-policy-id"
	rule, err := network.QoSRules.CreateMinimumPacketRateRecord(ctx, policyID,
		request.Field("min_kpps", 1000),
		request.Field("direction", "any"))
	if err != nil {
		log.Fatal(err)
	}
	var ruleID string
	if err := json.Unmarshal(rule.Resource.Body["id"], &ruleID); err != nil {
		log.Fatal(err)
	}
	if _, err := network.QoSRules.UpdateMinimumPacketRateRecord(ctx, policyID, ruleID, request.Field("min_kpps", 2000)); err != nil {
		log.Fatal(err)
	}
	result, err := network.QoSRules.DeleteMinimumPacketRateRecord(ctx, policyID, ruleID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
