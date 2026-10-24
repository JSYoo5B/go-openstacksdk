package openstack

import "context"

// WithServiceLookupFailureAsUnavailable selects Python's has_service policy
// for network role discovery and the floating IP helpers' backend choice: any
// failure to look up the Network endpoint (for example a catalog or
// authentication error) makes the service unavailable, so roles are empty and
// the helpers use Nova, instead of returning the error. A missing endpoint is
// always unavailable, and context cancellation is always returned. The default
// (false) returns lookup failures so that they stay observable. Nothing is
// cached: the next call looks the endpoint up again, as Python does.
func WithServiceLookupFailureAsUnavailable(enabled bool) ConnectionOption {
	return func(o *connectionOptions) error { o.lookupFailureUnavailable = enabled; return nil }
}

// networkLookupUnavailable reports whether a Network service lookup error
// means "no Neutron" rather than a returned failure.
func (c *Connection) networkLookupUnavailable(ctx context.Context, err error) bool {
	if missingCatalogEndpoint(err) {
		return true
	}
	return c.options.lookupFailureUnavailable && ctx.Err() == nil
}
