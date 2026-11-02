package quotasets

import (
	"context"
	"encoding/json"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"

	"github.com/JSYoo5B/go-openstacksdk/internal/quotafetch"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// QuotaReadOpts is the concrete Source-shaped quota fetch input. Usage defaults
// to false and is always sent as Requests' False/True query spelling.
type QuotaReadOpts = quotafetch.Options

// QuotaReadOption is one option of GetRecord: WithQuotaReadOpts,
// WithQuotaUsage, WithQuotaReadBasePath, WithQuotaReadRequiresID,
// WithQuotaReadSkipCache, WithQuotaReadErrorMessage, WithQuotaReadResponseKey,
// WithQuotaReadLocation, request.Query (a Source fetch keyword argument sent
// as JSON), request.Microversion or request.Custom over *QuotaReadOpts. Other
// shared calls fail with resource.ErrInvalidOption before HTTP.
//
// WithQuotaReadBasePath stays operation-specific: GetQueryCall, the closest
// shared category, does not admit request.BasePath.
type QuotaReadOption = request.GetQueryCall

var quotaReadCalls = calls.Spec[QuotaReadOpts]{
	Query: func(c *QuotaReadOpts, entry request.CallEntry) error {
		switch entry.Key {
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

// WithQuotaReadBasePath sets the Source fetch base_path. It is
// operation-specific: this read does not accept the shared BasePath call.
func WithQuotaReadBasePath(v string) QuotaReadOption {
	return request.Custom(quotafetch.WithBasePath(v))
}
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
	value, err := quotafetch.Read(ctx, client, projectID, "", quotaReadProfile, quotaReadOptions(options)...)
	return value, request.Wrap("GetRecord", "block storage quota", err)
}

func WithQuotaReadLocation(v resource.CloudLocation) QuotaReadOption {
	return request.Custom(quotafetch.WithLocation(v))
}
