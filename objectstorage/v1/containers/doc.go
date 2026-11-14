// Package containers manages Swift containers.
//
// Containers.CreateContainer takes CreateContainerOpts and sends metadata and
// system headers with PUT201/202. Containers.DeleteContainer expects DELETE204;
// by default a 404 is returned as IgnoredMissing with the actual response, and
// a 409 for a nonempty container stays an error. Neither call issues another
// HEAD or deletes objects.
//
// Containers.GetMetadata takes GetMetadataOpts and returns user metadata,
// nullable counters and raw timestamps from HEAD204. Containers.SetMetadata and
// Containers.DeleteMetadata return the single POST204 acknowledgement;
// SetMetadataAndRefresh reads again, and DeleteMetadataKeys sends POST even for
// no keys. ListRecords, CreateRecord and DeleteRecord follow the Python record
// shapes.
package containers
