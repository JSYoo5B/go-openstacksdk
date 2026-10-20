package quotasets

import (
	"context"

	"github.com/JSYoo5B/go-openstacksdk/internal/quotafetch"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// QuotaReadOpts is the concrete Source-shaped quota fetch input. Usage defaults
// to false and is always sent as Requests' False/True query spelling.
type QuotaReadOpts = quotafetch.Options
type QuotaReadOption = quotafetch.Option

// QuotaRecord separates the normalized SDK view from literal response evidence.
type QuotaRecord = quotafetch.Record

func WithQuotaReadOpts(v QuotaReadOpts) QuotaReadOption { return quotafetch.WithOptions(v) }
func WithQuotaReadQuery(key string, value any) QuotaReadOption {
	return quotafetch.WithQuery(key, value)
}
func WithQuotaUsage(v bool) QuotaReadOption              { return quotafetch.WithUsage(v) }
func WithQuotaReadMicroversion(v string) QuotaReadOption { return quotafetch.WithMicroversion(v) }
func WithQuotaReadBasePath(v string) QuotaReadOption     { return quotafetch.WithBasePath(v) }
func WithQuotaReadRequiresID(v bool) QuotaReadOption     { return quotafetch.WithRequiresID(v) }
func WithQuotaReadSkipCache(v bool) QuotaReadOption      { return quotafetch.WithSkipCache(v) }
func WithQuotaReadErrorMessage(v string) QuotaReadOption { return quotafetch.WithErrorMessage(v) }
func WithQuotaReadResponseKey(v string) QuotaReadOption  { return quotafetch.WithResponseKey(v) }

var quotaReadProfile = quotafetch.Profile{Kind: "block storage quota", Role: "volume", IntegerFields: []string{
	"backup_gigabytes", "backups", "gigabytes", "groups", "per_volume_gigabytes", "snapshots", "volumes",
}}

// GetRecord implements the pinned get_quota_set(project, usage=False, **query)
// contract, including inherited fetch controls and common quota normalization.
// Existing Get/Usage retain their reviewed native response model and defaults.
func (s *ProjectQuotaScope) GetRecord(ctx context.Context, options ...QuotaReadOption) (*QuotaRecord, error) {
	var client *gophercloud.ServiceClient
	projectID := ""
	if s != nil {
		projectID = s.projectID
		if s.api != nil {
			client = s.api.client
		}
	}
	value, err := quotafetch.Read(ctx, client, projectID, "", quotaReadProfile, options...)
	return value, request.Wrap("GetRecord", "block storage quota", err)
}

func WithQuotaReadLocation(v resource.CloudLocation) QuotaReadOption {
	return quotafetch.WithLocation(v)
}
