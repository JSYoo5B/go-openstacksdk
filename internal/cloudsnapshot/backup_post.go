package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/fixedrequest"
	"gophercloudsdk/internal/rest"
)

// backupPost owns the exact serialized POST through physical redirects too.
// Forced actions select3.64 on an operation copy; restore keeps selected MV.
func (p *reader) backupPost(ctx context.Context, target string, body json.RawMessage, forceVersion bool) (*rest.Response, error) {
	var version *string
	if forceVersion {
		selected := "3.64"
		version = &selected
	}
	return p.backupPostPolicy(ctx, target, body, version)
}

// An explicit version policy owns both presence and value. Nil preserves the
// restore policy; an empty override suppresses negotiation headers for import.
func (p *reader) backupPostPolicy(ctx context.Context, target string, body json.RawMessage, version *string) (*rest.Response, error) {
	client, err := fixedrequest.NewGuarded(&p.source.Client, http.MethodPost, target, p.source.Guard)
	if err != nil {
		return nil, err
	}
	if version != nil {
		client.Microversion = *version
		if client.MoreHeaders == nil {
			client.MoreHeaders = make(map[string]string)
		}
		for key := range client.MoreHeaders {
			if _, owned := backupActionVersion(key); owned {
				delete(client.MoreHeaders, key)
			}
		}
		if *version != "" {
			client.MoreHeaders["Openstack-Api-Version"] = "volume " + *version
			client.MoreHeaders["X-Openstack-Volume-Api-Version"] = *version
		}
	}
	expected, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	var faults rejectedPollFaults
	guard := func(ctx context.Context) error { return errors.Join(p.source.Guard(ctx), faults.error()) }
	parent := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = rejectedPollTransport(func(req *http.Request) (*http.Response, error) {
		if version != nil {
			faults.add(backupPostHeaderVersion(req.Header, *version))
		}
		faults.add(backupPostBody(req, expected))
		if err := guard(req.Context()); err != nil {
			return nil, err
		}
		return parent.RoundTrip(req)
	})
	if retry := client.ProviderClient.RetryFunc; retry != nil {
		client.ProviderClient.RetryFunc = func(ctx context.Context, method, target string, options *gophercloud.RequestOpts, original error, count uint) error {
			callbackErr := retry(ctx, method, target, options, original, count)
			if version != nil {
				faults.add(backupPostRetryVersion(options, *version))
			}
			if err := guard(ctx); err != nil {
				return errors.Join(original, callbackErr, err)
			}
			return callbackErr
		}
	}
	return rest.DoJSONGuarded(ctx, client, guard, http.MethodPost, target, body, nil, sourceCodes()...)
}

func backupPostBody(req *http.Request, expected []byte) error {
	var policyErr error
	if req.Body == nil || req.Body == http.NoBody || req.ContentLength != int64(len(expected)) || len(req.TransferEncoding) != 0 {
		policyErr = invalid("backup POST changes physical JSON body framing")
	}
	for key, values := range req.Header {
		if strings.EqualFold(key, "Transfer-Encoding") || (strings.EqualFold(key, "Content-Length") && (len(values) != 1 || values[0] != strconv.Itoa(len(expected)))) {
			policyErr = invalid("backup POST changes physical JSON body framing")
		}
	}
	var actual []byte
	var readErr, closeErr error
	if req.Body != nil {
		actual, readErr = io.ReadAll(io.LimitReader(req.Body, int64(len(expected))+1))
		closeErr = req.Body.Close()
		req.Body = http.NoBody
	}
	if !bytes.Equal(actual, expected) {
		policyErr = invalid("backup POST changes physical JSON body")
	}
	if err := errors.Join(policyErr, readErr, closeErr); err != nil {
		return err
	}
	owned := bytes.Clone(expected)
	req.Body = io.NopCloser(bytes.NewReader(owned))
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(owned)), nil }
	return nil
}

func backupPostHeaderVersion(headers http.Header, version string) error {
	for _, name := range []string{"Openstack-Api-Version", "X-Openstack-Volume-Api-Version"} {
		count := 0
		for key, values := range headers {
			if !strings.EqualFold(key, name) {
				continue
			}
			count += len(values)
			expected := version
			if name == "Openstack-Api-Version" {
				expected = "volume " + version
			}
			if version == "" || len(values) != 1 || values[0] != expected {
				return invalid("backup POST changes physical microversion header %q", key)
			}
		}
		if version != "" && count != 1 {
			return invalid("backup POST removes physical microversion header %q", name)
		}
	}
	return nil
}

func backupPostRetryVersion(options *gophercloud.RequestOpts, version string) error {
	if options == nil {
		return invalid("backup POST retry options are required")
	}
	// Native options retain both service and generated spelling aliases. They
	// collapse to one physical header; reject conflicting values, not aliases.
	for _, name := range []string{"Openstack-Api-Version", "X-Openstack-Volume-Api-Version"} {
		present := false
		for key, value := range options.MoreHeaders {
			if !strings.EqualFold(key, name) {
				continue
			}
			present = true
			expected := version
			if name == "Openstack-Api-Version" {
				expected = "volume " + version
			}
			if version == "" || value != expected {
				return invalid("backup POST retry changes microversion header %q", key)
			}
		}
		if version != "" && !present {
			return invalid("backup POST retry removes microversion header %q", name)
		}
	}
	if version != "" {
		for _, key := range options.OmitHeaders {
			if _, owned := backupActionVersion(key); owned {
				return invalid("backup POST retry omits microversion header %q", key)
			}
		}
	}
	return nil
}
