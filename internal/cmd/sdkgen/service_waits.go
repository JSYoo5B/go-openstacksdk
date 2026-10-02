package main

import (
	"fmt"
	"go/types"
)

// Only these audited native parents have additive service-default wait methods.
// This does not replace native WaitForStatus or the shared collection policy.
func serviceWaitCollectionPolicy(pkg *types.Package, plan *collectionPlan) (string, error) {
	var policy, model string
	switch sdkPath(pkg.Path()) {
	case "compute/v2/servers":
		policy, model = "compute", "Server"
	case "blockstorage/v2/volumes", "blockstorage/v3/volumes":
		policy, model = "cinder", "Volume"
	case "blockstorage/v2/snapshots", "blockstorage/v3/snapshots":
		policy, model = "cinder", "Snapshot"
	case "image/v2/images":
		policy, model = "image", "Image"
	default:
		return "", nil
	}
	if plan == nil || plan.modelName != model || plan.getter == nil || plan.lister == nil || plan.deleter == nil || plan.id != "ID" || plan.name != "Name" || plan.status != "Status" {
		return "", fmt.Errorf("service wait policy %s requires its %s ID/name/status collection with GET/list/delete", sdkPath(pkg.Path()), model)
	}
	return policy, nil
}
