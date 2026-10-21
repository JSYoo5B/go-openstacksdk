package ironicwire

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// NodeCeiling is the highest microversion the pinned Python Node class uses.
const NodeCeiling = "1.109"

func unsupported(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrUnsupported, fmt.Sprintf(format, args...))
}

// Range reads the version range advertised by the Bare Metal endpoint.
func Range(ctx context.Context, client *gophercloud.ServiceClient) (maximum, minimum string, err error) {
	source, err := cloudread.Capture(ctx, client, "baremetal")
	if err != nil {
		return "", "", err
	}
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = i + 200
	}
	discovery, err := microversions.Read(ctx, source, microversions.Profile{Major: "1", HeaderService: "baremetal", LegacyVersionHeader: "X-OpenStack-Ironic-API-Version", OkCodes: codes, StopFaultRetryCallbacks: true})
	if err != nil {
		return "", "", err
	}
	return discovery.Maximum, discovery.Minimum, nil
}

func compare(a, b string) (int, error) {
	x, err := microversions.ParseVersion(a)
	if err != nil {
		return 0, err
	}
	y, err := microversions.ParseVersion(b)
	if err != nil {
		return 0, err
	}
	return x.Compare(y), nil
}

// Assert mirrors Resource._assert_microversion_for. A configured client
// microversion is the actual version; otherwise the highest version shared by
// the endpoint and ceiling is discovered. The actual version must be at least
// required (when non-empty) and is lowered to maximum (when non-empty). A
// failure wraps resource.ErrUnsupported.
func Assert(ctx context.Context, client *gophercloud.ServiceClient, required, ceiling, maximum string) (string, error) {
	actual := client.Microversion
	if actual == "" {
		high, low, err := Range(ctx, client)
		if err != nil {
			return "", err
		}
		actual, err = microversions.SelectVersion(high, low, ceiling)
		if err != nil {
			return "", err
		}
		if actual == "" {
			return "", unsupported("API version %s is required, but the default version will be used", orNone(required))
		}
	}
	if required != "" {
		c, err := compare(actual, required)
		if err != nil {
			return "", err
		}
		if c < 0 {
			return "", unsupported("API version %s is required, but %s will be used", required, actual)
		}
	}
	if maximum != "" {
		c, err := compare(actual, maximum)
		if err != nil {
			return "", err
		}
		if c > 0 {
			return maximum, nil
		}
	}
	return actual, nil
}

func orNone(v string) string {
	if v == "" {
		return "None"
	}
	return v
}

// Pick mirrors utils.pick_microversion: a configured client microversion that
// matches required is kept, a lower one fails, and an unset one is replaced by
// required. The result must lie in the endpoint range, which is always read.
func Pick(ctx context.Context, client *gophercloud.ServiceClient, required string) (string, error) {
	selected := client.Microversion
	version := required
	if selected != "" {
		ok, err := microversions.Matches(selected, required)
		if err != nil {
			return "", err
		}
		if ok {
			version = selected
		}
	}
	high, low, err := Range(ctx, client)
	if err != nil {
		return "", err
	}
	ok, err := microversions.Supports(high, low, selected, version)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", unsupported("requested microversion %s is not supported by the server side or the default microversion is too low", version)
	}
	return version, nil
}

// At returns a copy of client that sends the given microversion. The copy has
// its own header map; the original is never changed.
func At(client *gophercloud.ServiceClient, version string) *gophercloud.ServiceClient {
	copy := *client
	copy.MoreHeaders = maps.Clone(client.MoreHeaders)
	for key := range copy.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Ironic-API-Version") {
			delete(copy.MoreHeaders, key)
		}
	}
	copy.Microversion = version
	return &copy
}
