package quotasets

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/internal/quotafetch"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// QuotaReadOpts is the concrete Source-shaped quota fetch input. Usage chooses
// Nova's detail endpoint; Query accepts arbitrary JSON-domain fetch parameters.
type QuotaReadOpts = quotafetch.Options

// QuotaReadOption is one option of the project and user GetRecord:
// WithQuotaReadOpts, WithQuotaUsage, WithQuotaReadRequiresID,
// WithQuotaReadSkipCache, WithQuotaReadErrorMessage, WithQuotaReadResponseKey,
// WithQuotaReadLocation, request.Query (a Source fetch keyword argument sent as
// JSON; base_path is supplied by Nova get_quota_set itself and rejected),
// request.Microversion or request.Custom over *QuotaReadOpts. Other shared
// calls fail with resource.ErrInvalidOption before HTTP.
type QuotaReadOption = request.GetQueryCall

var quotaReadCalls = calls.Spec[QuotaReadOpts]{
	Query: func(c *QuotaReadOpts, entry request.CallEntry) error {
		switch entry.Key {
		case "base_path":
			return fmt.Errorf("%w: Nova get_quota_set supplies base_path itself", resource.ErrInvalidOption)
		case "self", "session", "project":
			// quotafetch rejects bound arguments by presence, before the value.
			return quotafetch.WithQuery(entry.Key, nil)(c)
		}
		if entry.JSONErr != nil {
			return entry.JSONErr
		}
		return quotafetch.WithQuery(entry.Key, json.RawMessage(entry.JSON))(c)
	},
	Microversion: func(c *QuotaReadOpts, version string) error { return quotafetch.WithMicroversion(version)(c) },
}

// quotaReadOptions translates the calls for quotafetch.Read; a nil call stays
// nil so quotafetch reports it at its position.
func quotaReadOptions(options []QuotaReadOption) []quotafetch.Option {
	translated := calls.Translate(options, quotaReadCalls)
	out := make([]quotafetch.Option, len(translated))
	for i, apply := range translated {
		if apply != nil {
			out[i] = apply
		}
	}
	return out
}

// QuotaRecord separates the normalized SDK view from literal response evidence.
type QuotaRecord = quotafetch.Record

func WithQuotaReadOpts(v QuotaReadOpts) QuotaReadOption {
	return request.Custom(quotafetch.WithOptions(v))
}
func WithQuotaUsage(v bool) QuotaReadOption { return request.Custom(quotafetch.WithUsage(v)) }
func WithQuotaReadRequiresID(v bool) QuotaReadOption {
	return request.Custom(quotafetch.WithRequiresID(v))
}
func WithQuotaReadSkipCache(v bool) QuotaReadOption {
	return request.Custom(quotafetch.WithSkipCache(v))
}
func WithQuotaReadErrorMessage(v string) QuotaReadOption {
	return request.Custom(quotafetch.WithErrorMessage(v))
}
func WithQuotaReadResponseKey(v string) QuotaReadOption {
	return request.Custom(quotafetch.WithResponseKey(v))
}

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
	value, err := quotafetch.Read(ctx, client, projectID, "", quotaReadProfile, quotaReadOptions(options)...)
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
	value, err := quotafetch.Read(ctx, client, projectID, userID, quotaReadProfile, quotaReadOptions(options)...)
	return value, request.Wrap("GetRecord", "compute quota", err)
}

func WithQuotaReadLocation(v resource.CloudLocation) QuotaReadOption {
	return request.Custom(quotafetch.WithLocation(v))
}
