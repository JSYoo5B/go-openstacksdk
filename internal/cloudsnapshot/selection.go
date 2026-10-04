package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"net/url"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cloudfilter"
	"gophercloudsdk/resource"
)

func prepareSearch(ctx context.Context, client *gophercloud.ServiceClient, options []SearchOption) (*reader, SearchOptions, error) {
	p, err := capture(ctx, client)
	if err != nil {
		return nil, SearchOptions{}, err
	}
	policy, err := prepare(ctx, options, cloneSearch, func() error { return p.source.Guard(ctx) })
	if err == nil {
		err = p.ownLocation(ctx, policy.Location)
	}
	return p, policy, err
}

func Search(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string, options ...SearchOption) (*SearchResult, error) {
	p, policy, err := prepareSearch(ctx, client, options)
	if err != nil {
		return nil, wrap(ctx, "SearchVolumeSnapshots", err)
	}
	result, err := p.search(ctx, nameOrID, policy.Filters)
	return result, wrap(ctx, "SearchVolumeSnapshots", err)
}

func (p *reader) search(ctx context.Context, nameOrID string, filters *json.RawMessage) (*SearchResult, error) {
	result := &SearchResult{}
	proof := &ListResult{}
	policy, err := compileList(ListOptions{})
	if err != nil {
		return result, err
	}
	var values []*entry
	err = p.readList(ctx, policy, proof, func(value *entry) (bool, error) { values = append(values, value); return true, nil })
	result.Pages = proof.Pages
	if err != nil {
		return result, err
	}
	rows := make([]json.RawMessage, len(values))
	for i, value := range values {
		rows[i] = value.view
	}
	selected, err := cloudfilter.Select(rows, nameOrID, filters, func() error { return p.source.Guard(ctx) })
	if err != nil {
		return result, fmt.Errorf("%w: snapshot search: %w", resource.ErrInvalidOption, err)
	}
	result.Value = bytes.Clone(selected.Value)
	if !selected.Expression {
		result.Snapshots = make([]*resource.RawResource, len(selected.Indices))
		for i, index := range selected.Indices {
			result.Snapshots[i] = values[index].resource
		}
	}
	return result, nil
}

// SelectionError reports ambiguity in arbitrary JSON without inventing IDs.
type SelectionError struct {
	NameOrID string
	Length   int
}

func (e *SelectionError) Error() string {
	return fmt.Sprintf("multiple snapshot matches found for %q: length %d", e.NameOrID, e.Length)
}
func (e *SelectionError) Unwrap() error { return resource.ErrAmbiguous }

func Get(ctx context.Context, client *gophercloud.ServiceClient, nameOrID string, options ...SearchOption) (*Result, error) {
	p, policy, err := prepareSearch(ctx, client, options)
	if err != nil {
		return nil, wrap(ctx, "GetVolumeSnapshot", err)
	}
	result := &Result{}
	if policy.Filters != nil && !bytes.Equal(bytes.TrimSpace(*policy.Filters), []byte("null")) {
		search, err := p.search(ctx, nameOrID, policy.Filters)
		result.Pages = search.Pages
		if err != nil {
			return result, wrap(ctx, "GetVolumeSnapshot", err)
		}
		selected, err := cloudfilter.First(search.Value)
		if err != nil {
			var multiple *cloudfilter.MultipleError
			if errors.As(err, &multiple) {
				err = &SelectionError{NameOrID: nameOrID, Length: multiple.Length}
			} else {
				err = fmt.Errorf("%w: snapshot selection: %w", resource.ErrInvalidOption, err)
			}
			return result, wrap(ctx, "GetVolumeSnapshot", err)
		}
		if err := p.source.Guard(ctx); err != nil {
			return result, wrap(ctx, "GetVolumeSnapshot", err)
		}
		result.Value = bytes.Clone(selected)
		if selected != nil && len(search.Snapshots) == 1 {
			result.Snapshot = search.Snapshots[0]
		}
		return result, nil
	}
	value, err := p.collection(result).FindIdentity(ctx, nameOrID)
	if err != nil {
		if ctx.Err() != nil && p.memberFailure != nil {
			err = errors.Join(p.memberFailure, err)
		}
		return result, wrap(ctx, "GetVolumeSnapshot", err)
	}
	if err := p.source.Guard(ctx); err != nil {
		return result, wrap(ctx, "GetVolumeSnapshot", err)
	}
	if value != nil {
		result.Value, result.Snapshot = bytes.Clone(value.view), value.resource
		if result.Observed != nil {
			result.RequestedID, result.SeededID = nameOrID, value.seeded
		}
	}
	return result, nil
}

func (p *reader) collection(result *Result) *resource.Collection[entry] {
	get := func(ctx context.Context, id string) (*entry, error) {
		value, err := p.member(ctx, id, result)
		if err != nil {
			// Only the original direct native rejection can enable fallback.
			// Expanded OkCodes and callback/transport wrappers retain native
			// evidence through Unwrap but cannot establish compatible absence.
			_, clean := err.(gophercloud.ErrUnexpectedResponseCode)
			if !clean && (gophercloud.ResponseCodeIs(err, 400) || gophercloud.ResponseCodeIs(err, 403) || gophercloud.ResponseCodeIs(err, 404)) {
				return value, &terminalMemberError{cause: err}
			}
		}
		return value, err
	}
	iterate := func(ctx context.Context, query url.Values, details bool) iter.Seq2[*entry, error] {
		return func(yield func(*entry, error) bool) {
			// A suppressed native member error is not the list's observation.
			p.memberFailure, result.Observed = nil, nil
			policy, err := compileList(ListOptions{Detailed: &details})
			if err != nil {
				yield(nil, err)
				return
			}
			for key, values := range query {
				raw, _ := json.Marshal(values)
				policy.query[key] = raw
			}
			proof := &ListResult{}
			err = p.readList(ctx, policy, proof, func(value *entry) (bool, error) { return yield(value, nil), nil })
			result.Pages = append(result.Pages, proof.Pages...)
			if err != nil {
				yield(nil, err)
			}
		}
	}
	return resource.NewCollection(resource.Adapter[entry]{Kind: "volume snapshot", IdentityFind: true, Get: get, IterateIdentity: iterate,
		ID: func(value *entry) string { return value.id }, IdentityResponseID: func(value *entry) (string, error) { return value.id, nil },
		Name: func(value *entry) string { return value.name }, NameQuery: func(name string) string { return name },
	})
}

type terminalMemberError struct{ cause error }

func (e *terminalMemberError) Error() string        { return e.cause.Error() }
func (e *terminalMemberError) Unwrap() error        { return e.cause }
func (e *terminalMemberError) Is(target error) bool { return target == resource.ErrInvalidOption }
