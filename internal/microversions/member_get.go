package microversions

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

// MemberGet owns a bodyless member request and its selected version.
// It retains accepted read/Close failures without retrying the observation.
func MemberGet(ctx context.Context, source *cloudread.Source, target, version string, profile Profile, codes ...int) (*rest.Response, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if source == nil || profile.HeaderService == "" || profile.LegacyVersionHeader == "" {
		return nil, invalid("member GET needs a source and version-header profile")
	}
	var faults faultsState
	guard := func(ctx context.Context) error {
		return errors.Join(source.Guard(ctx), faults.error(), rest.CheckOperationGuard(ctx))
	}
	client, err := fixedrequest.NewGuarded(&source.Client, http.MethodGet, target, guard)
	if err != nil {
		return nil, err
	}
	client.Microversion = version
	if client.MoreHeaders == nil {
		client.MoreHeaders = make(map[string]string)
	}
	for key := range client.MoreHeaders {
		if versionHeader(key, profile) {
			delete(client.MoreHeaders, key)
		}
	}
	if version != "" {
		client.MoreHeaders["Openstack-Api-Version"] = profile.HeaderService + " " + version
		client.MoreHeaders[profile.LegacyVersionHeader] = version
	}
	parent := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.Body != nil && req.Body != http.NoBody || req.ContentLength != 0 || len(req.TransferEncoding) != 0 {
			faults.add(invalid("version-owned member GET changes physical body framing"))
		}
		if req.Body != nil && req.Body != http.NoBody {
			faults.add(req.Body.Close())
			req.Body = http.NoBody
		}
		for key, values := range req.Header {
			if strings.EqualFold(key, "Transfer-Encoding") || strings.EqualFold(key, "Content-Length") && (len(values) != 1 || values[0] != "0") {
				faults.add(invalid("version-owned member GET changes physical body framing"))
			}
		}
		if err := memberHeaderVersion(req.Header, version, profile); err != nil {
			faults.add(fmt.Errorf("version-owned member GET policy: %w", err))
		}
		if err := guard(req.Context()); err != nil {
			return nil, err
		}
		return parent.RoundTrip(req)
	})
	if retry := client.ProviderClient.RetryFunc; retry != nil {
		client.ProviderClient.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
			callbackErr := retry(ctx, method, target, options, original, count)
			if err := memberRetryVersion(options, version, profile); err != nil {
				faults.add(fmt.Errorf("version-owned member GET retry policy: %w", err))
			}
			if err := guard(ctx); err != nil {
				return errors.Join(original, callbackErr, err)
			}
			return callbackErr
		}
	}
	return rest.DoJSONGuarded(ctx, client, guard, http.MethodGet, target, nil, nil, codes...)
}

func memberHeaderVersion(headers http.Header, version string, profile Profile) error {
	for _, name := range []string{"Openstack-Api-Version", profile.LegacyVersionHeader} {
		count := 0
		for key, values := range headers {
			if !strings.EqualFold(key, name) {
				continue
			}
			count += len(values)
			expected := version
			if name == "Openstack-Api-Version" {
				expected = profile.HeaderService + " " + version
			}
			if version == "" || len(values) != 1 || values[0] != expected {
				return invalid("version-owned member GET changes physical microversion header %q", key)
			}
		}
		if version != "" && count != 1 {
			return invalid("version-owned member GET removes physical microversion header %q", name)
		}
	}
	return nil
}

func memberRetryVersion(options *gophercloud.RequestOpts, version string, profile Profile) error {
	if options == nil {
		return invalid("version-owned member GET retry options are required")
	}
	// Native options retain both service and generated spelling aliases. They
	// collapse to one physical header; reject conflicting values, not aliases.
	for _, name := range []string{"Openstack-Api-Version", profile.LegacyVersionHeader} {
		present := false
		for key, value := range options.MoreHeaders {
			if !strings.EqualFold(key, name) {
				continue
			}
			present = true
			expected := version
			if name == "Openstack-Api-Version" {
				expected = profile.HeaderService + " " + version
			}
			if version == "" || value != expected {
				return invalid("version-owned member GET retry changes microversion header %q", key)
			}
		}
		if version != "" && !present {
			return invalid("version-owned member GET retry removes microversion header %q", name)
		}
	}
	if version != "" {
		for _, key := range options.OmitHeaders {
			if versionHeader(key, profile) {
				return invalid("version-owned member GET retry omits microversion header %q", key)
			}
		}
	}
	return nil
}
