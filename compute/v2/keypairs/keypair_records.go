package keypairs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"maps"
	"net/url"
	"strconv"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/microversions"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// Keep list-row decoding private so KeypairRecord retains its existing JSON
// behavior for creation and returned record values.
type keypairListRecord struct{ KeypairRecord }

// UnmarshalJSON retains a complete list row, including its keypair wrapper.
// The metadata pointer supplied by the REST decoder stays stable.
func (value *keypairListRecord) UnmarshalJSON(data []byte) error {
	if value == nil {
		return fmt.Errorf("%w: keypair record receiver is required", resource.ErrInvalidOption)
	}
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	if err := json.Unmarshal(data, value.Wire); err != nil {
		return err
	}
	value.Envelope = bytes.Clone(data)
	value.Resource = nil
	return nil
}

func keypairRecordMetadata(value *KeypairRecord) *resource.Metadata {
	if value.Wire == nil {
		value.Wire = &resource.RawResource{}
	}
	return &value.Wire.Metadata
}

// Only list construction merges the nested object onto the outer row. Fetch
// translation selects its response envelope, matching Resource.fetch.
func keypairListFields(wire *resource.RawResource) (map[string]json.RawMessage, error) {
	if wire == nil || wire.Body == nil {
		return nil, fmt.Errorf("%w: keypair response fields are required", resource.ErrInvalidOption)
	}
	fields := maps.Clone(wire.Body)
	if raw, present := fields["keypair"]; present {
		var nested resource.RawResource
		if err := json.Unmarshal(raw, &nested); err != nil {
			return nil, err
		}
		delete(fields, "keypair")
		maps.Copy(fields, nested.Body)
	}
	return normalizedKeypairCreateFields(fields), nil
}

func projectKeypairRecord(fields map[string]json.RawMessage, metadata resource.Metadata) (*resource.RawResource, error) {
	view := &resource.RawResource{Metadata: resource.Metadata{Body: make(map[string]json.RawMessage, 9), Header: metadata.Header.Clone(), StatusCode: metadata.StatusCode}}
	for _, key := range []string{"created_at", "fingerprint", "name", "private_key", "public_key", "type", "user_id"} {
		raw, err := resource.BodyRecordField(fields, key, resource.BodyFieldJSON)
		if err != nil {
			return nil, err
		}
		if key == "type" {
			if _, present := fields[key]; !present {
				raw = json.RawMessage(`"ssh"`)
			}
		}
		view.Body[key] = raw
	}
	view.Body["id"] = bytes.Clone(view.Body["name"])
	var err error
	view.Body["is_deleted"], err = resource.BodyRecordField(fields, "deleted", resource.BodyFieldBoolean)
	if err != nil {
		return nil, err
	}
	return view, nil
}

func prepareKeypairListRecord(value *KeypairRecord) error {
	fields, err := keypairListFields(value.Wire)
	if err != nil {
		return err
	}
	value.Header, value.StatusCode = value.Wire.Header.Clone(), value.Wire.StatusCode
	value.Resource, err = projectKeypairRecord(fields, value.Wire.Metadata)
	return err
}

func keypairRecordMarker(value *KeypairRecord) (string, error) {
	fields, err := keypairListFields(value.Wire)
	if err != nil {
		return "", err
	}
	var marker string
	if err := json.Unmarshal(fields["name"], &marker); err != nil {
		return "", fmt.Errorf("%w: keypair marker must be a string name: %w", resource.ErrInvalidOption, err)
	}
	if strings.TrimSpace(marker) == "" {
		return "", fmt.Errorf("%w: keypair marker must be nonempty", resource.ErrInvalidOption)
	}
	return marker, nil
}

func keypairRecordFilterValue(value *KeypairRecord, field string) (json.RawMessage, error) {
	if value == nil || value.Resource == nil {
		return nil, fmt.Errorf("%w: keypair Resource is required", resource.ErrInvalidOption)
	}
	return resource.BodyRecordField(value.Resource.Body, field, resource.BodyFieldJSON)
}

func keypairRecordSource(ctx context.Context, a *API) (*cloudread.Source, func(context.Context) error, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, nil, err
	}
	var original *gophercloud.ServiceClient
	if a != nil {
		original = a.client
	}
	source, err := cloudread.Capture(ctx, original, "compute")
	if err != nil {
		return nil, nil, err
	}
	var changed error
	guard := func(checkCtx context.Context) error {
		if changed == nil && (a == nil || a.client != original) {
			changed = fmt.Errorf("%w: keypair API source changed", resource.ErrInvalidOption)
		}
		return errors.Join(changed, source.Guard(checkCtx))
	}
	return source, guard, nil
}

func keypairRecordCodes() []int {
	codes := make([]int, 200)
	for index := range codes {
		codes[index] = index + 200
	}
	return codes
}

// Source facts are captured before options. The operation guard composes those
// facts with the caller's guard without recursing through the discovery reader.
func selectKeypairReadVersion(ctx context.Context, source *cloudread.Source, override *string, headers map[string]string) (string, error) {
	// Validate caller headers before discovery can perform HTTP. When the
	// version is not known yet, defer only its two version-header comparisons.
	trial := *source
	trial.Client.MoreHeaders = maps.Clone(source.Client.MoreHeaders)
	preflightHeaders := maps.Clone(headers)
	for key := range trial.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
			delete(trial.Client.MoreHeaders, key)
		}
	}
	if override == nil && source.Client.Microversion == "" {
		for key := range preflightHeaders {
			if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
				delete(preflightHeaders, key)
			}
		}
	}
	if err := trial.WithPolicy(ctx, override, preflightHeaders); err != nil {
		return "", err
	}
	chosen := source.Client.Microversion
	if override != nil {
		chosen = *override
	} else if chosen == "" {
		advertised, err := microversions.Read(ctx, source, microversions.NovaProfile)
		if err != nil {
			return "", err
		}
		chosen, err = microversions.SelectVersion(advertised.Maximum, advertised.Minimum, "2.10")
		if err != nil {
			err = fmt.Errorf("%w: keypair version selection: %w", resource.ErrInvalidOption, err)
			if len(advertised.Responses) != 0 {
				err = advertised.Responses[len(advertised.Responses)-1].Fail(cloudread.ContextError(ctx, err))
			}
			return "", err
		}
	}
	// Captured source headers are passive input for the private chosen version;
	// inherited version headers must not defeat an explicit empty override.
	for key := range source.Client.MoreHeaders {
		if strings.EqualFold(key, "OpenStack-API-Version") || strings.EqualFold(key, "X-OpenStack-Nova-API-Version") {
			delete(source.Client.MoreHeaders, key)
		}
	}
	if err := source.WithPolicy(ctx, &chosen, headers); err != nil {
		return "", err
	}
	return chosen, rest.CheckOperationGuard(ctx)
}

func validateKeypairListQuery(query url.Values) error {
	if values, present := query["limit"]; present {
		if len(values) != 1 {
			return fmt.Errorf("%w: keypair limit must be one positive integer", resource.ErrInvalidOption)
		}
		limit, err := strconv.Atoi(values[0])
		if err != nil || limit < 1 {
			return fmt.Errorf("%w: keypair limit must be positive", resource.ErrInvalidOption)
		}
	}
	if values, present := query["marker"]; present && (len(values) != 1 || strings.TrimSpace(values[0]) == "") {
		return fmt.Errorf("%w: keypair marker must be one nonempty value", resource.ErrInvalidOption)
	}
	return nil
}

func listKeypairRecordsPrepared(ctx context.Context, source *cloudread.Source, guard func(context.Context) error, parameters keypairListParameters, versionReady bool) iter.Seq2[*KeypairRecord, error] {
	descriptor := keypairRecordFilterDescriptor()
	collection := resource.NewCollection(resource.Adapter[KeypairRecord]{
		Kind: "keypairs", FilterDescriptor: descriptor, BodyFilterFields: descriptor.Body, BodyFilterValue: keypairRecordFilterValue,
		IterateControlled: func(ctx context.Context, semantic url.Values, _ resource.ListControl) iter.Seq2[*KeypairRecord, error] {
			return func(yield func(*KeypairRecord, error) bool) {
				query := maps.Clone(parameters.query)
				for key, values := range semantic {
					if _, exists := query[key]; exists {
						yield(nil, fmt.Errorf("%w: semantic query %q conflicts with typed/raw query", resource.ErrInvalidOption, key))
						return
					}
					query[key] = append([]string(nil), values...)
				}
				if err := validateKeypairListQuery(query); err != nil {
					yield(nil, err)
					return
				}
				chosen := source.Client.Microversion
				if !versionReady {
					var err error
					chosen, err = selectKeypairReadVersion(ctx, source, parameters.microversion, parameters.headers)
					if err != nil {
						yield(nil, err)
						return
					}
				}
				var origin *rest.Response
				check := func(ctx context.Context) error { return errors.Join(guard(ctx), rest.CheckOperationGuard(ctx)) }
				selected := rest.CollectionSpec[keypairListRecord]{
					Client: &source.Client, Path: "os-keypairs", Kind: "keypairs", PluralKey: "keypairs", Metadata: func(value *keypairListRecord) *resource.Metadata { return keypairRecordMetadata(&value.KeypairRecord) },
					Validate: guard, SourceGuard: guard, ListCodes: keypairRecordCodes(),
					ValidateResponse: func(response *rest.Response) error { origin = response; return check(ctx) },
					ValidateItem: func(value *keypairListRecord) error {
						if err := check(ctx); err != nil {
							return err
						}
						return errors.Join(prepareKeypairListRecord(&value.KeypairRecord), check(ctx))
					},
					ReadPage: func(ctx context.Context, target string, codes ...int) (*rest.Response, error) {
						return microversions.MemberGet(ctx, source, target, chosen, microversions.NovaProfile, codes...)
					},
					Paging: rest.PagePolicy[keypairListRecord]{LinkKeys: []string{"links", "keypairs_links"}, NextKey: "next", HTTPLink: true, DictionaryLinks: true,
						MarkerFallback: true, Marker: func(value *keypairListRecord) (string, error) { return keypairRecordMarker(&value.KeypairRecord) }, MarkerOnShortPage: true, AllowFirstServerLimit: true,
						MaxItemsLimitHint: true, StopOnEmptyPage: true, SingletonObject: true, DecodeNoContent: true},
				}
				for value, err := range rest.ListWithControl(ctx, selected, query, parameters.control) {
					var record *KeypairRecord
					if value != nil {
						record = &value.KeypairRecord
					}
					if !yield(record, err) {
						return
					}
					if err == nil {
						if err := check(ctx); err != nil {
							yield(nil, origin.Fail(cloudread.ContextError(ctx, err)))
							return
						}
					}
				}
			}
		},
	})
	return collection.List(ctx, parameters.filters...)
}

// ListRecords lazily lists source-shaped keypairs while retaining actual row
// JSON. Every iteration captures its source and applies options once. Break
// stops further requests. The native List and All methods keep their ABI.
func (a *API) ListRecords(ctx context.Context, options ...KeypairListOption) iter.Seq2[*KeypairRecord, error] {
	owned := append([]KeypairListOption(nil), options...)
	return func(yield func(*KeypairRecord, error) bool) {
		wrap := func(err error) error {
			return request.Wrap("ListRecords", "keypairs", cloudread.ContextError(ctx, err))
		}
		source, guard, err := keypairRecordSource(ctx, a)
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		operationCtx := rest.WithOperationGuard(ctx, guard)
		check := func(checkCtx context.Context) error {
			return errors.Join(guard(checkCtx), rest.CheckOperationGuard(ctx))
		}
		parameters, err := prepareKeypairList(ctx, check, owned)
		if err == nil {
			err = check(ctx)
		}
		if err != nil {
			yield(nil, wrap(err))
			return
		}
		if parameters.headers == nil {
			parameters.headers = make(map[string]string)
		}
		if !hasKeypairHeader(parameters.headers, "Accept") {
			parameters.headers["Accept"] = "application/json"
		}
		for value, err := range listKeypairRecordsPrepared(operationCtx, source, guard, parameters, false) {
			if !yield(value, wrap(err)) {
				return
			}
		}
	}
}

func hasKeypairHeader(headers map[string]string, target string) bool {
	for key := range headers {
		if strings.EqualFold(key, target) {
			return true
		}
	}
	return false
}
