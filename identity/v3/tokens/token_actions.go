package tokens

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func tokenActionCodes(first, last int) []int {
	codes := make([]int, last-first+1)
	for i := range codes {
		codes[i] = first + i
	}
	return codes
}

func (a *API) captureTokenAction(ctx context.Context, token string) (*cloudread.Source, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if a == nil {
		return nil, fmt.Errorf("%w: token API is required", resource.ErrInvalidOption)
	}
	// A token is a header value, never a route identity. Empty values have the
	// same wire representation as Python's empty string; CR/LF are forbidden.
	if !utf8.ValidString(token) || strings.ContainsAny(token, "\r\n\x00") {
		return nil, fmt.Errorf("%w: invalid subject-token header", resource.ErrInvalidOption)
	}
	p, err := cloudread.Capture(ctx, a.client, "identity")
	if err != nil {
		return nil, err
	}
	// Operation headers take precedence over configured casing aliases without
	// altering the shared ServiceClient. Native retry hooks cannot retarget it.
	for key := range p.Client.MoreHeaders {
		if strings.EqualFold(key, "X-Subject-Token") {
			delete(p.Client.MoreHeaders, key)
		}
	}
	return p, nil
}

func tokenActionTarget(p *cloudread.Source, noCatalog, allowExpired bool) string {
	target := p.Client.ResourceBaseURL() + "auth/tokens"
	query := make(url.Values)
	if noCatalog {
		query.Set("nocatalog", "True")
	}
	if allowExpired {
		query.Set("allow_expired", "True")
	}
	if len(query) != 0 {
		target += "?" + query.Encode()
	}
	return target
}

// ValidateToken is Python validate_token: GET with an independent subject
// token, optional true-only query arguments, and an owned raw Token view.
// Existing Get retains Gophercloud's native Token and timestamp semantics.
func (a *API) ValidateToken(ctx context.Context, token string, options ...ValidateTokenOption) (*TokenRecord, error) {
	const operation = "ValidateToken"
	owned := slices.Clone(options)
	p, err := a.captureTokenAction(ctx, token)
	if err != nil {
		return nil, request.Wrap(operation, "tokens", err)
	}
	callbacks := make([]func(*ValidateTokenOpts) error, len(owned))
	for i, option := range owned {
		callbacks[i] = option
	}
	policy, err := prepareTokenOptions(ctx, p.Guard, callbacks)
	if err != nil {
		return nil, request.Wrap(operation, "tokens", err)
	}
	response, err := rest.DoJSONGuardedHeaders(ctx, &p.Client, p.Guard, http.MethodGet,
		tokenActionTarget(p, policy.NoCatalog, policy.AllowExpired), nil,
		map[string]string{"X-Subject-Token": token}, tokenActionCodes(200, 399)...)
	if err != nil {
		return tokenRecordReceipt(response), request.Wrap(operation, "tokens", err)
	}
	value, err := translateTokenRecord(response)
	if err = errors.Join(err, p.Guard(ctx)); err != nil {
		return value, request.Wrap(operation, "tokens", response.Fail(err))
	}
	return value, nil
}

// CheckToken returns true only for an actual HEAD 200. Every other HTTP status
// returns false without an HTTP status error; transport and context errors are
// still errors. This includes forbidden, missing and server-error responses.
func (a *API) CheckToken(ctx context.Context, token string, options ...CheckTokenOption) (bool, error) {
	const operation = "CheckToken"
	owned := slices.Clone(options)
	p, err := a.captureTokenAction(ctx, token)
	if err != nil {
		return false, request.Wrap(operation, "tokens", err)
	}
	callbacks := make([]func(*CheckTokenOpts) error, len(owned))
	for i, option := range owned {
		callbacks[i] = option
	}
	policy, err := prepareTokenOptions(ctx, p.Guard, callbacks)
	if err != nil {
		return false, request.Wrap(operation, "tokens", err)
	}
	response, err := rest.DoJSONGuardedHeaders(ctx, &p.Client, p.Guard, http.MethodHead,
		tokenActionTarget(p, false, policy.AllowExpired), nil,
		map[string]string{"X-Subject-Token": token}, tokenActionCodes(100, 599)...)
	if err != nil {
		return false, request.Wrap(operation, "tokens", err)
	}
	return response.StatusCode == http.StatusOK, nil
}

// RevokeToken accepts any actual 200..399 response and does not decode its
// body. Native Revoke remains available with its unchanged native result.
func (a *API) RevokeToken(ctx context.Context, token string) error {
	p, err := a.captureTokenAction(ctx, token)
	if err == nil {
		_, err = rest.DoJSONGuardedHeaders(ctx, &p.Client, p.Guard, http.MethodDelete,
			tokenActionTarget(p, false, false), nil,
			map[string]string{"X-Subject-Token": token}, tokenActionCodes(200, 399)...)
	}
	return request.Wrap("RevokeToken", "tokens", err)
}
