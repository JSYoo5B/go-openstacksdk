package quotasets

import (
	"context"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/quotafetch"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// QuotaReadOpts is the concrete Source-shaped quota fetch input. Usage chooses
// Nova's detail endpoint; Query accepts arbitrary JSON-domain fetch parameters.
type QuotaReadOpts = quotafetch.Options
type QuotaReadOption = quotafetch.Option

// QuotaRecord separates the normalized SDK view from literal response evidence.
type QuotaRecord = quotafetch.Record

func WithQuotaReadOpts(v QuotaReadOpts) QuotaReadOption { return quotafetch.WithOptions(v) }
func WithQuotaReadQuery(key string, value any) QuotaReadOption {
	if key == "base_path" {
		return func(*QuotaReadOpts) error {
			return fmt.Errorf("%w: Nova get_quota_set supplies base_path itself", resource.ErrInvalidOption)
		}
	}
	return quotafetch.WithQuery(key, value)
}
func WithQuotaUsage(v bool) QuotaReadOption              { return quotafetch.WithUsage(v) }
func WithQuotaReadMicroversion(v string) QuotaReadOption { return quotafetch.WithMicroversion(v) }
func WithQuotaReadRequiresID(v bool) QuotaReadOption     { return quotafetch.WithRequiresID(v) }
func WithQuotaReadSkipCache(v bool) QuotaReadOption      { return quotafetch.WithSkipCache(v) }
func WithQuotaReadErrorMessage(v string) QuotaReadOption { return quotafetch.WithErrorMessage(v) }
func WithQuotaReadResponseKey(v string) QuotaReadOption  { return quotafetch.WithResponseKey(v) }

var quotaReadProfile = quotafetch.Profile{Kind: "compute quota", Role: "compute", Ceiling: "2.56", NovaDetail: true, IntegerFields: []string{
	"cores", "fixed_ips", "floating_ips", "injected_file_content_bytes", "injected_file_path_bytes", "injected_files", "instances", "key_pairs", "metadata_items", "networks", "ram", "security_group_rules", "security_groups", "server_groups", "server_group_members",
}}

// GetRecord implements the pinned get_quota_set(project, usage=False, **query)
// contract. Default unversioned clients discover a version up to 2.56; explicit
// or client versions remain uncapped. Existing Get/Detail retain native models.
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
	return value, request.Wrap("GetRecord", "compute quota", err)
}

// GetRecord binds user_id to the previously resolved UserQuotaScope. It never
// resolves an identity during fetch or infers project/user from the response.
func (s *UserQuotaScope) GetRecord(ctx context.Context, options ...QuotaReadOption) (*QuotaRecord, error) {
	var client *gophercloud.ServiceClient
	projectID, userID := "", ""
	if s != nil {
		projectID = s.projectID
		userID = s.userID
		if s.api != nil {
			client = s.api.client
		}
	}
	value, err := quotafetch.Read(ctx, client, projectID, userID, quotaReadProfile, options...)
	return value, request.Wrap("GetRecord", "compute quota", err)
}

func WithQuotaReadLocation(v resource.CloudLocation) QuotaReadOption {
	return quotafetch.WithLocation(v)
}
