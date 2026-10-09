package rest

import (
	"context"
	"errors"
	"io"
	"net/http"
	"slices"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/internal/fixedrequest"
	"github.com/gophercloud/gophercloud/v2"
)

// RejectedResponseFaults retains handling failures which native Gophercloud
// discards while constructing an unexpected-status error. Each operation owns
// its state; selected statuses remain native rejections, never admitted pages.
type RejectedResponseFaults struct {
	mu    sync.Mutex
	cause error
}

func (f *RejectedResponseFaults) Add(err error) {
	if err == nil {
		return
	}
	f.mu.Lock()
	f.cause = errors.Join(f.cause, err)
	f.mu.Unlock()
}
func (f *RejectedResponseFaults) Err() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cause
}

// RejectionPolicy selects rejected bodies to observe. PreserveCleanRetry also
// allows a retry hook returning the identical native rejection to leave that
// rejection eligible for lookup fallback. Existing workflows omit this opt-in.
type RejectionPolicy struct {
	Codes              []int
	PreserveCleanRetry bool
}

// DoJSONGuardedRejections observes Read/Close and source failures only for the
// specified rejected statuses. It preserves native retry and authentication
// policy, while a faulty rejection cannot establish absence or allow fallback.
func DoJSONGuardedRejections(ctx context.Context, source *gophercloud.ServiceClient, sourceGuard func(context.Context) error, method, endpoint string, body any, headers map[string]string, policy RejectionPolicy, codes ...int) (*Response, error) {
	return doJSONGuardedRejections(ctx, source, sourceGuard, method, endpoint, body, headers, policy, false, nil, codes...)
}

// DoJSONGuardedRejectionsHeaders additionally fixes SDK-owned request headers
// across retries and physical attempts while observing selected rejection faults.
func DoJSONGuardedRejectionsHeaders(ctx context.Context, source *gophercloud.ServiceClient, sourceGuard func(context.Context) error, method, endpoint string, body any, headers map[string]string, policy RejectionPolicy, codes ...int) (*Response, error) {
	return doJSONGuardedRejections(ctx, source, sourceGuard, method, endpoint, body, headers, policy, true, nil, codes...)
}

// DoJSONGuardedRejectionsHeaderPolicy also protects required header absence
// without introducing an empty header. Rejected response faults and clean
// native retries retain the same policy as DoJSONGuardedRejectionsHeaders.
func DoJSONGuardedRejectionsHeaderPolicy(ctx context.Context, source *gophercloud.ServiceClient, sourceGuard func(context.Context) error, method, endpoint string, body any, headers RequestHeaderPolicy, policy RejectionPolicy, codes ...int) (*Response, error) {
	return doJSONGuardedRejections(ctx, source, sourceGuard, method, endpoint, body, headers.Values, policy, true, slices.Clone(headers.Absent), codes...)
}

func doJSONGuardedRejections(ctx context.Context, source *gophercloud.ServiceClient, sourceGuard func(context.Context) error, method, endpoint string, body any, headers map[string]string, policy RejectionPolicy, fixedHeaders bool, absentHeaders []string, codes ...int) (*Response, error) {
	selected := slices.Clone(policy.Codes)
	client, err := fixedrequest.NewGuarded(source, method, endpoint, sourceGuard)
	if err != nil {
		return nil, err
	}
	var faults RejectedResponseFaults
	guard := func(ctx context.Context) error {
		var sourceErr error
		if sourceGuard != nil {
			sourceErr = sourceGuard(ctx)
		}
		return joinResponseErrors(sourceErr, faults.Err())
	}
	parent := client.ProviderClient.HTTPClient.Transport
	client.ProviderClient.HTTPClient.Transport = rejectedResponseTransport(func(req *http.Request) (*http.Response, error) {
		response, err := parent.RoundTrip(req)
		if response != nil && response.Body != nil && slices.Contains(selected, response.StatusCode) {
			response.Body = &rejectedResponseBody{ReadCloser: response.Body, faults: &faults, ctx: req.Context(), guard: sourceGuard}
		}
		return response, err
	})
	var cleanRetry []int
	if policy.PreserveCleanRetry {
		cleanRetry = selected
	}
	return doJSONGuarded(ctx, client, guard, method, endpoint, body, headers, fixedHeaders, absentHeaders, cleanRetry, codes...)
}

type rejectedResponseTransport func(*http.Request) (*http.Response, error)

func (f rejectedResponseTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type rejectedResponseBody struct {
	io.ReadCloser
	faults *RejectedResponseFaults
	ctx    context.Context
	guard  func(context.Context) error
}

func (b *rejectedResponseBody) observeSource() {
	if b.guard != nil {
		b.faults.Add(b.guard(b.ctx))
	}
}
func (b *rejectedResponseBody) Read(data []byte) (int, error) {
	n, err := b.ReadCloser.Read(data)
	if err != io.EOF {
		b.faults.Add(err)
	}
	b.observeSource()
	return n, err
}
func (b *rejectedResponseBody) Close() error {
	err := b.ReadCloser.Close()
	b.faults.Add(err)
	b.observeSource()
	return err
}
