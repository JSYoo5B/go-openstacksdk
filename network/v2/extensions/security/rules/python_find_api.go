package rules

import (
	"bytes"
	"context"
	"errors"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func findSegment(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) {
		return findInvalid("nonblank UTF-8 identity required")
	}
	for _, r := range id {
		if unicode.IsControl(r) {
			return findInvalid("identity has controls")
		}
	}
	return nil
}
func findCodes(start, end int) []int {
	out := make([]int, end-start)
	for i := range out {
		out[i] = i + start
	}
	return out
}

// FindRecord follows the pinned Source member-first exact ID-or-name lookup.
// Only clean native400/403/404 can fall back; default missing is nil,nil after
// an entirely successful search, while late duplicate and page errors survive.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*SecurityGroupRuleRecord, error) {
	wrap := func(err error) error { return request.Wrap("FindRecord", "rules", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("configured API required"))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, wrap(err)
	}
	listSource := *source
	var changed error
	guard := func(ctx context.Context) error {
		if changed == nil && a.client != selected {
			changed = findInvalid("selected Neutron client changed")
		}
		return errors.Join(changed, source.Guard(ctx), rest.CheckOperationGuard(ctx))
	}
	policy := RecordFindOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil find option"))
		}
		next := copyFindOpts(policy)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = copyFindOpts(next)
	}
	memberQuery, listQuery, filters, seed, match, control, err := prepareFindQuery(policy, identity)
	if err == nil {
		err = findSegment(identity)
	}
	if err == nil {
		// Resource.find passes explicit microversion only to fetch. Snapshot the
		// fallback service default before applying that member override.
		if policy.Microversion != nil {
			source.Client.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
			for key := range source.Client.MoreHeaders {
				if strings.EqualFold(key, "OpenStack-API-Version") {
					delete(source.Client.MoreHeaders, key)
				}
			}
		}
		err = errors.Join(source.WithPolicy(ctx, policy.Microversion, policy.Headers), listSource.WithPolicy(ctx, nil, policy.Headers))
	}
	if err = errors.Join(err, guard(ctx)); err != nil {
		return nil, wrap(err)
	}
	// These Neutron resources have no class ceiling. Do not perform discovery.
	// Keep selected protocol values or absence and all captured ordinary headers
	// unchanged through native retries/reauthentication and physical attempts.
	read := func(target string, list bool, codes ...int) (*rest.Response, error) {
		client := source.Client
		if list {
			client = listSource.Client
		}
		client.MoreHeaders = maps.Clone(client.MoreHeaders)
		if client.MoreHeaders == nil {
			client.MoreHeaders = make(map[string]string)
		}
		if list {
			// Source list supplies its own Accept default even if the service
			// was configured with another default. The explicit Go option wins.
			client.MoreHeaders["Accept"] = "application/json"
			for key, value := range policy.Headers {
				if strings.EqualFold(key, "Accept") {
					client.MoreHeaders["Accept"] = value
				}
			}
		}
		fixed := rest.RequestHeaderPolicy{Values: maps.Clone(client.MoreHeaders), Absent: []string{
			"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length",
			"Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade",
			"X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version",
		}}
		if client.Microversion != "" {
			fixed.Values["OpenStack-API-Version"] = "network " + client.Microversion
		} else {
			fixed.Absent = append(fixed.Absent, "OpenStack-API-Version")
		}
		return rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &client, guard, http.MethodGet, target, nil, fixed, rest.RejectionPolicy{Codes: findCodes(400, 600), PreserveCleanRetry: true}, codes...)
	}
	var failedMember *SecurityGroupRuleRecord
	get := func(_ context.Context, id string) (*SecurityGroupRuleRecord, error) {
		target, err := url.Parse(source.Client.ServiceURL("security-group-rules", url.PathEscape(id)))
		if err != nil {
			return nil, err
		}
		target.RawQuery = memberQuery.Encode()
		wire, err := read(target.String(), false, findCodes(200, 400)...)
		if err != nil {
			failedMember = findReceipt(wire)
			return failedMember, err
		}
		value, err := findResponse(wire, seed, match)
		if err = errors.Join(err, guard(ctx)); err != nil {
			failedMember = value
			err = wire.Fail(err)
		}
		return value, err
	}
	list := func(context.Context, url.Values) iter.Seq2[*SecurityGroupRuleRecord, error] {
		return func(yield func(*SecurityGroupRuleRecord, error) bool) {
			var origin *rest.Response
			spec := rest.CollectionSpec[SecurityGroupRuleRecord]{Client: &listSource.Client, Path: "security-group-rules", Kind: "rules", PluralKey: "security_group_rules", Metadata: findMetadata, Validate: guard, SourceGuard: guard, ListCodes: findCodes(200, 400), ReadPage: func(_ context.Context, target string, codes ...int) (*rest.Response, error) {
				return read(target, true, codes...)
			}, ValidateResponse: func(wire *rest.Response) error { origin = wire; return guard(ctx) }, ValidateItem: func(value *SecurityGroupRuleRecord) error {
				value.Envelope = bytes.Clone(origin.Body)
				value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
				return errors.Join(findProject(value, value.fields, value.match), guard(ctx))
			}, Paging: rest.PagePolicy[SecurityGroupRuleRecord]{VersionedPath: "/v2.0/security-group-rules", HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, MarkerFallback: true, Marker: findMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true}}
			keys := make([]string, 0, len(filters))
			for key := range filters {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for value, err := range rest.ListWithControl(ctx, spec, listQuery, control) {
				if err != nil {
					yield(nil, err)
					return
				}
				matched := true
				for _, key := range keys {
					equal, cause := findLocalValue(value.Resource.Body[key], filters[key])
					if cause = errors.Join(cause, guard(ctx)); cause != nil {
						yield(nil, origin.Fail(cause))
						return
					}
					if !equal {
						matched = false
						break
					}
				}
				if matched && !yield(value, nil) {
					return
				}
			}
		}
	}
	collection := resource.NewCollection(resource.Adapter[SecurityGroupRuleRecord]{Kind: "rules", IdentityFind: true, ValidateID: findSegment, IdentityDirectGet: findSegment, IdentityResponseID: func(value *SecurityGroupRuleRecord) (string, error) { return findText(value, "id"), nil }, Name: func(value *SecurityGroupRuleRecord) string { return findText(value, "name") }, Get: get, Iterate: list})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	if value == nil && err != nil && failedMember != nil {
		value = failedMember
	}
	return value, wrap(errors.Join(err, guard(ctx)))
}
