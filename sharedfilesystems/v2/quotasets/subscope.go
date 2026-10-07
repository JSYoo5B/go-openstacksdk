package quotasets

import (
	"context"
	"fmt"
	"net/url"

	"github.com/JSYoo5B/gophercloudsdk/resource"
)

// quotaBinding is private composition, never an embedded project scope. There
// is no project-wide reset/defaults method to promote on a user or share type.
type quotaBinding struct {
	api       *API
	projectID string
	selector  string
	id        string
}

func (b quotaBinding) validate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := b.api.validate(ctx); err != nil {
		return err
	}
	if err := resource.ID(b.projectID).Validate(); err != nil {
		return err
	}
	if err := resource.ID(b.id).Validate(); err != nil {
		return err
	}
	switch b.selector {
	case "user_id":
		_, err := b.api.minorVersion()
		return err
	case "share_type":
		return b.api.requireVersion(39)
	default:
		return fmt.Errorf("%w: invalid quota scope selector", resource.ErrInvalidOption)
	}
}

func (b quotaBinding) endpoint(suffix string) (string, error) {
	endpoint, err := b.api.quotaURL(b.projectID, suffix)
	if err != nil {
		return "", err
	}
	query := url.Values{b.selector: []string{b.id}}
	return endpoint + "?" + query.Encode(), nil
}

func (b quotaBinding) wrap(operation string, err error) error {
	return quotaError(operation, b.projectID+"/"+b.id, err)
}

func (b quotaBinding) get(ctx context.Context) (*QuotaResource, error) {
	if err := b.validate(ctx); err != nil {
		return nil, b.wrap("Get", err)
	}
	endpoint, err := b.endpoint("")
	if err != nil {
		return nil, b.wrap("Get", err)
	}
	result, err := b.api.read(ctx, endpoint)
	if err != nil {
		return nil, b.wrap("Get", err)
	}
	value, err := decodeQuota(result, b.projectID)
	if value != nil {
		if b.selector == "user_id" {
			value.UserID = b.id
		} else {
			value.ShareTypeID = b.id
		}
	}
	return value, b.wrap("Get", err)
}

func (b quotaBinding) detail(ctx context.Context) (*QuotaDetailResource, error) {
	if err := b.validate(ctx); err != nil {
		return nil, b.wrap("Detail", err)
	}
	if err := b.api.requireVersion(25); err != nil {
		return nil, b.wrap("Detail", err)
	}
	endpoint, err := b.endpoint("detail")
	if err != nil {
		return nil, b.wrap("Detail", err)
	}
	result, err := b.api.read(ctx, endpoint)
	if err != nil {
		return nil, b.wrap("Detail", err)
	}
	value, err := decodeDetail(result, b.projectID)
	if value != nil {
		if b.selector == "user_id" {
			value.UserID = b.id
		} else {
			value.ShareTypeID = b.id
		}
	}
	return value, b.wrap("Detail", err)
}

func (b quotaBinding) update(ctx context.Context, opts UpdateOpts, options ...UpdateOption) (*QuotaResource, error) {
	if err := b.validate(ctx); err != nil {
		return nil, b.wrap("Update", err)
	}
	body, err := prepareUpdate(opts, options...)
	if err != nil {
		return nil, b.wrap("Update", err)
	}
	if b.selector == "share_type" {
		if _, present := body["quota_set"].(map[string]any)["share_networks"]; present {
			return nil, b.wrap("Update", fmt.Errorf("%w: share_networks is not a share-type quota", resource.ErrUnsupported))
		}
	}
	endpoint, err := b.endpoint("")
	if err != nil {
		return nil, b.wrap("Update", err)
	}
	result, err := b.api.update(ctx, endpoint, body)
	if err != nil {
		return nil, b.wrap("Update", err)
	}
	value, err := decodeQuota(result, b.projectID)
	if value != nil {
		if b.selector == "user_id" {
			value.UserID = b.id
		} else {
			value.ShareTypeID = b.id
		}
	}
	return value, b.wrap("Update", err)
}

func (b quotaBinding) reset(ctx context.Context, options ...ResetOption) (*ResetResponse, error) {
	if err := b.validate(ctx); err != nil {
		return nil, b.wrap("Reset", err)
	}
	config, err := prepareReset(options...)
	if err != nil {
		return nil, b.wrap("Reset", err)
	}
	endpoint, err := b.endpoint("")
	if err != nil {
		return nil, b.wrap("Reset", err)
	}
	value, err := b.api.reset(ctx, endpoint, config.ignoreMissing)
	if value != nil {
		value.ProjectID = b.projectID
		if b.selector == "user_id" {
			value.UserID = b.id
		} else {
			value.ShareTypeID = b.id
		}
	}
	return value, b.wrap("Reset", err)
}
