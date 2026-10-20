package subnets

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/neutronfind"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// CloudCreateOpts preserves the translated cloud keyword insertion order.
// Bulk map-only callers use a deterministic lexical order for remaining keys.
type CloudCreateOpts struct{ Order []string }
type CloudCreateOption func(*CloudCreateOpts) error

func WithCloudCreateOrder(keys ...string) CloudCreateOption {
	owned := slices.Clone(keys)
	return func(c *CloudCreateOpts) error { c.Order = slices.Clone(owned); return nil }
}
func subnetCloudKeys(fields map[string]json.RawMessage, order []string) []string {
	keys := make([]string, 0, len(fields))
	seen := make(map[string]bool, len(fields))
	for _, key := range order {
		if _, present := fields[key]; present && !seen[key] {
			keys = append(keys, key)
			seen[key] = true
		}
	}
	for _, key := range slices.Sorted(maps.Keys(fields)) {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	return keys
}
func subnetCloudFields(attrs map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	fields := make(map[string]json.RawMessage, len(attrs))
	for key, raw := range attrs {
		if raw == nil {
			raw = json.RawMessage("null")
		}
		if _, err := cloudfilter.PythonJSONTruthy(raw); err != nil {
			return nil, findInvalid("cloud subnet attribute %q requires complete UTF-8 JSON: %v", key, err)
		}
		fields[key] = bytes.Clone(raw)
	}
	return fields, nil
}
func subnetCloudHeader(raw json.RawMessage) (string, error) {
	trim := bytes.TrimSpace(raw)
	if len(trim) > 0 && trim[0] == '[' {
		var values []json.RawMessage
		if err := json.Unmarshal(trim, &values); err != nil {
			return "", err
		}
		stringsOnly := make([]string, len(values))
		for index, value := range values {
			if len(bytes.TrimSpace(value)) == 0 || bytes.TrimSpace(value)[0] != '"' {
				return "", findInvalid("if_match list members require strings")
			}
			if err := json.Unmarshal(value, &stringsOnly[index]); err != nil {
				return "", err
			}
		}
		return strings.Join(stringsOnly, ", "), nil
	}
	return cloudfilter.PythonString(raw)
}
func subnetCloudReceipt(value *neutronfind.Record, err error) *SubnetRecord {
	if value != nil {
		return &SubnetRecord{Resource: value.Resource, Wire: value.Wire, Envelope: value.Envelope, Header: value.Header, StatusCode: value.StatusCode, row: value.Row(), fromList: value.FromList()}
	}
	original := err
	var auth *gophercloud.ErrUnableToReauthenticate
	if errors.As(err, &auth) && auth != nil {
		original = auth.ErrOriginal
	}
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(original, &native) && native.Method == http.MethodPost {
		return &SubnetRecord{Envelope: bytes.Clone(native.Body), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
	}
	return nil
}
func subnetCloudPrepare(attrs map[string]json.RawMessage, order []string) (neutronfind.Profile, map[string]json.RawMessage, map[string]json.RawMessage, neutronfind.Options, error) {
	profile := recordFindProfile
	profile.Body = slices.Clone(profile.Body)
	profile.Headers = slices.Clone(profile.Headers)
	profile.Query = slices.Clone(profile.Query)
	profile.SourceFindPolicy, profile.SourceStatusPolicy, profile.SourceLinkPolicy, profile.PythonLoadedEquality, profile.StopFaultRetryCallbacks = true, true, true, true, true
	fields, err := subnetCloudFields(attrs)
	if err != nil {
		return profile, nil, nil, neutronfind.Options{}, err
	}
	for _, key := range []string{"self", "resource_type"} {
		if _, present := fields[key]; present {
			return profile, nil, nil, neutronfind.Options{}, findInvalid("cloud Proxy create collides with %q", key)
		}
	}
	// Proxy._create consumes base_path before restoring conflicting attributes.
	if raw, present := fields["base_path"]; present {
		if !bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			var path string
			if err := json.Unmarshal(raw, &path); err != nil {
				return profile, nil, nil, neutronfind.Options{}, findInvalid("base_path requires a string or null")
			}
			if strings.ContainsAny(path, "?#%\\") || strings.Contains(path, "://") {
				return profile, nil, nil, neutronfind.Options{}, findInvalid("base_path requires a literal service-relative route")
			}
			profile.Path = strings.Trim(path, "/")
			profile.VersionedPath = ""
		}
		delete(fields, "base_path")
	}
	if raw, present := fields["__conflicting_attrs"]; present {
		truth, err := cloudfilter.PythonJSONTruthy(raw)
		if err != nil {
			return profile, nil, nil, neutronfind.Options{}, err
		}
		if truth {
			members, err := cloudfilter.ObjectMembers(raw)
			if err != nil {
				return profile, nil, nil, neutronfind.Options{}, findInvalid("truthy __conflicting_attrs requires a JSON object")
			}
			for _, member := range members {
				if _, present := fields[member.Key]; !present {
					order = append(order, member.Key)
				}
				fields[member.Key] = bytes.Clone(member.Value)
			}
			delete(fields, "__conflicting_attrs")
		}
	}
	for _, key := range []string{"self", "cls", "connection", "_synchronized"} {
		if _, present := fields[key]; present {
			return profile, nil, nil, neutronfind.Options{}, findInvalid("new Subnet constructor collides with %q", key)
		}
	}
	delete(fields, "microversion") // Resource.create selects the adapter version.
	body, seed := make(map[string]json.RawMessage), make(map[string]json.RawMessage)
	policy := neutronfind.Options{Headers: make(map[string]string)}
	var headerRaw json.RawMessage
	headerPresent := false
	for _, key := range subnetCloudKeys(fields, order) {
		raw := fields[key]
		header := false
		for _, descriptor := range profile.Headers {
			if key == descriptor.Name || strings.EqualFold(key, descriptor.Wire) {
				headerRaw, headerPresent = bytes.Clone(raw), true
				header = true
			}
		}
		if header {
			continue
		}
		destination := key
		for _, descriptor := range profile.Body {
			if key == descriptor.Name || key == descriptor.Wire {
				destination = descriptor.Wire
			}
		}
		body[destination] = bytes.Clone(raw)
		seed[destination] = bytes.Clone(raw)
	}
	if headerPresent {
		value, err := subnetCloudHeader(headerRaw)
		if err != nil {
			return profile, nil, nil, policy, err
		}
		policy.Headers["If-Match"] = value
		seed["if_match"] = bytes.Clone(headerRaw)
	}
	// ProjectSeed consumes canonical names. Keep the raw dirty wire map separate
	// from the getter seed, and never cast values placed in the POST Body.
	for _, descriptor := range profile.Body {
		if descriptor.Name != descriptor.Wire {
			if raw, present := body[descriptor.Wire]; present {
				seed[descriptor.Name] = bytes.Clone(raw)
			}
			delete(seed, descriptor.Wire)
		}
	}
	return profile, body, seed, policy, nil
}

// CreateCloudRecord implements the translated create_subnet dirty/seed lane.
// It leaves native Create/CreateWithAttributes and baseline Find unchanged.
func (a *API) CreateCloudRecord(ctx context.Context, attrs map[string]json.RawMessage, options ...CloudCreateOption) (*SubnetRecord, error) {
	wrap := func(err error) error {
		return request.Wrap("CreateCloudRecord", "subnets", cloudread.ContextError(ctx, err))
	}
	if err := cloudread.Context(ctx); err != nil {
		return nil, wrap(err)
	}
	if a == nil {
		return nil, wrap(findInvalid("configured subnets API required"))
	}
	selected := a.client
	source, err := cloudread.Capture(ctx, selected, "network")
	if err != nil {
		return nil, wrap(err)
	}
	var changed error
	guard := func(ctx context.Context) error {
		if changed == nil && a.client != selected {
			changed = findInvalid("selected subnets client changed")
		}
		return errors.Join(changed, source.Guard(ctx))
	}
	ownedAttrs, err := subnetCloudFields(attrs)
	if err != nil {
		return nil, wrap(err)
	}
	cfg := CloudCreateOpts{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(findInvalid("nil cloud subnet create option"))
		}
		next := CloudCreateOpts{Order: slices.Clone(cfg.Order)}
		if err := errors.Join(apply(&next), guard(ctx)); err != nil {
			return nil, wrap(err)
		}
		cfg = CloudCreateOpts{Order: slices.Clone(next.Order)}
	}
	profile, body, seed, policy, err := subnetCloudPrepare(ownedAttrs, cfg.Order)
	if err != nil {
		return nil, wrap(err)
	}
	engine, err := neutronfind.New(ctx, &source.Client, profile, guard)
	if err != nil {
		return nil, wrap(err)
	}
	if _, err := engine.ProjectSeed(ctx, seed, policy); err != nil {
		return nil, wrap(err)
	}
	value, err := engine.Mutate(ctx, http.MethodPost, "", body, seed, policy)
	record := subnetCloudReceipt(value, err)
	if record != nil && record.Resource != nil {
		if _, present := record.Resource.Body["base_path"]; present {
			record.Resource.Body["base_path"] = json.RawMessage(`"/subnets"`)
		}
	}
	finalErr := errors.Join(err, engine.Check(ctx))
	if finalErr != nil && record != nil && record.StatusCode >= 100 && record.StatusCode < 400 {
		record.Resource = nil
		finalErr = &resource.ResponseError{StatusCode: record.StatusCode, Header: record.Header.Clone(), Body: bytes.Clone(record.Envelope), Cause: finalErr}
	}
	return record, wrap(finalErr)
}
