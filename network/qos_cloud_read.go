package network

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	networkv2 "github.com/JSYoo5B/go-openstacksdk/network/v2"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ErrUnavailableExtension identifies an absent advertised extension. A cloud
// read still retains the complete, actually observed extension inventory.
var ErrUnavailableExtension = errors.New("OpenStack extension is unavailable")

type UnavailableExtensionError struct{ Alias string }

func (e *UnavailableExtensionError) Error() string {
	return fmt.Sprintf("network extension %q: %v", e.Alias, ErrUnavailableExtension)
}
func (e *UnavailableExtensionError) Unwrap() error { return ErrUnavailableExtension }

// qosReadState is owned by a single SDK cloud operation. The cloned API freezes
// request headers; its guard observes the original selected service and client.
// It deliberately does not cache extension results or stop after an early match:
// Source builds the complete alias set afresh for each _has_neutron_extension.
type qosReadState struct {
	API            *networkv2.Service
	Extensions     []*extensions.ExtensionRecord
	service        *Service
	selectedAPI    *networkv2.Service
	source         *cloudread.Source
	originalClient *gophercloud.ServiceClient
}

func qosReadInvalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", resource.ErrInvalidOption, fmt.Sprintf(format, args...))
}

// Check is source-only. It must not invoke rest.CheckOperationGuard because the
// SDK also registers it as an outer guard for the dependent leaf HTTP operation.
func (s *qosReadState) Check(ctx context.Context) error {
	if err := cloudread.Context(ctx); err != nil {
		return err
	}
	if s == nil || s.source == nil || s.service == nil {
		return qosReadInvalid("configured QoS read state is required")
	}
	if s.service.client != s.originalClient || s.service.API != s.selectedAPI || s.selectedAPI.RawClient() != s.service.client {
		return qosReadInvalid("selected network service or API changed")
	}
	return s.source.Guard(ctx)
}

func qosReadStart(ctx context.Context, service *Service, aliases ...string) (*qosReadState, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if service == nil || service.API == nil || service.client == nil || service.API.RawClient() != service.client {
		return nil, qosReadInvalid("configured network service and API are required")
	}
	source, err := cloudread.Capture(ctx, service.client, "network")
	if err != nil {
		return nil, err
	}
	state := &qosReadState{service: service, selectedAPI: service.API, source: source, originalClient: service.client}
	state.API = networkv2.New(&source.Client)
	ctx = rest.WithOperationGuard(ctx, state.Check)
	for _, alias := range aliases {
		if err := state.Check(ctx); err != nil {
			return state, err
		}
		found := false
		for record, failure := range state.API.Extensions.ListRecords(ctx) {
			if record != nil {
				state.Extensions = append(state.Extensions, record)
			}
			if err := errors.Join(failure, state.Check(ctx)); err != nil {
				return state, err
			}
			if record == nil || record.Resource == nil {
				return state, qosReadInvalid("extension list yielded no Resource")
			}
			raw := record.Resource.Body["alias"]
			var value any
			if len(raw) != 0 {
				if err := json.Unmarshal(raw, &value); err != nil {
					return state, qosExtensionFailure(record, err)
				}
			}
			switch value.(type) {
			case map[string]any, []any:
				// Even a later unhashable alias makes Source set() fail after
				// an earlier requested alias matched. Never hide that failure.
				return state, qosExtensionFailure(record, qosReadInvalid("unhashable extension alias"))
			}
			if value == alias {
				found = true
			}
		}
		if err := state.Check(ctx); err != nil {
			return state, err
		}
		if !found {
			return state, &UnavailableExtensionError{Alias: alias}
		}
	}
	return state, nil
}

func qosExtensionFailure(record *extensions.ExtensionRecord, err error) error {
	response := &rest.Response{Body: append(json.RawMessage(nil), record.Envelope...), Header: record.Header.Clone(), StatusCode: record.StatusCode}
	return response.Fail(err)
}

// Policy follows the fixed cloud helper lookup; it does not repeat the QoS
// extension gate. The minimum-bandwidth getter uses the strict Source variant.
func (s *qosReadState) Policy(ctx context.Context, identity string) (*policies.PolicyRecord, error) {
	return s.policy(ctx, identity, false)
}
func (s *qosReadState) PolicyStrict(ctx context.Context, identity string) (*policies.PolicyRecord, error) {
	return s.policy(ctx, identity, true)
}
func (s *qosReadState) policy(ctx context.Context, identity string, strict bool) (*policies.PolicyRecord, error) {
	if err := s.Check(ctx); err != nil {
		return nil, err
	}
	ctx = rest.WithOperationGuard(ctx, s.Check)
	options := []policies.RecordFindOption{}
	if strict {
		options = append(options, policies.WithRecordFindIgnoreMissing(false))
	}
	record, err := s.API.QoSPolicies.FindRecord(ctx, identity, options...)
	if err = errors.Join(err, s.Check(ctx)); err != nil {
		return record, err
	}
	if record == nil {
		return nil, &resource.NotFoundError{Resource: "QoS policy", Reference: identity}
	}
	return record, nil
}
