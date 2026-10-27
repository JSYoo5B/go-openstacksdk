package projectcleanup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/loadbalancer/v2/loadbalancers"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// loadBalancerCleanup follows openstack/load_balancer/v2/_proxy.py
// _service_cleanup: it lists load balancers without a query, evaluates each
// one, deletes the selected ones with cascade=true (Octavia removes their
// listeners, pools, members, health monitors and L7 policies) and then waits
// for each deletion. Python's dependency graph runs load_balancer before
// compute and network, which stageOrder reproduces.
type loadBalancerCleanup struct{}

func (loadBalancerCleanup) kinds() []kindSpec {
	return []kindSpec{
		{kind: "load_balancer", createdField: "created_at", updatedField: "updated_at", wait: true,
			remove: removeLoadBalancer, absent: loadBalancerAbsent},
	}
}

func (loadBalancerCleanup) discover(ctx context.Context, d *discovery) error {
	if d.Skipped("load_balancer") {
		return nil
	}
	project := d.Project().ID
	// Python sends no query. Octavia limits the list to the token project
	// unless the token may list every project; rows of other projects are
	// excluded below, so the owner field is the ownership boundary.
	for r, err := range listRows(ctx, d.Client(), "lbaas/loadbalancers", "loadbalancers", nil) {
		if err != nil {
			return fmt.Errorf("list load balancers: %w", err)
		}
		id, name, ok := rowIdentity(d, "load_balancer", r)
		if !ok {
			continue
		}
		ref := Ref{Kind: "load_balancer", ID: id}
		if reason, foreign := ownerMismatch(r, project, "project_id", "tenant_id"); foreign {
			d.Exclude(ref, name, reason, r.raw)
			continue
		}
		if owner, _ := r.str("project_id"); owner != project {
			d.Exclude(ref, name, "project_id is not reported", r.raw)
			continue
		}
		c := d.Row("load_balancer", r, id, name)
		keep, err := d.Evaluate(c)
		if err != nil {
			return err
		}
		if keep {
			c.Reason = selectionReason(d)
			d.Add(c, 0)
		}
	}
	return nil
}

// removeLoadBalancer sends one DELETE lbaas/loadbalancers/{id}?cascade=true,
// as Python's delete_load_balancer(cascade=True) does. A 404 is reported as
// resource.ErrNotFound.
func removeLoadBalancer(ctx context.Context, client *gophercloud.ServiceClient, c Candidate) error {
	if err := resource.ID(c.ID).Validate(); err != nil {
		return err
	}
	err := loadbalancers.New(client).Delete(ctx, c.ID, loadbalancers.WithDeleteOptions(loadbalancers.DeleteOpts{Cascade: true}))
	if err != nil && resource.IgnoreMissing(err) == nil {
		return &resource.NotFoundError{Resource: "loadbalancers", Reference: c.ID, Cause: err}
	}
	return err
}

// loadBalancerAbsent polls GET lbaas/loadbalancers/{id} with the shared wait
// engine. A 404 or provisioning_status DELETED is absence and ERROR fails at
// once; PENDING_DELETE and every other state keep polling.
func loadBalancerAbsent(ctx context.Context, client *gophercloud.ServiceClient, c Candidate, timeout, interval time.Duration) error {
	api := loadbalancers.New(client)
	collection := resource.NewCollection(resource.Adapter[loadbalancers.LoadBalancer]{
		Kind: "loadbalancers",
		ID:   func(value *loadbalancers.LoadBalancer) string { return value.ID },
		Get: func(ctx context.Context, id string) (*loadbalancers.LoadBalancer, error) {
			value, err := api.Get(ctx, id)
			if err != nil {
				return nil, err
			}
			switch strings.ToUpper(value.ProvisioningStatus) {
			case "DELETED":
				return nil, &resource.NotFoundError{Resource: "loadbalancers", Reference: id}
			case "ERROR":
				return nil, &resource.FailedStateError{Resource: "loadbalancers", ID: id, Status: value.ProvisioningStatus}
			}
			return value, nil
		},
	})
	return collection.WaitDeleted(ctx, resource.ID(c.ID), resource.WithTimeout(timeout), resource.WithPollInterval(interval))
}
