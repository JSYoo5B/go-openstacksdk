package gophercloudsdk

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
)

type Service string

const (
	Compute      Service = "compute"
	Network      Service = "network"
	Image        Service = "image"
	BlockStorage Service = "block-storage"
)

type connectionOptions struct {
	auth           *gophercloud.AuthOptions
	cloud          string
	cloudFiles     []string
	region         *string
	availability   *gophercloud.Availability
	httpClient     http.Client
	httpConfigured bool
	tlsConfig      *tls.Config
	endpoints      map[Service]string
	microversions  map[Service]string
}

type ConnectionOption func(*connectionOptions) error

// WithAuth uses explicit credentials instead of environment or clouds.yaml.
func WithAuth(auth gophercloud.AuthOptions) ConnectionOption {
	return func(o *connectionOptions) error { o.auth = &auth; o.cloud = ""; return nil }
}

// WithCloud selects a clouds.yaml entry instead of environment authentication.
func WithCloud(name string) ConnectionOption {
	return func(o *connectionOptions) error {
		if strings.TrimSpace(name) == "" {
			return invalid("cloud name must not be empty")
		}
		o.cloud = name
		o.auth = nil
		return nil
	}
}

// WithCloudFiles replaces the default clouds.yaml search paths.
func WithCloudFiles(paths ...string) ConnectionOption {
	paths = append([]string(nil), paths...)
	return func(o *connectionOptions) error {
		if len(paths) == 0 {
			return invalid("at least one cloud file is required")
		}
		for _, p := range paths {
			if p == "" {
				return invalid("cloud file path must not be empty")
			}
		}
		o.cloudFiles = append([]string(nil), paths...)
		return nil
	}
}

func WithRegion(region string) ConnectionOption {
	return func(o *connectionOptions) error { o.region = &region; return nil }
}

func WithInterface(availability gophercloud.Availability) ConnectionOption {
	return func(o *connectionOptions) error {
		switch availability {
		case gophercloud.AvailabilityPublic, gophercloud.AvailabilityInternal, gophercloud.AvailabilityAdmin:
		default:
			return invalid("invalid endpoint interface %q", availability)
		}
		o.availability = &availability
		return nil
	}
}

// WithHTTPClient configures authentication and subsequent API transport.
// TLS settings from clouds.yaml are applied to a cloned *http.Transport.
func WithHTTPClient(client http.Client) ConnectionOption {
	return func(o *connectionOptions) error { o.httpClient = client; o.httpConfigured = true; return nil }
}

// WithEndpoint overrides catalog lookup. Supply a versioned service endpoint,
// except for Neutron, whose base endpoint is followed by /v2.0/ by Gophercloud.
func WithEndpoint(service Service, endpoint string) ConnectionOption {
	return func(o *connectionOptions) error {
		if !validService(service) {
			return invalid("unknown service %q", service)
		}
		u, err := url.Parse(endpoint)
		if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return invalid("endpoint must be an absolute HTTP(S) URL without credentials, query or fragment")
		}
		o.endpoints[service] = endpoint
		return nil
	}
}

var microversionPattern = regexp.MustCompile(`^[1-9][0-9]*\.[0-9]+$`)

// WithMicroversion selects a version explicitly. Automatic negotiation is not
// implemented yet; extension compatibility remains subject to the cloud API.
func WithMicroversion(service Service, version string) ConnectionOption {
	return func(o *connectionOptions) error {
		if service != Compute && service != BlockStorage {
			return unsupported(string(service), "microversions")
		}
		if !microversionPattern.MatchString(version) {
			return invalid("invalid microversion %q", version)
		}
		major := "2."
		if service == BlockStorage {
			major = "3."
		}
		if !strings.HasPrefix(version, major) {
			return invalid("microversion %q does not belong to %s", version, service)
		}
		o.microversions[service] = version
		return nil
	}
}

func validService(service Service) bool {
	return service == Compute || service == Network || service == Image || service == BlockStorage
}

func parseConnection(opts []ConnectionOption) (connectionOptions, error) {
	o := connectionOptions{
		endpoints: make(map[Service]string), microversions: make(map[Service]string),
	}
	for _, apply := range opts {
		if apply == nil {
			return o, invalid("nil connection option")
		}
		if err := apply(&o); err != nil {
			return o, err
		}
	}
	return o, nil
}
