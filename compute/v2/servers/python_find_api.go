package servers

import (
	"bytes"
	"context"
	"errors"
	"iter"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlocation"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func serverFindSegment(id string) error {
	if strings.TrimSpace(id) == "" || id == "." || id == ".." || !utf8.ValidString(id) || strings.IndexFunc(id, unicode.IsControl) >= 0 {
		return serverFindInvalid("identity must be nonblank literal UTF-8 without controls")
	}
	return nil
}
func serverFindCodes(start, end int) []int {
	out := make([]int, end-start)
	for i := range out {
		out[i] = i + start
	}
	return out
}

// FindRecord maps Source find_server: GET first, details enabled, scope local,
// and ignored logical absence only after a complete successful fallback list.
func (a *API) FindRecord(ctx context.Context, identity string, options ...RecordFindOption) (*ServerFindRecord, error) {
	wrap := func(err error) error { return request.Wrap("FindRecord", "servers", cloudread.ContextError(ctx, err)) }
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(serverFindInvalid("server API required"))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "compute")
	if err != nil {
		return nil, wrap(err)
	}
	listSource := *source
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
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(serverFindInvalid("nil find option"))
		}
		next := serverFindCopy(policy)
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		policy = serverFindCopy(next)
	}
	getQuery, listQuery, filters, seed, control, err := serverFindPrepareQuery(policy, identity)
	if err == nil {
		err = serverFindSegment(identity)
	}
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
	// Named ordinary Headers cannot impersonate auth, framing or protocol; version
	// selection belongs to the captured service or explicit microversion option.
	if err == nil {
		for key := range policy.Headers {
			if strings.EqualFold(key, "OpenStack-API-Version") || strings.HasPrefix(strings.ToLower(key), "x-openstack-") && strings.HasSuffix(strings.ToLower(key), "-api-version") {
				err = serverFindInvalid("protocol Header %q requires the owned microversion policy", key)
				break
			}
		}
	}
	if err == nil {
		err = errors.Join(source.WithPolicy(ctx, nil, policy.Headers), listSource.WithPolicy(ctx, nil, policy.Headers), guard(ctx))
	}
	if err != nil {
		return nil, wrap(err)
	}
	var observations []ServerFindObservation
	baseReady := false
	baseVersion := ""
	var baseErr error
	base := func() (string, error) {
		if baseReady {
			return baseVersion, baseErr
		}
		baseReady = true
		baseVersion = listSource.Client.Microversion
		if baseVersion == "" {
			discovery, e := microversions.Read(ctx, &listSource, microversions.NovaProfile)
			for _, wire := range discovery.Responses {
				observations = append(observations, ServerFindObservation{Envelope: bytes.Clone(wire.Body), Header: wire.Header.Clone(), StatusCode: wire.StatusCode})
			}
			baseErr = e
			if baseErr == nil {
				baseVersion, baseErr = microversions.SelectVersion(discovery.Maximum, discovery.Minimum, "2.100")
			}
		}
		baseErr = errors.Join(baseErr, guard(ctx))
		return baseVersion, baseErr
	}
	memberVersion := ""
	if policy.Microversion != nil {
		memberVersion = *policy.Microversion
	} else {
		memberVersion, err = base()
	}
	if err != nil {
		return nil, wrap(err)
	}
	phase := func(list bool) (*cloudread.Source, rest.RequestHeaderPolicy, string, error) {
		selectedSource := *source
		version := memberVersion
		if list {
			selectedSource = listSource
			var err error
			version, err = base()
			if err != nil {
				return nil, rest.RequestHeaderPolicy{}, "", err
			}
		}
		selectedSource.Client.MoreHeaders = maps.Clone(selectedSource.Client.MoreHeaders)
		for key := range selectedSource.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
				delete(selectedSource.Client.MoreHeaders, key)
			}
		}
		if err := errors.Join(selectedSource.WithPolicy(ctx, &version, policy.Headers), guard(ctx)); err != nil {
			return nil, rest.RequestHeaderPolicy{}, "", err
		}
		headers := maps.Clone(selectedSource.Client.MoreHeaders)
		if headers == nil {
			headers = make(map[string]string)
		}
		if list {
			headers["Accept"] = "application/json"
			for key, value := range policy.Headers {
				if strings.EqualFold(key, "Accept") {
					headers["Accept"] = value
				}
			}
		}
		fixed := rest.RequestHeaderPolicy{Values: headers, Absent: []string{"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "X-OpenStack-Volume-API-Version", "X-OpenStack-Manila-API-Version", "X-OpenStack-Ironic-API-Version", "X-OpenStack-Ironic-Inspector-API-Version"}}
		if version != "" {
			headers["OpenStack-API-Version"] = "compute " + version
			headers["X-OpenStack-Nova-API-Version"] = version
		} else {
			fixed.Absent = append(fixed.Absent, "OpenStack-API-Version", "X-OpenStack-Nova-API-Version")
		}
		selectedSource.Client.MoreHeaders = maps.Clone(headers)
		return &selectedSource, fixed, version, nil
	}
	memberSource, memberHeaders, _, err := phase(false)
	if err != nil {
		return nil, wrap(err)
	}
	receipt := func(value *ServerFindRecord, version string) *ServerFindRecord {
		if value == nil {
			return nil
		}
		value.Microversion = version
		value.Discovery = make([]ServerFindObservation, len(observations))
		for i, o := range observations {
			value.Discovery[i] = ServerFindObservation{Envelope: bytes.Clone(o.Envelope), Header: o.Header.Clone(), StatusCode: o.StatusCode}
		}
		return value
	}
	var failedMember *ServerFindRecord
	get := func(_ context.Context, id string) (*ServerFindRecord, error) {
		target := memberSource.Client.ServiceURL("servers", url.PathEscape(id))
		if len(getQuery) > 0 {
			target += "?" + getQuery.Encode()
		}
		wire, err := rest.DoJSONGuardedRejectionsHeaderPolicy(ctx, &memberSource.Client, guard, http.MethodGet, target, nil, memberHeaders, rest.RejectionPolicy{Codes: serverFindCodes(400, 600), PreserveCleanRetry: true}, serverFindCodes(200, 400)...)
		if err != nil {
			failedMember = receipt(serverFindReceipt(wire), memberVersion)
			return failedMember, err
		}
		value, err := serverFindResponse(wire, seed, location)
		if err = errors.Join(err, guard(ctx)); err != nil {
			failedMember = receipt(value, memberVersion)
			return failedMember, wire.Fail(err)
		}
		return receipt(value, memberVersion), nil
	}
	details := policy.Details == nil || *policy.Details
	list := func(context.Context, url.Values) iter.Seq2[*ServerFindRecord, error] {
		return func(yield func(*ServerFindRecord, error) bool) {
			listSource, headers, version, err := phase(true)
			if err != nil {
				yield(nil, err)
				return
			}
			path, versioned := "servers", "/v2.1/servers"
			if details {
				path += "/detail"
				versioned += "/detail"
			}
			var origin *rest.Response
			spec := rest.CollectionSpec[ServerFindRecord]{Client: &listSource.Client, Path: path, Kind: "servers", PluralKey: "servers", Metadata: serverFindMetadata, Validate: guard, SourceGuard: guard, ListCodes: serverFindCodes(200, 400), ReadPage: func(c context.Context, target string, codes ...int) (*rest.Response, error) {
				return rest.DoJSONGuardedRejectionsHeaderPolicy(c, &listSource.Client, guard, http.MethodGet, target, nil, headers, rest.RejectionPolicy{Codes: serverFindCodes(400, 600), PreserveCleanRetry: true}, codes...)
			}, ValidateResponse: func(wire *rest.Response) error { origin = wire; return guard(ctx) }, ValidateItem: func(value *ServerFindRecord) error {
				value.Envelope = bytes.Clone(origin.Body)
				value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
				receipt(value, version)
				return errors.Join(serverFindProject(value, value.fields, location), guard(ctx))
			}, Paging: rest.PagePolicy[ServerFindRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, NextTruthy: serverFindTruthy, MarkerFallback: true, Marker: serverFindMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true, VersionedPath: versioned}}
			for value, err := range rest.ListWithControl(ctx, spec, listQuery, control) {
				if err != nil {
					yield(nil, err)
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
						yield(nil, origin.Fail(cause))
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
	collection := resource.NewCollection(resource.Adapter[ServerFindRecord]{Kind: "servers", IdentityFind: true, ValidateID: serverFindSegment, IdentityDirectGet: serverFindSegment, IdentityResponseID: func(value *ServerFindRecord) (string, error) { return serverFindText(value, "id"), nil }, Name: func(value *ServerFindRecord) string { return serverFindText(value, "name") }, Get: get, Iterate: list})
	value, err := collection.FindIdentity(ctx, identity, resource.WithIdentityFindIgnoreMissing(policy.IgnoreMissing == nil || *policy.IgnoreMissing))
	if err != nil && failedMember != nil {
		return failedMember, wrap(errors.Join(err, guard(ctx)))
	}
	return value, wrap(errors.Join(err, guard(ctx)))
}
