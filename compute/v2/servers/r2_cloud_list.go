package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlocation"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// ListCloudRecords streams the Source Proxy.servers rows that cloud
// list_servers/search_servers consume: GET /servers/detail by default
// (WithRecordFindDetails(false) selects GET /servers), all_tenants=True only
// when WithRecordFindAllProjects(true) is supplied, and the same Python-named
// query pushdown, local descriptor filters, paging, microversion negotiation,
// header policy and location projection as FindRecord. Each yielded record
// carries the Source Resource view, the received row and the page receipt.
// IgnoreMissing is ignored: a list has no missing member. The read is never
// replayed and an absent row set is an empty stream, not an error.
func (a *API) ListCloudRecords(ctx context.Context, options ...RecordFindOption) iter.Seq2[*ServerFindRecord, error] {
	return func(yield func(*ServerFindRecord, error) bool) {
		wrap := func(err error) error {
			return request.Wrap("ListCloudRecords", "servers", cloudread.ContextError(ctx, err))
		}
		if err := cloudread.Context(ctx); err != nil {
			yield(nil, wrap(err))
			return
		}
		if a == nil {
			yield(nil, wrap(serverFindInvalid("server API required")))
			return
		}
		selected := a.client
		source, err := cloudread.Capture(ctx, selected, "compute")
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		var changed error
		ownGuard := func(c context.Context) error {
			if changed == nil && a.client != selected {
				changed = serverFindInvalid("selected server API client changed")
			}
			return errors.Join(changed, source.Guard(c))
		}
		ctx = rest.WithOperationGuard(ctx, ownGuard)
		guard := func(c context.Context) error { return errors.Join(ownGuard(c), rest.CheckOperationGuard(c)) }
		policy := RecordFindOpts{}
		for _, apply := range slices.Clone(options) {
			if err := guard(ctx); err != nil {
				yield(nil, wrap(err))
				return
			}
			if apply == nil {
				yield(nil, wrap(serverFindInvalid("nil list option")))
				return
			}
			next := serverFindCopy(policy)
			if err := errors.Join(apply(&next), guard(ctx)); err != nil {
				yield(nil, wrap(err))
				return
			}
			policy = serverFindCopy(next)
		}
		listQuery, filters, control, err := serverListPrepareQuery(policy)
		location := resource.CloudLocation{}
		if err == nil {
			if policy.Location != nil {
				location = policy.Location.Clone()
			} else {
				location.Project.ID, err = cloudlocation.ProjectID(selected.ProviderClient)
			}
		}
		if err == nil {
			_, err = location.ForResource(nil, nil)
		}
		if err == nil {
			for key := range policy.Headers {
				lower := strings.ToLower(key)
				if lower == "openstack-api-version" || strings.HasPrefix(lower, "x-openstack-") && strings.HasSuffix(lower, "-api-version") {
					err = serverFindInvalid("protocol Header %q requires the owned microversion policy", key)
					break
				}
			}
		}
		if err == nil {
			err = errors.Join(source.WithPolicy(ctx, nil, policy.Headers), guard(ctx))
		}
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		var observations []ServerFindObservation
		version := source.Client.Microversion
		if policy.Microversion != nil {
			version = *policy.Microversion
		} else if version == "" {
			discovery, e := microversions.Read(ctx, source, microversions.NovaProfile)
			for _, wire := range discovery.Responses {
				observations = append(observations, ServerFindObservation{Envelope: bytes.Clone(wire.Body), Header: wire.Header.Clone(), StatusCode: wire.StatusCode})
			}
			if e == nil {
				version, e = microversions.SelectVersion(discovery.Maximum, discovery.Minimum, "2.100")
			}
			if e = errors.Join(e, guard(ctx)); e != nil {
				yield(nil, wrap(e))
				return
			}
		}
		listSource := *source
		listSource.Client.MoreHeaders = maps.Clone(listSource.Client.MoreHeaders)
		for key := range listSource.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
				delete(listSource.Client.MoreHeaders, key)
			}
		}
		if err := errors.Join(listSource.WithPolicy(ctx, &version, policy.Headers), guard(ctx)); err != nil {
			yield(nil, wrap(err))
			return
		}
		headers := maps.Clone(listSource.Client.MoreHeaders)
		if headers == nil {
			headers = make(map[string]string)
		}
		headers["Accept"] = "application/json"
		for key, value := range policy.Headers {
			if strings.EqualFold(key, "Accept") {
				headers["Accept"] = value
			}
		}
		fixed := rest.RequestHeaderPolicy{Values: headers, Absent: []string{"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "X-OpenStack-Volume-API-Version", "X-OpenStack-Manila-API-Version", "X-OpenStack-Ironic-API-Version", "X-OpenStack-Ironic-Inspector-API-Version"}}
		if version != "" {
			headers["OpenStack-API-Version"] = "compute " + version
			headers["X-OpenStack-Nova-API-Version"] = version
		} else {
			fixed.Absent = append(fixed.Absent, "OpenStack-API-Version", "X-OpenStack-Nova-API-Version")
		}
		listSource.Client.MoreHeaders = maps.Clone(headers)
		path, versioned := "servers", "/v2.1/servers"
		if policy.Details == nil || *policy.Details {
			path += "/detail"
			versioned += "/detail"
		}
		var origin *rest.Response
		spec := rest.CollectionSpec[ServerFindRecord]{Client: &listSource.Client, Path: path, Kind: "servers", PluralKey: "servers", Metadata: serverFindMetadata, Validate: guard, SourceGuard: guard, ListCodes: serverFindCodes(200, 400), ReadPage: func(c context.Context, target string, codes ...int) (*rest.Response, error) {
			return rest.DoJSONGuardedRejectionsHeaderPolicy(c, &listSource.Client, guard, http.MethodGet, target, nil, fixed, rest.RejectionPolicy{Codes: serverFindCodes(400, 600), PreserveCleanRetry: true}, codes...)
		}, ValidateResponse: func(wire *rest.Response) error { origin = wire; return guard(ctx) }, ValidateItem: func(value *ServerFindRecord) error {
			value.Envelope = bytes.Clone(origin.Body)
			value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
			value.Microversion = version
			value.Discovery = make([]ServerFindObservation, len(observations))
			for i, o := range observations {
				value.Discovery[i] = ServerFindObservation{Envelope: bytes.Clone(o.Envelope), Header: o.Header.Clone(), StatusCode: o.StatusCode}
			}
			return errors.Join(serverFindProject(value, value.fields, location), guard(ctx))
		}, Paging: rest.PagePolicy[ServerFindRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, NextTruthy: serverFindTruthy, MarkerFallback: true, Marker: serverFindMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true, VersionedPath: versioned}}
		for value, err := range rest.ListWithControl(ctx, spec, listQuery, control) {
			if err != nil {
				yield(nil, wrap(err))
				return
			}
			matched := true
			for _, key := range serverFindKeys(policy) {
				filter, ok := filters[key]
				if !ok {
					continue
				}
				same, cause := serverFindLocalMatch(value.Resource.Body[key], filter)
				if cause = errors.Join(cause, guard(ctx)); cause != nil {
					yield(nil, wrap(origin.Fail(cause)))
					return
				}
				if !same {
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

// serverListPrepareQuery adapts the shared find query preparation to the
// Proxy.servers listing. Unlike find_server, whose positional identity owns
// "id", the listing transposes a caller "id" to the Nova "uuid" filter and
// prefers it over a raw "uuid" exactly like QueryParameters._transpose. The
// deprecated Proxy._list jmespath_filters argument is rejected before HTTP:
// Python applies jmespath.search to the lazy generator itself, which is not a
// row selection; the cloud SearchServers expression filter is the supported
// alternative. No automatic name hint is added.
func serverListPrepareQuery(policy RecordFindOpts) (url.Values, map[string]json.RawMessage, rest.ListControl, error) {
	owned := serverFindCopy(policy)
	if _, ok := owned.Query["jmespath_filters"]; ok {
		return nil, nil, rest.ListControl{}, serverFindInvalid("deprecated jmespath_filters is not a Proxy.servers query; use the cloud SearchServers expression filter")
	}
	id, hasID := owned.Query["id"]
	delete(owned.Query, "id")
	_, list, filters, _, control, err := serverFindPrepareQuery(owned, "")
	if err != nil {
		return nil, nil, control, err
	}
	if _, named := owned.Query["name"]; !named {
		list.Del("name")
	}
	if hasID {
		if !utf8.Valid(id) || !json.Valid(id) {
			return nil, nil, control, serverFindInvalid("query %q must contain UTF-8 JSON", "id")
		}
		values, err := cloudfilter.RequestQueryValues(id)
		if err != nil {
			return nil, nil, control, err
		}
		list.Del("uuid")
		if len(values) > 0 {
			list["uuid"] = slices.Clone(values)
		}
	}
	// Resource.list sends max_items as the limit when the limit is falsey and
	// checks the cap before constructing each further row, so a cap reached
	// exactly at a page boundary still requests the following page.
	if limit := control.MaxItems; limit > 0 {
		if values := list["limit"]; len(values) == 0 || len(values) == 1 && (values[0] == "0" || values[0] == "") {
			list.Set("limit", strconv.Itoa(limit))
		}
		control.MaxItems, control.LimitHint = 0, false
		control.BeforeItem = func(_ context.Context, consumed int) (bool, error) { return consumed >= limit, nil }
	}
	return list, filters, control, nil
}
