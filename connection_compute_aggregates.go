package openstack

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"unicode"

	"github.com/JSYoo5B/go-openstacksdk/compute/v2/aggregates"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// UpdateAggregate finds a host aggregate by Nova ID-or-name (strict: a missing
// aggregate is resource.ErrNotFound and no update is sent) and then sends one
// PUT with only the attributes supplied through the update options.
func (c *Connection) UpdateAggregate(ctx context.Context, nameOrID string, options ...aggregates.UpdateRecordOption) (*aggregates.AggregateRecord, error) {
	fail := func(err error) (*aggregates.AggregateRecord, error) {
		return nil, request.Wrap("UpdateAggregate", "aggregate", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return fail(err)
	}
	found, err := service.Aggregates.FindIdentity(ctx, nameOrID, resource.WithIdentityFindIgnoreMissing(false))
	if err != nil {
		return fail(err)
	}
	record, err := service.Aggregates.UpdateRecord(ctx, found.ID, options...)
	if err != nil {
		return fail(err)
	}
	return record, nil
}

// DeleteAggregate deletes a host aggregate and reports whether a delete
// happened. An identity made only of digits is taken as an aggregate ID and
// deleted without any lookup; any other identity is first found by name or ID.
// A missing aggregate, including a 404 on the DELETE itself, returns
// (false, nil). Every other failure is returned and the DELETE is never replayed.
func (c *Connection) DeleteAggregate(ctx context.Context, nameOrID string) (bool, error) {
	fail := func(err error) (bool, error) {
		return false, request.Wrap("DeleteAggregate", "aggregate", err)
	}
	if err := ctx.Err(); err != nil {
		return fail(err)
	}
	if nameOrID == "" {
		return fail(fmt.Errorf("%w: aggregate name or ID is required", resource.ErrInvalidOption))
	}
	service, err := c.ComputeV2(ctx)
	if err != nil {
		return fail(err)
	}
	var id int
	if aggregateDigitsOnly(nameOrID) {
		id, err = strconv.Atoi(nameOrID)
		if err != nil {
			return fail(fmt.Errorf("%w: aggregate ID %q is not a valid integer", resource.ErrInvalidOption, nameOrID))
		}
	} else {
		found, err := service.Aggregates.FindIdentity(ctx, nameOrID)
		if err != nil {
			return fail(err)
		}
		if found == nil {
			return false, nil
		}
		id = found.ID
	}
	if err := service.Aggregates.Delete(ctx, id); err != nil {
		if gophercloud.ResponseCodeIs(err, http.StatusNotFound) {
			return false, nil
		}
		return fail(err)
	}
	return true, nil
}

func aggregateDigitsOnly(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r > unicode.MaxASCII || !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
