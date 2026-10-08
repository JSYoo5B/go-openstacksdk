package quotas

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/gophercloud/gophercloud/v2"
)

// QuotaResource preserves response fields separately from the request identity.
// Body keeps extension values, nulls and exact JSON numbers without float64.
type QuotaResource struct {
	Quotas
	RequestProjectID string
	RequestResource  ResourceName
	Body             map[string]json.RawMessage
	Header           http.Header
	StatusCode       int
}

// QuotaResponseError retains successful HTTP evidence on decoding/read failure.
type QuotaResponseError struct {
	Body       []byte
	Header     http.Header
	StatusCode int
	Cause      error
}

func (e *QuotaResponseError) Error() string {
	return fmt.Sprintf("quota response HTTP %d: %v", e.StatusCode, e.Cause)
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

func (a *API) requestQuota(ctx context.Context, method, endpoint string, body json.RawMessage, accepted []int) (quotaWire, error) {
	client, err := fixedrequest.New(a.client, method, endpoint)
	if err != nil {
		return quotaWire{}, err
	}
	options := &gophercloud.RequestOpts{OkCodes: accepted, KeepResponseBody: true}
	if body != nil {
		options.JSONBody = body
	}
	response, err := client.Request(ctx, method, endpoint, options)
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

func decodeQuota(wire quotaWire, projectID string, name ResourceName) (*QuotaResource, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(wire.body, &fields); err != nil {
		return nil, wire.fail(err)
	}
	if fields == nil {
		return nil, wire.fail(gophercloud.ErrUnexpectedType{Expected: "quota JSON object", Actual: "null"})
	}
	if limit, exists := fields["hard_limit"]; !exists || string(limit) == "null" {
		return nil, wire.fail(fmt.Errorf("quota response lacks an integer hard_limit"))
	}
	var quota Quotas
	if err := json.Unmarshal(wire.body, &quota); err != nil {
		return nil, wire.fail(err)
	}
	if raw, exists := fields["id"]; exists {
		// Native Quotas.UnmarshalJSON uses float64 for the row ID. Re-read its
		// original token instead of deriving identity from a rounded float.
		if string(raw) == "null" {
			quota.ID = ""
		} else if strings.HasPrefix(string(raw), "\"") {
			if err := json.Unmarshal(raw, &quota.ID); err != nil {
				return nil, wire.fail(err)
			}
		} else {
			integer, ok := new(big.Int).SetString(string(raw), 10)
			if !ok || integer.Sign() <= 0 {
				return nil, wire.fail(fmt.Errorf("quota row id must be a positive integer or string"))
			}
			quota.ID = string(raw)
		}
	}
	return &QuotaResource{Quotas: quota, RequestProjectID: projectID, RequestResource: name, Body: fields, Header: wire.header.Clone(), StatusCode: wire.status}, nil
}
