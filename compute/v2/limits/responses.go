package limits

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
)

// RateLimit retains a legacy v2 rule. NextAvailable is exact JSON because the
// pinned Python resource does not constrain its timestamp representation.
// Body distinguishes omitted, null and zero values and keeps unknown fields.
type RateLimit struct {
	NextAvailable json.RawMessage            `json:"next-available"`
	Remaining     int                        `json:"remaining"`
	Unit          string                     `json:"unit"`
	Value         int                        `json:"value"`
	Verb          string                     `json:"verb"`
	Body          map[string]json.RawMessage `json:"-"`
}

func (r *RateLimit) UnmarshalJSON(data []byte) error {
	fields, err := objectFields(data, "rate limit rule")
	if err != nil {
		return err
	}
	type plain RateLimit
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*r = RateLimit(value)
	r.Body = fields
	return nil
}

// RateLimits groups legacy rules by URI and regex. Current Nova v2.1 returns
// an empty rate list; these values are preserved when an older server sends them.
type RateLimits struct {
	Limits []RateLimit                `json:"limit"`
	Regex  string                     `json:"regex"`
	URI    string                     `json:"uri"`
	Body   map[string]json.RawMessage `json:"-"`
}

func (r *RateLimits) UnmarshalJSON(data []byte) error {
	fields, err := objectFields(data, "rate limits group")
	if err != nil {
		return err
	}
	type plain RateLimits
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	*r = RateLimits(value)
	r.Body = fields
	return nil
}

// LimitsResource retains native absolute values and the rate fields missing
// from the native model. Body is the complete inner limits object; AbsoluteBody
// preserves extensions and null/omitted values within absolute.
type LimitsResource struct {
	Limits
	ProjectID    string
	Rate         []RateLimits
	Body         map[string]json.RawMessage
	AbsoluteBody map[string]json.RawMessage
	Header       http.Header
	StatusCode   int
}

// LimitsResponseError retains successful HTTP evidence on JSON, read or context
// failures. Unwrap preserves the original error for errors.Is and errors.As.
type LimitsResponseError struct {
	Body       []byte
	Header     http.Header
	StatusCode int
	Cause      error
}

func (e *LimitsResponseError) Error() string {
	return fmt.Sprintf("limits response HTTP %d: %v", e.StatusCode, e.Cause)
}
func (e *LimitsResponseError) Unwrap() error { return e.Cause }

type limitsWire struct {
	body   []byte
	header http.Header
	status int
}

func (w limitsWire) fail(err error) error {
	return &LimitsResponseError{Body: append([]byte(nil), w.body...), Header: w.header.Clone(), StatusCode: w.status, Cause: err}
}

func objectFields(body []byte, expected string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		return nil, gophercloud.ErrUnexpectedType{Expected: expected + " JSON object", Actual: "null"}
	}
	return fields, nil
}

func decodeLimits(wire limitsWire, projectID string) (*LimitsResource, error) {
	envelope, err := objectFields(wire.body, "limits envelope")
	if err != nil {
		return nil, wire.fail(err)
	}
	fields, err := objectFields(envelope["limits"], "limits")
	if err != nil {
		return nil, wire.fail(err)
	}
	var native Limits
	if err := json.Unmarshal(envelope["limits"], &native); err != nil {
		return nil, wire.fail(err)
	}
	var absolute map[string]json.RawMessage
	if data, exists := fields["absolute"]; exists {
		if err := json.Unmarshal(data, &absolute); err != nil {
			return nil, wire.fail(err)
		}
	}
	var rate []RateLimits
	if data, exists := fields["rate"]; exists {
		if err := json.Unmarshal(data, &rate); err != nil {
			return nil, wire.fail(err)
		}
	}
	return &LimitsResource{Limits: native, ProjectID: projectID, Rate: rate, Body: fields, AbsoluteBody: absolute, Header: wire.header.Clone(), StatusCode: wire.status}, nil
}

func (a *API) fetchLimits(ctx context.Context, query url.Values, projectID string) (*LimitsResource, error) {
	endpoint := a.client.ServiceURL("limits")
	if len(query) != 0 {
		endpoint += "?" + query.Encode()
	}
	client, err := fixedrequest.New(a.client, http.MethodGet, endpoint)
	if err != nil {
		return nil, err
	}
	response, err := client.Get(ctx, endpoint, nil, &gophercloud.RequestOpts{OkCodes: []int{http.StatusOK}, KeepResponseBody: true})
	if err != nil {
		return nil, err
	}
	wire := limitsWire{header: response.Header.Clone(), status: response.StatusCode}
	defer response.Body.Close()
	wire.body, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, wire.fail(err)
	}
	if err := ctx.Err(); err != nil {
		return nil, wire.fail(err)
	}
	return decodeLimits(wire, projectID)
}
