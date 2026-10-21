package network

import (
	"context"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/ruletypes"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// QoSRuleTypeDetails is the result of GetQoSRuleTypeDetails. Extensions holds
// every extension row observed by the two ordered gates (the "qos" gate rows
// first). RuleType separates the declared Resource view from the actual Wire
// row and keeps the response envelope, headers and status.
type QoSRuleTypeDetails struct {
	RuleType   *ruletypes.QoSRuleTypeRecord
	Extensions []*extensions.ExtensionRecord
}

// GetQoSRuleTypeDetails reads one QoS rule type by name the way the pinned
// cloud get_qos_rule_type_details helper does: a complete fresh extension
// listing that must contain "qos", a second complete fresh listing that must
// contain "qos-rule-type-details", then GET qos/rule-types/{ruleType}. The
// Python filters argument is ignored by the pinned source and has no Go
// counterpart. A missing alias returns *UnavailableExtensionError (matching
// ErrUnavailableExtension) before any rule type request. A 404 on the final
// GET is an error, not a nil result.
//
// Stock Neutron additionally restricts the details read to administrators;
// a denial is reported as the server's HTTP error. No request is retried by
// this helper and ctx cancellation never implies remote rollback (the helper
// is read-only).
func (s *Service) GetQoSRuleTypeDetails(ctx context.Context, ruleType string, options ...ruletypes.RecordGetOption) (*QoSRuleTypeDetails, error) {
	wrap := func(err error) error {
		return request.Wrap("GetQoSRuleTypeDetails", "qos rule types", cloudread.ContextError(ctx, err))
	}
	if err := resource.ID(ruleType).Validate(); err != nil {
		return nil, wrap(err)
	}
	state, err := qosReadStart(ctx, s, "qos", "qos-rule-type-details")
	result := &QoSRuleTypeDetails{}
	if state != nil {
		result.Extensions = slices.Clone(state.Extensions)
	}
	if err != nil {
		return result, wrap(err)
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	record, err := state.API.QoSRuleTypes.GetRecord(ctx, ruleType, options...)
	result.RuleType = record
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return result, wrap(err)
	}
	return result, nil
}
