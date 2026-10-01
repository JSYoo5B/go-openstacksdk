// Package nativefind preserves native result extraction for audited SDK-owned
// identity getters that need a per-call service query.
package nativefind

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"unicode"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/resource"
)

// Get uses the original service/provider clients and native request machinery.
// Segments are literal, unescaped member-route parts. The caller supplies the
// audited getter's accepted codes and its native result Body destination.
// Query and code slices are copied before HTTP; no client setting is modified.
func Get(ctx context.Context, client *gophercloud.ServiceClient, segments []string, query url.Values, okCodes []int, body any) (http.Header, error) {
	invalid := func(message string) (http.Header, error) {
		return nil, fmt.Errorf("%w: %s", resource.ErrInvalidOption, message)
	}
	if ctx == nil {
		return invalid("native identity GET requires a context")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if client == nil || client.ProviderClient == nil {
		return invalid("native identity GET requires a service and provider client")
	}
	if len(segments) == 0 || len(okCodes) == 0 {
		return invalid("native identity GET requires route segments and accepted codes")
	}
	parts := append([]string(nil), segments...)
	for _, part := range parts {
		if !utf8.ValidString(part) || resource.ID(part).Validate() != nil {
			return invalid("native identity GET requires literal single path segments")
		}
		for _, char := range part {
			if unicode.IsControl(char) || unicode.IsSpace(char) {
				return invalid("native identity GET path segments cannot contain whitespace or controls")
			}
		}
	}
	accepted := append([]int(nil), okCodes...)
	for _, code := range accepted {
		if code < 100 || code > 599 {
			return invalid("native identity GET requires valid accepted HTTP codes")
		}
	}
	frozen := make(url.Values, len(query))
	for key, values := range query {
		frozen[key] = append([]string(nil), values...)
	}
	target, err := url.Parse(client.ServiceURL(parts...))
	if err != nil || target == nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" || target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return invalid("native identity GET requires an absolute service URL without credentials, query or fragment")
	}
	target.RawQuery = frozen.Encode()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	response, err := client.Get(ctx, target.String(), body, &gophercloud.RequestOpts{OkCodes: accepted})
	_, header, err := gophercloud.ParseResponse(response, err)
	return header, err
}
