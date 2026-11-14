package v2_test

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	sdk "github.com/JSYoo5B/go-openstacksdk"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

func runRbacPolicyMutations(ctx context.Context, cloud, mode, policyID, objectID, target string) (any, error) {
	conn, err := sdk.Connect(ctx, sdk.WithCloud(cloud))
	if err != nil {
		return nil, err
	}
	network, err := conn.NetworkV2(ctx)
	if err != nil {
		return nil, err
	}
	switch mode {
	case "create":
		value, err := network.RBACPolicies.CreateRecord(ctx,
			request.Field("object_type", "network"),
			request.Field("object_id", objectID),
			request.Field("action", "access_as_shared"),
			request.Field("target_project_id", target),
		)
		if err != nil {
			return nil, err
		}
		return value.Resource, nil
	case "update":
		value, err := network.RBACPolicies.UpdateRecord(ctx, policyID,
			request.Field("target_project_id", target),
		)
		if err != nil {
			return nil, err
		}
		return value.Resource, nil
	case "delete":
		return network.RBACPolicies.DeleteRecord(ctx, policyID)
	default:
		return nil, fmt.Errorf("mode must be create, update or delete")
	}
}

func Example_rbacPolicyMutations() {
	cloud := flag.String("cloud", "devstack", "clouds.yaml cloud")
	mode := flag.String("mode", "", "create, update or delete")
	policyID := flag.String("policy", "", "literal RBAC policy ID")
	objectID := flag.String("object", "", "network ID for create")
	target := flag.String("target-project", "", "concrete target project ID")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	value, err := runRbacPolicyMutations(ctx, *cloud, *mode, *policyID, *objectID, *target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
