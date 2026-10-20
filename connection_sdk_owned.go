package openstack

import (
	"net/url"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
)

func newClusteringV1(provider *gophercloud.ProviderClient, options gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
	return newSDKOwnedV1(provider, options, Clustering)
}

func newInstanceHAV1(provider *gophercloud.ProviderClient, options gophercloud.EndpointOpts) (*gophercloud.ServiceClient, error) {
	return newSDKOwnedV1(provider, options, InstanceHA)
}

// Preserve a catalog's version/project suffix and reverse-proxy prefix. Senlin
// also publishes unversioned catalog endpoints, which need the v1 resource root.
func newSDKOwnedV1(provider *gophercloud.ProviderClient, options gophercloud.EndpointOpts, service Service) (*gophercloud.ServiceClient, error) {
	options.ApplyDefaults(string(service))
	endpoint, err := provider.EndpointLocator(options)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || u.User != nil || u.Opaque != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil, invalid("%s endpoint must be an absolute HTTP(S) URL without credentials, query or fragment", service)
	}
	parts := strings.Split(strings.TrimRight(u.EscapedPath(), "/"), "/")
	index := len(parts) - 1
	if index > 0 && !discoveryPathVersion.MatchString(parts[index]) && discoveryPathVersion.MatchString(parts[index-1]) {
		index--
	}
	if discoveryPathVersion.MatchString(parts[index]) {
		version := parts[index]
		if version != "v1" && !strings.HasPrefix(version, "v1.") {
			return nil, unsupported(string(service), "endpoint version "+version)
		}
	} else {
		u.RawPath = strings.TrimRight(u.EscapedPath(), "/") + "/v1"
		u.Path, err = url.PathUnescape(u.RawPath)
		if err != nil {
			return nil, err
		}
		endpoint = u.String()
	}
	return &gophercloud.ServiceClient{ProviderClient: provider, Type: string(service), Endpoint: gophercloud.NormalizeURL(endpoint)}, nil
}
