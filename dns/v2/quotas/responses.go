package quotas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/gophercloud/gophercloud/v2"
)

// QuotaResource preserves native values and all response fields. Native ints
// represent null as zero; Body distinguishes null, omitted and numeric values.
// ProjectID is always the fixed request target, not a response identity.
type QuotaResource struct {
	Quota
	ProjectID  string
	Body       map[string]json.RawMessage
	Header     http.Header
	StatusCode int
}

// QuotaResponseError retains HTTP evidence for successful responses that cannot
// be decoded. It unwraps to the original JSON, read or context error.
type QuotaResponseError struct {
	Body       []byte
	Header     http.Header
	StatusCode int
	Cause      error
}

func (e *QuotaResponseError) Error() string {
	return fmt.Sprintf("DNS quota response HTTP %d: %v", e.StatusCode, e.Cause)
}
func (e *QuotaResponseError) Unwrap() error { return e.Cause }

type quotaWire struct {
	body   []byte
	header http.Header
	status int
}

func (w quotaWire) fail(err error) error {
	return &QuotaResponseError{Body: append([]byte(nil), w.body...), Header: w.header.Clone(), StatusCode: w.status, Cause: err}
}

// requestQuota follows the Python resource's sudo-project header while keeping
// source headers/configuration and provider/token locks owned by the caller.
func (s *ProjectQuotaScope) requestQuota(ctx context.Context, method string, body json.RawMessage, accepted []int) (quotaWire, error) {
	source := *s.api.client
	source.MoreHeaders = maps.Clone(source.MoreHeaders)
	if source.MoreHeaders == nil {
		source.MoreHeaders = make(map[string]string)
	}
	setHeader := func(name, value string) {
		for key := range source.MoreHeaders {
			if strings.EqualFold(key, name) {
				delete(source.MoreHeaders, key)
			}
		}
		source.MoreHeaders[name] = value
	}
	setHeader("X-Auth-Sudo-Project-ID", s.projectID)
	if s.allProjects != nil {
		setHeader("X-Auth-All-Projects", strconv.FormatBool(*s.allProjects))
	}
	endpoint := source.ServiceURL("quotas", s.projectID)
	client, err := fixedrequest.New(&source, method, endpoint)
	if err != nil {
		return quotaWire{}, err
	}
	opts := &gophercloud.RequestOpts{OkCodes: accepted, KeepResponseBody: true}
	if body != nil {
		opts.JSONBody = body
	}
	response, err := client.Request(ctx, method, endpoint, opts)
	if err != nil {
		return quotaWire{}, err
	}
	wire := quotaWire{header: response.Header.Clone(), status: response.StatusCode}
	defer response.Body.Close()
	wire.body, err = io.ReadAll(response.Body)
	if err != nil {
		return wire, wire.fail(err)
	}
	if err := ctx.Err(); err != nil {
		return wire, wire.fail(err)
	}
	return wire, nil
}

func decodeQuota(wire quotaWire, projectID string) (*QuotaResource, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(wire.body, &fields); err != nil {
		return nil, wire.fail(err)
	}
	if fields == nil {
		return nil, wire.fail(gophercloud.ErrUnexpectedType{Expected: "quota JSON object", Actual: "null"})
	}
	var quota Quota
	if err := json.Unmarshal(wire.body, &quota); err != nil {
		return nil, wire.fail(err)
	}
	return &QuotaResource{Quota: quota, ProjectID: projectID, Body: fields, Header: wire.header.Clone(), StatusCode: wire.status}, nil
}

// Get fetches the quota singleton for this fixed project.
func (s *ProjectQuotaScope) Get(ctx context.Context) (*QuotaResource, error) {
	if err := s.validate(ctx); err != nil {
		return nil, quotaError("Get", s.ProjectID(), err)
	}
	wire, err := s.requestQuota(ctx, http.MethodGet, nil, []int{http.StatusOK})
	if err != nil {
		return nil, quotaError("Get", s.projectID, err)
	}
	value, err := decodeQuota(wire, s.projectID)
	return value, quotaError("Get", s.projectID, err)
}
