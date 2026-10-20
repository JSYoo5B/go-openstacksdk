package consistencygroupsnapshots

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlocation"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// findPhase clones one original capture, never re-capturing after a callback or
// carrying the GET override into fallback. The original sticky guard
// remains the operation owner, even if the caller restores a changed source.
func (p *recordPrepared) findPhase() *recordPrepared {
	owned := *p.source
	owned.Client.MoreHeaders = maps.Clone(p.source.Client.MoreHeaders)
	for key := range owned.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Volume-API-Version") {
			delete(owned.Client.MoreHeaders, key)
		}
	}
	return &recordPrepared{ctx: p.ctx, source: &owned, guard: p.guard}
}
func prepareFind(identity string, cfg RecordFindOpts) (map[string]json.RawMessage, url.Values, RecordListOpts, error) {
	seed := make(map[string]json.RawMessage)
	query := make(url.Values)
	list := RecordListOpts{Headers: recordHeaders(cfg.Headers), Location: cloneLocation(cfg.Location), Query: recordClone(cfg.Query), Details: cloneBool(cfg.Details), Paginated: cloneBool(cfg.Paginated), MaxItems: cfg.MaxItems}
	for _, key := range slices.Sorted(maps.Keys(cfg.Query)) {
		if err := findQueryKey(key); err != nil {
			return nil, nil, list, err
		}
		raw := cfg.Query[key]
		if !utf8.Valid(raw) || !json.Valid(raw) {
			return nil, nil, list, recordInvalid("find query %q requires UTF-8 JSON", key)
		}

		if key == "all_projects" {
			// Resource.find's all_projects parameter is excluded from fetch.
			if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
				delete(list.Query, key)
				continue
			}
			var value bool
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, nil, list, recordInvalid("find all_projects requires bool or null")
			}
			continue
		}
		values, err := cloudfilter.RequestQueryValues(raw)
		if err != nil {
			return nil, nil, list, err
		}
		if len(values) != 0 {
			query[key] = slices.Clone(values)
		}
		if recordKnown(key) {
			seed[key] = bytes.Clone(raw)
		}
	}
	seed["id"], _ = json.Marshal(identity)
	// Reuse list validation before any GET, rather than finding
	// an invalid cap/cursor only after a clean member rejection.
	_, _, _, err := prepareList(list)
	return seed, query, list, err
}
func findText(value *ConsistencyGroupSnapshotRecord, key string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	var text string
	if json.Unmarshal(value.Resource.Body[key], &text) != nil {
		return ""
	}
	return text
}
func findErrorReceipt(err error) *ConsistencyGroupSnapshotRecord {
	var accepted *resource.ResponseError
	if errors.As(err, &accepted) {
		return recordReceipt(&rest.Response{Body: bytes.Clone(accepted.Body), Header: accepted.Header.Clone(), StatusCode: accepted.StatusCode})
	}
	return recordReceipt(recordNativeResponse(err))
}

// FindRecord implements the pinned named find_cgsnapshot defaults: escaped
// GET first, clean 400/403/404 fallback, exact ID-or-name match, complete unique
// search, duplicate error, and nil for a complete missing result by default.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*ConsistencyGroupSnapshotRecord, error) {
	const op = "FindRecord"
	p, err := a.capture(ctx)
	if err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	if err := recordID(identity); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	cfg, err := applyRecord(p, RecordFindOpts{}, copyFind, slices.Clone(options))
	var seed map[string]json.RawMessage
	var query url.Values
	var listCfg RecordListOpts
	if err == nil {
		seed, query, listCfg, err = prepareFind(identity, cfg)
	}
	// Source existing.__init__ eagerly projects descriptors/location before fetch.
	location := resource.CloudLocation{}
	if err == nil {
		if cfg.Location != nil {
			location = cfg.Location.Clone()
		} else {
			location.Project.ID, err = cloudlocation.ProjectID(p.source.Client.ProviderClient)
		}
	}
	if err == nil {
		err = recordProject(&ConsistencyGroupSnapshotRecord{}, seed, location)
	}
	if err = errors.Join(err, p.guard(ctx)); err != nil {
		return nil, recordWrap(ctx, op, err)
	}
	getPhase, listPhase := p.findPhase(), p.findPhase()
	// Only a member rejection belongs to the identity fallback policy. Policy,
	// constructor and version policy errors remain preparation failures.
	if err := errors.Join(getPhase.prepare(cfg.Microversion, cfg.Headers, cfg.Location, false), p.guard(ctx)); err != nil {
		return findErrorReceipt(err), recordWrap(ctx, op, err)
	}
	var failure *ConsistencyGroupSnapshotRecord
	listStarted := false
	collection := resource.NewCollection(resource.Adapter[ConsistencyGroupSnapshotRecord]{
		Kind: "cgsnapshots", IdentityFind: true, ValidateID: recordID, IdentityDirectGet: recordID,
		IdentityResponseID: func(v *ConsistencyGroupSnapshotRecord) (string, error) {
			if v == nil || v.Resource == nil {
				return "", recordInvalid("find requires a declared consistency group snapshot")
			}
			return findText(v, "id"), nil
		},
		Name: func(v *ConsistencyGroupSnapshotRecord) string { return findText(v, "name") },
		Get: func(_ context.Context, id string) (*ConsistencyGroupSnapshotRecord, error) {
			v, err := getPhase.getPrepared(id, seed, query)
			if err != nil {
				failure = v
				if failure == nil {
					failure = findErrorReceipt(err)
				}
			}
			return v, err
		},
		Iterate: func(_ context.Context, _ url.Values) iter.Seq2[*ConsistencyGroupSnapshotRecord, error] {
			listStarted = true
			failure = nil
			return func(yield func(*ConsistencyGroupSnapshotRecord, error) bool) {
				for v, err := range listPhase.listRecords(listCfg) {
					if err != nil {
						failure = findErrorReceipt(err)
					}
					if !yield(v, err) {
						return
					}
				}
			}
		},
	})
	findOptions := []resource.IdentityFindOption{}
	if cfg.IgnoreMissing != nil {
		findOptions = append(findOptions, resource.WithIdentityFindIgnoreMissing(*cfg.IgnoreMissing))
	}
	value, err := collection.FindIdentity(ctx, identity, findOptions...)
	err = errors.Join(err, p.guard(ctx))
	if err != nil {
		// Actual terminal HTTP evidence is retained; logical missing/duplicates
		// never receive a suppressed member receipt or a successful candidate.
		if value == nil {
			value = failure
		}
		if listStarted && failure == nil {
			value = nil
		}
	}

	return value, recordWrap(ctx, op, err)
}
