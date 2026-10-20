package main

import (
	"fmt"
	"go/types"
)

// These manual scopes supplement four existing native collections. Metadata
// is neither a new CRUD collection nor a capability inferred for other models.
func cinderMetadataCollectionScope(pkg *types.Package, plan *collectionPlan) (string, error) {
	model := ""
	switch sdkPath(pkg.Path()) {
	case "blockstorage/v2/volumes", "blockstorage/v3/volumes":
		model = "Volume"
	case "blockstorage/v2/snapshots", "blockstorage/v3/snapshots":
		model = "Snapshot"
	default:
		return "", nil
	}
	if plan == nil || plan.modelName != model || plan.getter == nil || plan.lister == nil || plan.id != "ID" || plan.name != "Name" {
		return "", fmt.Errorf("Cinder metadata scope %s requires its %s ID/name collection", sdkPath(pkg.Path()), model)
	}
	return "MetadataIn", nil
}
