package images

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
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudlocation"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func taskRecordCodes(start, end int) []int {
	out := make([]int, end-start)
	for i := range out {
		out[i] = start + i
	}
	return out
}

// The catalog Glance major API /v2 may advertise API IDs v2.17. Unlike Nova's
// fixed API identity row, Glance matches the major and uses optional advertised
// microversion bounds, never the API minor ID as a microversion maximum.
func taskRecordProfile() microversions.Profile {
	return microversions.Profile{Major: "2", HeaderService: "image", RequireSelfLink: true, OkCodes: taskRecordCodes(200, 400)}
}

// TaskRecords maps the image-scoped Source ImageTasks list, with literal parent
// selector, Source default local image_id filter and independently owned rows.
func (a *API) TaskRecords(ctx context.Context, imageID string, options ...TaskRecordsOption) iter.Seq2[*ImageTaskRecord, error] {
	options = slices.Clone(options)
	return func(yield func(*ImageTaskRecord, error) bool) {
		fail := func(err error) {
			yield(nil, request.Wrap("TaskRecords", "image_tasks", cloudread.ContextError(ctx, err)))
		}
		if err := cloudread.Context(ctx); err != nil {
			fail(err)
			return
		}
		if a == nil {
			fail(taskRecordInvalid("image API required"))
			return
		}
		selected := a.client
		source, err := cloudread.Capture(ctx, selected, "image")
		if err != nil {
			fail(err)
			return
		}
		var changed error
		ownGuard := func(c context.Context) error {
			if changed == nil && a.client != selected {
				changed = taskRecordInvalid("selected image API client changed")
			}
			return errors.Join(changed, source.Guard(c))
		}
		operationCtx := rest.WithOperationGuard(ctx, ownGuard)
		guard := func(c context.Context) error { return errors.Join(ownGuard(c), rest.CheckOperationGuard(c)) }
		policy := TaskRecordsOpts{}
		for _, apply := range options {
			if err := guard(operationCtx); err != nil {
				fail(err)
				return
			}
			if apply == nil {
				fail(taskRecordInvalid("nil task records option"))
				return
			}
			next := taskRecordCopy(policy)
			if err := errors.Join(apply(&next), guard(operationCtx)); err != nil {
				fail(err)
				return
			}
			policy = taskRecordCopy(next)
		}
		query, filters, control, err := taskRecordPrepare(policy, imageID)
		if err == nil {
			err = taskRecordIdentity(imageID)
		}
		location := resource.CloudLocation{}
		if err == nil {
			if policy.Location != nil {
				location = policy.Location.Clone()
			} else {
				location.Project.ID, err = cloudlocation.ProjectID(selected.ProviderClient)
			}
		}
		var locationJSON json.RawMessage
		if err == nil {
			locationJSON, err = json.Marshal(location)
		}
		if err == nil {
			for key := range policy.Headers {
				if strings.EqualFold(key, "OpenStack-API-Version") || strings.HasPrefix(strings.ToLower(key), "x-openstack-") && strings.HasSuffix(strings.ToLower(key), "-api-version") {
					err = taskRecordInvalid("Header %q is an owned version policy", key)
					break
				}
			}
		}
		// Validate explicit operation headers before discovery without injecting them
		// into its adapter request. Source list headers apply only to task-list GETs.
		validation := *source
		if err == nil {
			err = errors.Join(validation.WithPolicy(operationCtx, nil, policy.Headers), guard(operationCtx))
		}
		if err != nil {
			fail(err)
			return
		}
		version := source.Client.Microversion
		var observations []TaskRecordObservation
		if policy.Microversion != nil {
			version = *policy.Microversion
		} else if version == "" {
			discovery, e := microversions.Read(operationCtx, source, taskRecordProfile())
			for _, wire := range discovery.Responses {
				observations = append(observations, TaskRecordObservation{Envelope: bytes.Clone(wire.Body), Header: wire.Header.Clone(), StatusCode: wire.StatusCode})
			}
			err = e
			if err == nil {
				version, err = microversions.SelectVersion(discovery.Maximum, discovery.Minimum, "2.17")
			}
		}
		if err = errors.Join(err, guard(operationCtx)); err != nil {
			fail(err)
			return
		}
		source.Client.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
		for key := range source.Client.MoreHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") {
				delete(source.Client.MoreHeaders, key)
			}
		}
		if err := errors.Join(source.WithPolicy(operationCtx, &version, policy.Headers), guard(operationCtx)); err != nil {
			fail(err)
			return
		}
		headers := maps.Clone(source.Client.MoreHeaders)
		if headers == nil {
			headers = make(map[string]string)
		}
		headers["Accept"] = "application/json"
		for key, value := range policy.Headers {
			if strings.EqualFold(key, "Accept") {
				headers["Accept"] = value
			}
		}
		fixed := rest.RequestHeaderPolicy{Values: headers, Absent: []string{"Authorization", "X-Service-Token", "Host", "Cookie", "Content-Length", "Transfer-Encoding", "Connection", "Trailer", "Te", "Upgrade", "X-OpenStack-Nova-API-Version", "X-OpenStack-Volume-API-Version", "X-OpenStack-Manila-API-Version", "X-OpenStack-Ironic-API-Version", "X-OpenStack-Ironic-Inspector-API-Version"}}
		if version != "" {
			headers["OpenStack-API-Version"] = "image " + version
		} else {
			fixed.Absent = append(fixed.Absent, "OpenStack-API-Version")
		}
		source.Client.MoreHeaders = maps.Clone(headers)
		var origin *rest.Response
		path := "images/" + url.PathEscape(imageID) + "/tasks"
		spec := rest.CollectionSpec[ImageTaskRecord]{Client: &source.Client, Path: path, Kind: "image_tasks", PluralKey: "tasks", Metadata: taskRecordMetadata, Validate: guard, SourceGuard: guard, ListCodes: taskRecordCodes(200, 400), ReadPage: func(c context.Context, target string, codes ...int) (*rest.Response, error) {
			return rest.DoJSONGuardedRejectionsHeaderPolicy(c, &source.Client, guard, http.MethodGet, target, nil, fixed, rest.RejectionPolicy{Codes: taskRecordCodes(400, 600), PreserveCleanRetry: true}, codes...)
		}, ValidateResponse: func(wire *rest.Response) error { origin = wire; return guard(operationCtx) }, ValidateItem: func(value *ImageTaskRecord) error {
			value.ImageID = imageID
			value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
			value.Envelope = bytes.Clone(origin.Body)
			value.Microversion = version
			value.Discovery = make([]TaskRecordObservation, len(observations))
			for i, wire := range observations {
				value.Discovery[i] = TaskRecordObservation{Envelope: bytes.Clone(wire.Envelope), Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
			}
			return errors.Join(taskRecordProject(value, locationJSON), guard(operationCtx))
		}, Paging: rest.PagePolicy[ImageTaskRecord]{HTTPLink: true, DictionaryLinks: true, IgnoreFalseyNext: true, NextTruthy: taskRecordTruthy, MarkerFallback: true, Marker: taskRecordMarker, MarkerOnShortPage: true, AllowFirstServerLimit: true, AllowZeroLimit: true, MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true, VersionedPath: "/v2/" + path}}
		filterKeys := append([]string{"image_id"}, taskRecordKeys(policy)...)
		for value, err := range rest.ListWithControl(operationCtx, spec, query, control) {
			if err != nil {
				fail(err)
				return
			}
			matched := true
			for _, key := range filterKeys {
				wanted, ok := filters[key]
				if !ok {
					continue
				}
				same, cause := taskRecordLocalMatch(value.Resource.Body[key], wanted)
				if cause = errors.Join(cause, guard(operationCtx)); cause != nil {
					fail(origin.Fail(cause))
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

// AllTaskRecords returns a nonnil empty slice on success and discards partial
// collected records on error; iterator consumers retain already yielded rows.
func (a *API) AllTaskRecords(ctx context.Context, imageID string, options ...TaskRecordsOption) ([]*ImageTaskRecord, error) {
	values := make([]*ImageTaskRecord, 0)
	for value, err := range a.TaskRecords(ctx, imageID, options...) {
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}
