package projectcleanup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/orchestration/v1/stacks"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// orchestrationCleanup follows openstack/orchestration/v1/_proxy.py
// _service_cleanup: it lists the project's stacks with the default query,
// evaluates each one, deletes the selected stacks and then waits for each
// deletion. The default Heat list contains only top-level stacks; Heat
// deletes nested stacks together with their parent, so they never become
// candidates of their own. Python's dependency graph runs orchestration
// before compute and network, which stageOrder reproduces.
type orchestrationCleanup struct{}

func (orchestrationCleanup) kinds() []kindSpec {
	return []kindSpec{
		{kind: "stack", createdField: "creation_time", updatedField: "updated_time", wait: true,
			remove: removeStack, absent: stackAbsent},
	}
}

func (orchestrationCleanup) discover(ctx context.Context, d *discovery) error {
	if d.Skipped("stack") {
		return nil
	}
	project := d.Project().ID
	for r, err := range listRows(ctx, d.Client(), "stacks", "stacks", nil) {
		if err != nil {
			return fmt.Errorf("list stacks: %w", err)
		}
		id, _ := r.str("id")
		name, _ := r.str("stack_name")
		// Heat routes a stack by name and ID; both must be usable segments.
		if (stacks.StackIdentity{Name: name, ID: id}).Validate() != nil {
			d.Exclude(Ref{Kind: "stack", ID: id}, name, "row has no usable stack_name and id", r.raw)
			continue
		}
		if reason, foreign := ownerMismatch(r, project, "project"); foreign {
			d.Exclude(Ref{Kind: "stack", ID: id}, name, reason, r.raw)
			continue
		}
		c := d.Row("stack", r, id, name)
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

// removeStack sends one DELETE to the canonical stacks/{name}/{id} route.
func removeStack(ctx context.Context, client *gophercloud.ServiceClient, c Candidate) error {
	scope, err := stacks.New(client).ForStack(stacks.StackIdentity{Name: c.Name, ID: c.ID})
	if err != nil {
		return err
	}
	return scope.Delete(ctx, resource.WithMissingError())
}

// stackAbsent polls GET stacks/{name}/{id} with the shared wait engine. A
// 404 or DELETE_COMPLETE is absence and DELETE_FAILED fails at once. Other
// states, including a *_FAILED state of an earlier action that Heat has not
// yet replaced with DELETE_IN_PROGRESS, keep polling.
func stackAbsent(ctx context.Context, client *gophercloud.ServiceClient, c Candidate, timeout, interval time.Duration) error {
	scope, err := stacks.New(client).ForStack(stacks.StackIdentity{Name: c.Name, ID: c.ID})
	if err != nil {
		return err
	}
	collection := resource.NewCollection(resource.Adapter[stacks.StackResource]{
		Kind: "stacks",
		ID:   func(value *stacks.StackResource) string { return value.ID },
		Get: func(ctx context.Context, id string) (*stacks.StackResource, error) {
			if id != c.ID {
				return nil, fmt.Errorf("%w: ID conflicts with the planned stack", resource.ErrInvalidOption)
			}
			value, err := scope.Get(ctx)
			if err != nil {
				return nil, err
			}
			switch strings.ToUpper(value.Status) {
			case "DELETE_COMPLETE":
				return nil, &resource.NotFoundError{Resource: "stacks", Reference: value.ID}
			case "DELETE_FAILED":
				return nil, &resource.FailedStateError{Resource: "stacks", ID: value.ID, Status: value.Status}
			}
			return value, nil
		},
	})
	return collection.WaitDeleted(ctx, resource.ID(c.ID), resource.WithTimeout(timeout), resource.WithPollInterval(interval))
}
