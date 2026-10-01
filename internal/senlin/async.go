package senlin

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/resource"
)

// LocationValues includes case variants supplied by custom HTTP transports.
// The returned slice is independent of the response header.
func LocationValues(headers http.Header) []string {
	var locations []string
	for key, values := range headers {
		if strings.EqualFold(key, "Location") {
			locations = append(locations, values...)
		}
	}
	return locations
}

// ActionID extracts a Location reference without following it. The reference
// must identify one action in the selected service's collection. A malformed
// accepted mutation preserves its response evidence and must not be resent.
func ActionID(client *gophercloud.ServiceClient, response *rest.Response) (string, error) {
	fail := func(reason string) (string, error) {
		return "", response.Fail(fmt.Errorf("%w: Senlin action Location %s", resource.ErrInvalidOption, reason))
	}
	if response == nil {
		return fail("requires an HTTP response")
	}
	locations := LocationValues(response.Header)
	if len(locations) != 1 || strings.TrimSpace(locations[0]) == "" {
		return fail("must contain exactly one nonempty value")
	}
	if client == nil || client.ProviderClient == nil {
		return fail("requires the source service client")
	}
	location, err := url.Parse(locations[0])
	if err != nil || location.User != nil || location.Opaque != "" || location.RawQuery != "" || location.ForceQuery || strings.Contains(locations[0], "#") {
		return fail("must be an action URI without query, userinfo or fragment")
	}
	for _, part := range strings.Split(location.EscapedPath(), "/") {
		decoded, err := url.PathUnescape(part)
		if err != nil || decoded == "." || decoded == ".." {
			return fail("must not contain dot path segments")
		}
	}
	base, err := url.Parse(client.ServiceURL())
	if err != nil {
		return fail("has an invalid source endpoint")
	}
	target := base.ResolveReference(location)
	if err := rest.ValidateTarget(client, target.String()); err != nil {
		return "", response.Fail(err)
	}
	collection, err := url.Parse(client.ServiceURL("actions"))
	if err != nil {
		return fail("has an invalid action collection")
	}
	prefix := strings.TrimSuffix(collection.EscapedPath(), "/") + "/"
	if !strings.HasPrefix(target.EscapedPath(), prefix) {
		return fail("does not belong to the selected action collection")
	}
	escapedID := strings.TrimPrefix(target.EscapedPath(), prefix)
	if escapedID == "" || strings.Contains(escapedID, "/") {
		return fail("must identify a single action")
	}
	id, err := url.PathUnescape(escapedID)
	if err != nil {
		return fail("has an invalid escaped action ID")
	}
	if err := Identifier(id); err != nil {
		return "", response.Fail(err)
	}
	return id, nil
}
