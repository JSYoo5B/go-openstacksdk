package cinderrequest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/internal/fixedrequest"
	"gophercloudsdk/internal/rest"
)

// MemberGet owns the negotiated version and bodyless physical member request.
// It retains accepted read/Close failures without retrying the observation.
func MemberGet(ctx context.Context, source *cloudread.Source, target, version string, codes ...int) (*rest.Response, error) {
	client, err := fixedrequest.NewGuarded(&source.Client, http.MethodGet, target, source.Guard)
	if err != nil {
		return nil, err
	}
	client.Microversion = version
	if client.MoreHeaders == nil {
		client.MoreHeaders = make(map[string]string)
	}
	for key := range client.MoreHeaders {
		if _, owned := versionHeader(key); owned {
			delete(client.MoreHeaders, key)
		}
	}
	if version != "" {
		client.MoreHeaders["Openstack-Api-Version"] = "volume " + version
		client.MoreHeaders["X-Openstack-Volume-Api-Version"] = version
	}
	var faults faultsState
	guard := func(ctx context.Context) error { return errors.Join(source.Guard(ctx), faults.error()) }
	parent := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = transportFunc(func(req *http.Request) (*http.Response, error) {
		if req.Body != nil && req.Body != http.NoBody || req.ContentLength != 0 || len(req.TransferEncoding) != 0 {
			faults.add(invalid("Cinder member GET changes physical body framing"))
		}
		for key, values := range req.Header {
			if strings.EqualFold(key, "Transfer-Encoding") || strings.EqualFold(key, "Content-Length") && (len(values) != 1 || values[0] != "0") {
				faults.add(invalid("Cinder member GET changes physical body framing"))
			}
		}
		if err := postHeaderVersion(req.Header, version); err != nil {
			faults.add(fmt.Errorf("Cinder member GET policy: %w", err))
		}
		if err := guard(req.Context()); err != nil {
			return nil, err
		}
		return parent.RoundTrip(req)
	})
	if retry := client.ProviderClient.RetryFunc; retry != nil {
		client.ProviderClient.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
			callbackErr := retry(ctx, method, target, options, original, count)
			if err := postRetryVersion(options, version); err != nil {
				faults.add(fmt.Errorf("Cinder member GET retry policy: %w", err))
			}
			if err := guard(ctx); err != nil {
				return errors.Join(original, callbackErr, err)
			}
			return callbackErr
		}
	}
	return rest.DoJSONGuarded(ctx, client, guard, http.MethodGet, target, nil, nil, codes...)
}
