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
	auth                   *gophercloud.AuthOptions
	cloud                  string
	cloudFiles             []string
	region                 *string
	availability           *gophercloud.Availability
	httpClient             http.Client
	httpConfigured         bool
	tlsConfig              *tls.Config
	endpoints              map[Service]string
	microversions          map[Service]string
	microversionRanges     map[Service]microversionRange
	microversionSelections map[string]MicroversionSelection
	versionedEndpoints     map[string]string
	messagingClientID      string
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

// WithEndpointFor configures a particular major API version independently.
func WithEndpointFor(service Service, version, endpoint string) ConnectionOption {
	return func(o *connectionOptions) error {
		definition, ok := serviceDefinitions[service]
		if !ok {
			return invalid("unknown service %q", service)
		}
		if _, ok := definition.factories[version]; !ok && !(service == Messaging && version == "v2") {
			return unsupported(string(service), version)
		}
		temporary := connectionOptions{endpoints: make(map[Service]string)}
		if err := WithEndpoint(service, endpoint)(&temporary); err != nil {
			return err
		}
		o.versionedEndpoints[serviceKey(service, version)] = endpoint
		return nil
	}
}

// WithMessagingClientID selects the stable Zaqar client UUID. By default a
// connection creates one UUID and shares it across all messaging operations.
func WithMessagingClientID(id string) ConnectionOption {
	return func(o *connectionOptions) error {
		if !regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`).MatchString(id) {
			return invalid("messaging client ID must be a UUID")
		}
		o.messagingClientID = id
		return nil
	}
}

var microversionPattern = regexp.MustCompile(`^[1-9][0-9]*\.[0-9]+$`)

// WithMicroversion selects an exact version without discovery. It takes
// precedence over WithMicroversionRange and WithLatestMicroversion regardless
// of option order. Extension compatibility remains subject to the cloud API.
func WithMicroversion(service Service, version string) ConnectionOption {
	return func(o *connectionOptions) error {
		major := serviceDefinitions[service].microversionMajor
		if major == "" {
			return unsupported(string(service), "microversions")
		}
		if _, err := parseMicroversion(version); err != nil {
			return invalid("invalid microversion %q", version)
		}
		if !strings.HasPrefix(version, major+".") {
			return invalid("microversion %q does not belong to %s", version, service)
		}
		o.microversions[service] = version
		return nil
	}
}

func validService(service Service) bool {
	_, ok := serviceDefinitions[service]
	return ok
}

func parseConnection(opts []ConnectionOption) (connectionOptions, error) {
	o := connectionOptions{
		endpoints: make(map[Service]string), microversions: make(map[Service]string),
		microversionRanges: make(map[Service]microversionRange), microversionSelections: make(map[string]MicroversionSelection),
		versionedEndpoints: make(map[string]string),
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
