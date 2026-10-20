package availabilityzones

import (
	"context"
	"iter"
)

// ListDetailRecords lazily reads the details=True branch of Python
// availability_zones from os-availability-zone/detail. Rows use the same
// AvailabilityZoneRecord view as ListRecords, so hosts and their service
// states remain raw JSON in Resource, Wire and Envelope. The options, success
// range, paging and versionless default are those of ListRecords.
func (a *API) ListDetailRecords(ctx context.Context, options ...AvailabilityZoneListOption) iter.Seq2[*AvailabilityZoneRecord, error] {
	return a.listZoneRecords(ctx, "ListDetailRecords", "os-availability-zone/detail", options)
}
