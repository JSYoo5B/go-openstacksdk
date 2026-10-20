package request

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// NeutronDeleteOpts selects conditional deletion and the Python proxy's
// default missing-resource behavior. Nil IgnoreMissing means true. Nil Revision
// omits the condition, while an explicit zero sends revision_number=0.
type NeutronDeleteOpts struct {
	Revision      *int64
	IgnoreMissing *bool
}

type NeutronDeleteOption func(*NeutronDeleteOpts) error

func cloneNeutronDeleteOpts(value NeutronDeleteOpts) NeutronDeleteOpts {
	if value.Revision != nil {
		v := *value.Revision
		value.Revision = &v
	}
	if value.IgnoreMissing != nil {
		v := *value.IgnoreMissing
		value.IgnoreMissing = &v
	}
	return value
}

// WithNeutronDeleteOptions snapshots a complete configuration now; applying it
// replaces earlier options and creates new pointers for each invocation.
func WithNeutronDeleteOptions(value NeutronDeleteOpts) NeutronDeleteOption {
	snapshot := cloneNeutronDeleteOpts(value)
	return func(target *NeutronDeleteOpts) error { *target = cloneNeutronDeleteOpts(snapshot); return nil }
}

func WithNeutronDeleteRevision(value int64) NeutronDeleteOption {
	return func(target *NeutronDeleteOpts) error { copy := value; target.Revision = &copy; return nil }
}

func WithNeutronDeleteIgnoreMissing(value bool) NeutronDeleteOption {
	return func(target *NeutronDeleteOpts) error { copy := value; target.IgnoreMissing = &copy; return nil }
}

// DeleteNeutron submits one SDK-owned Neutron DELETE without decoding its body.
// Revision headers are scoped to this invocation; the configured client is not
// changed. Native generated Delete methods retain their strict status policy.
func DeleteNeutron(ctx context.Context, client *gophercloud.ServiceClient, url string, options ...NeutronDeleteOption) error {
	mutable := NeutronDeleteOpts{}
	for _, apply := range append([]NeutronDeleteOption(nil), options...) {
		if apply == nil {
			return fmt.Errorf("%w: nil delete option", resource.ErrInvalidOption)
		}
		if err := apply(&mutable); err != nil {
			if ctx != nil && ctx.Err() != nil {
				return neutronDeleteContextError(ctx, err)
			}
			return err
		}
	}
	config := cloneNeutronDeleteOpts(mutable)
	if ctx == nil || client == nil || client.ProviderClient == nil {
		return fmt.Errorf("%w: delete requires a context and configured client", resource.ErrInvalidOption)
	}
	if err := ctx.Err(); err != nil {
		return neutronDeleteContextError(ctx, nil)
	}
	captured := *client
	captured.MoreHeaders = make(map[string]string, len(client.MoreHeaders))
	for key, value := range client.MoreHeaders {
		if config.Revision == nil || !strings.EqualFold(key, "If-Match") {
			captured.MoreHeaders[key] = value
		}
	}
	headers := map[string]string{}
	if config.Revision != nil {
		headers["If-Match"] = "revision_number=" + strconv.FormatInt(*config.Revision, 10)
	}
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	_, err := rest.DoJSONGuardedRejectionsHeaders(ctx, &captured, nil,
		http.MethodDelete, url, nil, headers,
		rest.RejectionPolicy{Codes: []int{http.StatusNotFound}, PreserveCleanRetry: true}, codes...)
	if ctx.Err() != nil {
		return neutronDeleteContextError(ctx, err)
	}
	if config.IgnoreMissing == nil || *config.IgnoreMissing {
		return resource.IgnoreMissing(err)
	}
	return err
}

func neutronDeleteContextError(ctx context.Context, err error) error {
	canceled := ctx.Err()
	cause := context.Cause(ctx)
	if cause == canceled {
		cause = nil
	}
	return errors.Join(err, canceled, cause)
}
