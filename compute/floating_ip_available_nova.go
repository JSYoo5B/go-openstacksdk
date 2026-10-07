package compute

import (
	"bytes"
	"encoding/json"
	"errors"

	"gophercloudsdk/internal/cloudfilter"
	"gophercloudsdk/resource"
)

// Nova availability filters raw rows before normalizing every selected row.
// The association engine's executable ID/IPv4 checks do not apply to this read.
func (p *floatingIPQueryState) availableNova(result *AvailableFloatingIPResult, ref resource.Ref) (*AvailableFloatingIPResult, error) {
	client, err := p.novaClient()
	if err != nil {
		return result, err
	}
	value := &NovaFloatingIPAvailability{}
	result.Nova = value
	var pool json.RawMessage
	if ref.String() != "" {
		pool, err = json.Marshal(ref.String())
	} else {
		value.PoolQuery, err = p.pools()
		if err == nil {
			if len(value.PoolQuery.Pools) == 0 {
				return result, &resource.NotFoundError{Resource: "floating IP pool"}
			}
			pool = bytes.Clone(value.PoolQuery.Pools[0].Body["name"])
		}
	}
	if err != nil {
		return result, err
	}
	rows, pages, err := p.collect(FloatingIPNova, client, "os-floating-ips", "floating_ips", floatingIPQueryParameters{})
	inventory := &FloatingIPQueryResult{Backend: FloatingIPNova, Pages: pages, Failure: queryFailure(FloatingIPNova, err)}
	value.Inventory = inventory
	if err != nil {
		if !queryNotFound(err) {
			return result, err
		}
		inventory.SuppressedNotFound = err
		rows = nil
	}
	// Preserve the source dictionary's insertion order and missing-key errors.
	filter := append(json.RawMessage(`{"instance_id":null,"pool":`), pool...)
	filter = append(filter, '}')
	selected := make([]floatingIPQueryRow, 0, len(rows))
	for _, row := range rows {
		raw, encodeErr := json.Marshal(row.wire)
		if encodeErr != nil {
			return result, encodeErr
		}
		match, selectErr := cloudfilter.Select([]json.RawMessage{raw}, "", &filter, func() error { return p.state.check(p.ctx) })
		if selectErr != nil {
			err = row.origin.Fail(selectErr)
			inventory.Failure = queryFailure(FloatingIPNova, err)
			return result, err
		}
		if len(match.Indices) != 0 {
			selected = append(selected, row)
		}
	}
	inventory.FloatingIPs, err = p.records(FloatingIPNova, client, selected, nil)
	if err == nil {
		inventory.Value, err = queryValues(inventory.FloatingIPs)
	}
	if err != nil {
		inventory.Failure = queryFailure(FloatingIPNova, err)
		return result, err
	}
	if len(inventory.FloatingIPs) != 0 {
		result.FloatingIP = cloneFloatingIPRecord(inventory.FloatingIPs[0])
		result.Reused, value.Reused = true, true
	} else {
		creation := &CreateFloatingIPResult{Backend: FloatingIPNova, PoolQuery: value.PoolQuery}
		// A null first pool becomes None in Python and triggers create's own
		// default lookup. All other already selected values go directly to POST.
		if bytes.Equal(bytes.TrimSpace(pool), []byte("null")) {
			value.Creation, err = p.createNova(creation, nil)
		} else {
			value.Creation, err = p.createNovaInPool(creation, client, pool)
		}
		if created := value.Creation; created != nil {
			result.Allocated, value.Allocated = created.Allocated, created.Allocated
			result.FloatingIP = cloneFloatingIPRecord(created.FloatingIP)
			if receipt := created.AllocationResponse; receipt != nil {
				value.AllocationResponse = &NovaFloatingIPResponse{Body: bytes.Clone(receipt.Envelope), Header: receipt.Header.Clone(), StatusCode: receipt.StatusCode}
			}
		}
	}
	// The cloud view is authoritative. The old string-based model is an
	// optional independent projection and never narrows permissive read values.
	if record := result.FloatingIP; record != nil && record.Wire != nil {
		raw, encodeErr := json.Marshal(record.Wire)
		var typed NovaFloatingIP
		if encodeErr == nil && json.Unmarshal(raw, &typed) == nil {
			typed.Header, typed.StatusCode = record.Wire.Header.Clone(), record.Wire.StatusCode
			value.FloatingIP = &typed
			result.ID, result.Address = typed.ID, typed.Address
		}
	}
	return result, errors.Join(err, p.state.check(p.ctx))
}
