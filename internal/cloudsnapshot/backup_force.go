package cloudsnapshot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
	"gophercloudsdk/internal/rest"
)

// Pinned Backup._action explicitly selects3.64. Ordinary create, lookup and
// wait requests keep the captured selected microversion. Only this action's
// independent client and canonical version headers use the forced policy.
func (p *reader) backupForceDelete(ctx context.Context, target string) (*rest.Response, error) {
	client, err := fixedrequest.NewGuarded(&p.source.Client, http.MethodPost, target, p.source.Guard)
	if err != nil {
		return nil, err
	}
	client.Microversion = "3.64"
	if client.MoreHeaders == nil {
		client.MoreHeaders = make(map[string]string)
	}
	client.MoreHeaders["Openstack-Api-Version"] = "volume 3.64"
	client.MoreHeaders["X-Openstack-Volume-Api-Version"] = "3.64"
	var faults rejectedPollFaults
	guard := func(ctx context.Context) error { return errors.Join(p.source.Guard(ctx), faults.error()) }
	parent := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = rejectedPollTransport(func(req *http.Request) (*http.Response, error) {
		if err := backupActionHeaders(req.Header); err != nil {
			faults.add(err)
		}
		if err := guard(req.Context()); err != nil {
			return nil, err
		}
		return parent.RoundTrip(req)
	})
	if retry := client.ProviderClient.RetryFunc; retry != nil {
		client.ProviderClient.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
			callbackErr := retry(ctx, method, target, options, original, count)
			if options == nil {
				faults.add(invalid("backup force action retry options are required"))
			} else {
				for _, required := range []string{"Openstack-Api-Version", "X-Openstack-Volume-Api-Version"} {
					present := false
					for key := range options.MoreHeaders {
						if strings.EqualFold(key, required) {
							present = true
						}
					}
					if !present {
						faults.add(invalid("backup force action retry removes microversion header %q", required))
					}
				}
				for key, value := range options.MoreHeaders {
					if expected, owned := backupActionVersion(key); owned && value != expected {
						faults.add(invalid("backup force action retry changes microversion header %q", key))
					}
				}
				for _, key := range options.OmitHeaders {
					if _, owned := backupActionVersion(key); owned {
						faults.add(invalid("backup force action retry omits microversion header %q", key))
					}
				}
			}
			if err := guard(ctx); err != nil {
				return errors.Join(original, callbackErr, err)
			}
			return callbackErr
		}
	}
	return rest.DoJSONGuarded(ctx, client, guard, http.MethodPost, target,
		json.RawMessage(`{"os-force_delete":null}`), nil, sourceCodes()...)
}

func backupActionVersion(key string) (string, bool) {
	switch strings.ToLower(key) {
	case "openstack-api-version":
		return "volume 3.64", true
	case "x-openstack-volume-api-version":
		return "3.64", true
	}
	return "", false
}

// Check the actual physical headers too: redirects and native callbacks may
// modify a request after options validation, before an authenticated resend.
func backupActionHeaders(headers http.Header) error {
	for _, required := range []string{"Openstack-Api-Version", "X-Openstack-Volume-Api-Version"} {
		expected, _ := backupActionVersion(required)
		values := []string{}
		for key, candidates := range headers {
			if strings.EqualFold(key, required) {
				values = append(values, candidates...)
			}
		}
		if len(values) != 1 || values[0] != expected {
			return invalid("backup force action requires microversion header %q=%q", required, expected)
		}
	}
	return nil
}
