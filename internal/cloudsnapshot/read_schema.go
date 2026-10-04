package cloudsnapshot

import (
	"context"
	"encoding/json"

	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// These schemas are library-owned. Callers select concrete resource APIs;
// they never supply a transport, normalizer or query builder.
type readSchema struct {
	route, singular, plural, kind                string
	listOperation, searchOperation, getOperation string
	bodyAttributes                               []string
	bindAllProjects                              bool
	normalize                                    func(json.RawMessage, *string, bool, resource.CloudLocation) (json.RawMessage, bool, error)
}

func snapshotReadSchema() readSchema {
	attrs := make([]string, len(descriptors))
	for i, field := range descriptors {
		attrs[i] = field.attribute
	}
	return readSchema{
		route: "snapshots", singular: "snapshot", plural: "snapshots", kind: "volume snapshot",
		listOperation: "ListVolumeSnapshots", searchOperation: "SearchVolumeSnapshots", getOperation: "GetVolumeSnapshot",
		bodyAttributes: attrs, bindAllProjects: true, normalize: normalize,
	}
}

func backupReadSchema() readSchema {
	attrs := make([]string, len(backupDescriptors))
	for i, field := range backupDescriptors {
		attrs[i] = field.attribute
	}
	return readSchema{
		route: "backups", singular: "backup", plural: "backups", kind: "volume backup",
		listOperation: "ListVolumeBackups", searchOperation: "SearchVolumeBackups", getOperation: "GetVolumeBackup",
		bodyAttributes: attrs, normalize: normalizeBackup,
	}
}

func wrapRead(ctx context.Context, schema readSchema, operation string, err error) error {
	return request.Wrap(operation, schema.kind, cloudread.ContextError(ctx, err))
}
