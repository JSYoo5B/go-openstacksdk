package servicewait

import (
	"context"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// declaresStatus reports whether T has an exported string attribute named
// status, which the generic openstack/resource.py waiters read by default.
func declaresStatus[T any]() bool {
	return resource.ValidateWaitOptionsFor[T](resource.WithStatusAttribute("status")) == nil
}

// ProxyStatus applies the unmodified Proxy.wait_for_status defaults used by
// the Neutron, Keystone v3, Placement and Inspector proxies: the status
// attribute (when the model declares one), exact case-insensitive ERROR
// failure, a two-second interval and no SDK timeout. A model without a status
// attribute needs resource.WithStatusAttribute, matching Python's
// AttributeError for a missing attribute. Caller options apply last.
func ProxyStatus[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, target string, options ...resource.WaitOption) (*T, error) {
	if err := validate(ctx, collection); err != nil {
		return nil, err
	}
	policy := []resource.WaitOption{resource.WithUnlimitedWait(), resource.WithPollInterval(2 * time.Second), resource.WithFailureStates("ERROR")}
	if declaresStatus[T]() {
		policy = append(policy, resource.WithStatusAttribute("status"))
	}
	return collection.Wait(ctx, ref, target, append(policy, options...)...)
}

// ProxyDelete applies the unmodified Proxy.wait_for_delete defaults: a
// two-second interval and a 120-second SDK timeout. A 404 or a successful nil
// observation completes; a model that declares a status attribute also
// completes on a case-insensitive "deleted" value. Only GET requests are sent.
// Caller options apply last.
func ProxyDelete[T any](ctx context.Context, collection *resource.Collection[T], ref resource.Ref, options ...resource.WaitOption) error {
	if err := validate(ctx, collection); err != nil {
		return err
	}
	policy := []resource.WaitOption{resource.WithTimeout(120 * time.Second), resource.WithPollInterval(2 * time.Second)}
	if declaresStatus[T]() {
		policy = append(policy, resource.WithStatusAttribute("status"))
	}
	return collection.WaitDeleted(ctx, ref, append(policy, options...)...)
}
