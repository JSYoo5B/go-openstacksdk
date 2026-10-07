package compute

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"gophercloudsdk/internal/cloudfilter"
	"gophercloudsdk/internal/cloudread"
	"gophercloudsdk/resource"
)

// DeleteUnattachedFloatingIPs lists the complete public inventory before
// deleting falsey-port rows in order. Only configured, available Neutron is
// eligible; an inventory fallback does not change the deletion backend.
// Options are prepared once and one budget covers every query and deletion.
func (s *Service) DeleteUnattachedFloatingIPs(ctx context.Context, options ...FloatingIPDeleteOption) (*DeleteUnattachedFloatingIPsResult, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if s == nil || s.API == nil || s.Servers == nil || s.Servers.Collection == nil {
		return nil, invalid("floating IP cleanup facade is required")
	}
	base := s.captureAutomaticComputeSource()
	policy, err := prepareFloatingIPDeleteOptions(options, func() error { return base(ctx) })
	if err != nil {
		return nil, err
	}
	p, err := s.captureFloatingIPQuery(ctx, []FloatingIPQueryOption{WithFloatingIPQueryOptions(policy.query())})
	if err != nil {
		return nil, err
	}
	defer p.cancel()
	result := &DeleteUnattachedFloatingIPsResult{}
	backend, _, err := p.backend()
	if err != nil {
		return result, err
	}
	if backend != FloatingIPNeutron {
		if err := p.state.check(p.ctx); err != nil {
			return result, err
		}
		result.AllDeleted = true
		return result, nil
	}
	result.Eligible = true
	result.Inventory, err = p.list(nil)
	if err != nil {
		return result, errors.Join(err, p.state.check(p.ctx))
	}
	allDeleted := true
	for _, record := range result.Inventory.FloatingIPs {
		if err := p.state.check(p.ctx); err != nil {
			return result, err
		}
		if record == nil || record.Resource == nil {
			return result, errors.Join(invalid("floating IP cleanup inventory lacks a Resource view"), p.state.check(p.ctx))
		}
		port, present := record.Resource.Body["port_id"]
		if !present {
			return result, errors.Join(invalid("floating IP cleanup inventory lacks port_id"), p.state.check(p.ctx))
		}
		attached, err := cloudfilter.PythonTruthy(port)
		if err != nil {
			return result, errors.Join(err, p.state.check(p.ctx))
		}
		if attached {
			continue
		}
		item := &UnattachedFloatingIPDelete{FloatingIP: cloneFloatingIPRecord(record)}
		result.Items = append(result.Items, item)
		item.ID, err = unattachedFloatingIPID(record)
		if err == nil {
			item.Deletion, err = p.deleteWithRetries(item.ID, policy.Retries)
		}
		if err == nil {
			if item.Deletion.Deleted {
				result.Count++
			} else {
				allDeleted = false
			}
		}
		item.Error = errors.Join(err, p.state.check(p.ctx))
		if item.Error != nil {
			return result, item.Error
		}
	}
	if err := p.state.check(p.ctx); err != nil {
		return result, err
	}
	result.AllDeleted = allDeleted
	return result, nil
}

// IDs are consumed from the public Resource only after port eligibility,
// retaining prior deletions when a later eligible row has an unusable ID.
func unattachedFloatingIPID(record *FloatingIPRecord) (string, error) {
	raw := bytes.TrimSpace(record.Resource.Body["id"])
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) || raw[0] == '[' || raw[0] == '{' {
		return "", invalid("floating IP cleanup requires a scalar Resource ID")
	}
	values, err := cloudfilter.RequestQueryValues(raw)
	if err != nil {
		return "", fmt.Errorf("%w: floating IP cleanup ID: %w", resource.ErrInvalidOption, err)
	}
	if len(values) != 1 {
		return "", invalid("floating IP cleanup requires one Resource ID")
	}
	if err := resource.ID(values[0]).Validate(); err != nil {
		return "", err
	}
	return values[0], nil
}
