package compute

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

// ServerWithoutAddressesPolicy selects what the readiness workflows do when a
// server reaches ACTIVE with explicit empty address metadata.
type ServerWithoutAddressesPolicy int

const (
	// KeepServerWithoutAddresses returns the server with
	// ServerAddressesUnavailableError and sends no DELETE. It is the default.
	KeepServerWithoutAddresses ServerWithoutAddressesPolicy = iota
	// DeleteServerWithoutAddresses follows Python's get_active_server: it
	// sends exactly one DELETE for the server and still returns
	// ServerAddressesUnavailableError. A 404 DELETE is recorded as NotFound
	// without an error. When the workflow waits (WaitForServer,
	// CreateWithAutomaticFloatingIP, or GetActiveServer with
	// WithActiveServerWait), the SDK then polls GET until 404 inside the
	// remaining workflow budget. A failed DELETE or wait is never retried.
	DeleteServerWithoutAddresses
)

// ServerDeletion records the opt-in cleanup of an ACTIVE server without
// addresses. StatusCode is the DELETE response status, or zero when no
// response was received. Err is the DELETE or deletion-wait failure.
type ServerDeletion struct {
	StatusCode int
	NotFound   bool
	Waited     bool
	Err        error
}

// WithServerWithoutAddressesPolicy selects the ACTIVE-without-addresses
// policy for GetActiveServer and WaitForServer.
func WithServerWithoutAddressesPolicy(policy ServerWithoutAddressesPolicy) ServerReadyOption {
	return func(o *serverReadyOptions) error {
		if err := policy.validate(); err != nil {
			return err
		}
		o.withoutAddresses = policy
		return nil
	}
}

func (p ServerWithoutAddressesPolicy) validate() error {
	if p != KeepServerWithoutAddresses && p != DeleteServerWithoutAddresses {
		return invalid("unknown server-without-addresses policy %d", p)
	}
	return nil
}

// applyWithoutAddresses performs the selected cleanup once. It records the
// outcome on the ServerAddressesUnavailableError and joins a cleanup failure
// so errors.Is reaches both causes.
func (state *automaticIPState) applyWithoutAddresses(ctx context.Context, err error, policy ServerWithoutAddressesPolicy, wait bool) error {
	var unavailable *ServerAddressesUnavailableError
	if policy != DeleteServerWithoutAddresses || !errors.As(err, &unavailable) || unavailable.Deletion != nil {
		return err
	}
	deletion := state.deleteServer(ctx, wait)
	unavailable.Deletion = deletion
	if deletion.Err != nil {
		return errors.Join(err, deletion.Err)
	}
	return err
}

func (state *automaticIPState) deleteServer(ctx context.Context, wait bool) *ServerDeletion {
	deletion := &ServerDeletion{}
	client, err := state.rawClient(ctx)
	if err != nil {
		deletion.Err = err
		return deletion
	}
	target := client.ServiceURL("servers", url.PathEscape(state.serverID))
	if err := rest.ValidateTarget(client, target); err != nil {
		deletion.Err = err
		return deletion
	}
	response, err := rest.DoJSONGuarded(ctx, client, state.check, http.MethodDelete, target, nil, nil, 202, 204, 404)
	if response != nil {
		deletion.StatusCode = response.StatusCode
	}
	var unexpected gophercloud.ErrUnexpectedResponseCode
	if deletion.StatusCode == 0 && errors.As(err, &unexpected) {
		deletion.StatusCode = unexpected.Actual
	}
	if err != nil {
		deletion.Err = err
		return deletion
	}
	if response.StatusCode == http.StatusNotFound {
		deletion.NotFound = true
		return deletion
	}
	if !wait {
		return deletion
	}
	interval := state.options.interval
	if interval <= 0 {
		interval = 2 * time.Second
	}
	for {
		response, err := rest.DoJSONGuarded(ctx, client, state.check, http.MethodGet, target, nil, nil, 200, 203, 404)
		if err != nil {
			deletion.Err = err
			return deletion
		}
		if response.StatusCode == http.StatusNotFound {
			deletion.Waited = true
			return deletion
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			deletion.Err = errors.Join(ctx.Err(), context.Cause(ctx))
			return deletion
		case <-timer.C:
		}
	}
}
