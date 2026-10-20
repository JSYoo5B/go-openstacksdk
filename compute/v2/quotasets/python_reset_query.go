package quotasets

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// WithResetQuery adds one revert_quota_set(**query) parameter to the reset
// DELETE. Python's Resource.delete encodes params with urlencode, which sends
// str(value), so the value is the literal text the server receives. A later
// call for the same key wins. user_id belongs to UserQuotaScope and is
// rejected on both scopes.
func WithResetQuery(key, value string) ResetOption {
	return func(options *resetOptions) error {
		if strings.TrimSpace(key) == "" || !utf8.ValidString(key) || strings.IndexFunc(key, unicode.IsControl) >= 0 || !utf8.ValidString(value) {
			return fmt.Errorf("%w: reset query needs a nonblank text key and valid UTF-8 value", resource.ErrInvalidOption)
		}
		if key == "user_id" {
			return fmt.Errorf("%w: reset user_id is selected with ProjectQuotaScope.InUser", resource.ErrInvalidOption)
		}
		if options.query == nil {
			options.query = make(url.Values)
		}
		options.query.Set(key, value)
		return nil
	}
}

// resetQueryURL adds the reset parameters next to an existing user_id.
func resetQueryURL(endpoint string, query url.Values) string {
	if len(query) == 0 {
		return endpoint
	}
	base, raw, _ := strings.Cut(endpoint, "?")
	values, _ := url.ParseQuery(raw)
	for key, value := range query {
		values[key] = append([]string(nil), value...)
	}
	return base + "?" + values.Encode()
}

// resetWithQuery sends the project reset DELETE with WithResetQuery
// parameters through the same fixed-target guard as the user scope. The
// native 202 and 204 success codes are kept.
func (a *API) resetWithQuery(ctx context.Context, projectID, userID string, config resetOptions) (*ResetResponse, error) {
	endpoint := resetQueryURL(a.quotaURL(projectID, userID, ""), config.query)
	client, err := guardedQuotaClient(a.client, http.MethodDelete, endpoint)
	if err != nil {
		return nil, quotaError("Reset", projectID, err)
	}
	response, err := client.Delete(ctx, endpoint, &gophercloud.RequestOpts{OkCodes: []int{http.StatusAccepted, http.StatusNoContent}})
	_, header, err := gophercloud.ParseResponse(response, err)
	if config.ignoreMissing && gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, quotaError("Reset", projectID, err)
	}
	return &ResetResponse{ProjectID: projectID, UserID: userID, Header: header.Clone()}, nil
}
