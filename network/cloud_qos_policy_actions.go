package network

import (
	"context"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/policies"
)

// QoSPolicyLookup retains completed extension discovery and an optional policy.
// Policy nil with a nil error is the cloud get_qos_policy missing result.
type QoSPolicyLookup struct {
	Policy     *policies.PolicyRecord
	Extensions []*extensions.ExtensionRecord
}

// QoSPolicyDeletion distinguishes missing lookup (Deleted=false, Delete=nil)
// from a found policy followed by successful/ignored404 DELETE (Deleted=true).
// Lookup and DELETE evidence remain independent snapshots.
type QoSPolicyDeletion struct {
	Deleted    bool
	Policy     *policies.PolicyRecord
	Delete     *policies.PolicyDeleteRecord
	Extensions []*extensions.ExtensionRecord
}

// GetQoSPolicyRecord performs a complete fresh qos gate, then the fixed Source
// cloud get lookup with default ignore-missing. Filters are kwargs, not local
// _filter_list/JMES expressions. Remaining fetch-control gaps are documented.
func (s *Service) GetQoSPolicyRecord(ctx context.Context, identity string, options ...QoSPolicyReadOption) (*QoSPolicyLookup, error) {
	const op = "GetQoSPolicyRecord"
	state, err := qosReadStart(ctx, s, "qos")
	var result *QoSPolicyLookup
	if state != nil {
		result = &QoSPolicyLookup{Extensions: state.Extensions}
	}
	if err != nil {
		return result, qosPolicyCloudWrap(ctx, op, err)
	}
	fail := func(err error) (*QoSPolicyLookup, error) { return result, qosPolicyCloudWrap(ctx, op, err) }
	policy := copyQoSPolicyReadOpts(QoSPolicyReadOpts{})
	for _, apply := range slices.Clone(options) {
		if err := state.Check(ctx); err != nil {
			return fail(err)
		}
		if apply == nil {
			return fail(qosPolicyCloudInvalid("nil QoS policy lookup option"))
		}
		next := copyQoSPolicyReadOpts(policy)
		err := apply(&next)
		if err = errors.Join(err, state.Check(ctx)); err != nil {
			return fail(err)
		}
		policy = copyQoSPolicyReadOpts(next)
	}
	members, err := qosPolicyCloudPairs(policy.Filters, false)
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return fail(err)
	}
	leaf := []policies.RecordFindOption{policies.WithRecordFindIgnoreMissing(true)}
	for _, member := range members {
		// Cloud already binds these arguments. Reject by presence, including null.
		if member.Key == "name_or_id" || member.Key == "ignore_missing" {
			return fail(qosPolicyCloudInvalid("%q duplicates the cloud Find argument", member.Key))
		}
		leaf = append(leaf, policies.WithRecordFindQuery(member.Key, member.Value))
	}
	if len(policy.Headers) > 0 {
		leaf = append(leaf, policies.WithRecordFindHeaders(policy.Headers))
	}
	if policy.Microversion != nil {
		leaf = append(leaf, policies.WithRecordFindMicroversion(*policy.Microversion))
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	record, err := state.API.QoSPolicies.FindSourceRecord(ctx, identity, leaf...)
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return fail(err)
	}
	result.Policy = record
	return result, nil
}

// DeleteQoSPolicyRecord implements Source cloud Delete's fixed missing policy.
// It looks up once after fresh full qos discovery. Missing lookup returns false;
// found lookup followed by cleanDELETE404 still returns true (Proxy default).
func (s *Service) DeleteQoSPolicyRecord(ctx context.Context, identity string) (*QoSPolicyDeletion, error) {
	const op = "DeleteQoSPolicyRecord"
	state, err := qosReadStart(ctx, s, "qos")
	var result *QoSPolicyDeletion
	if state != nil {
		result = &QoSPolicyDeletion{Extensions: state.Extensions}
	}
	if err != nil {
		return result, qosPolicyCloudWrap(ctx, op, err)
	}
	ctx = rest.WithOperationGuard(ctx, state.Check)
	record, err := state.API.QoSPolicies.FindSourceRecord(ctx, identity, policies.WithRecordFindIgnoreMissing(true))
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return result, qosPolicyCloudWrap(ctx, op, err)
	}
	if record == nil {
		return result, nil
	}
	result.Policy = record
	raw := record.Resource.Body["id"]
	truthy, cause := cloudfilter.PythonJSONTruthy(raw)
	if cause != nil || !truthy {
		// The SDK keeps the deletion a literal member operation. Source urljoin
		// can turn false/0/empty IDs into a collection route; this Go boundary is
		// explicit and is never reported as missing or successful deletion.
		return result, qosPolicyCloudWrap(ctx, op, errors.Join(cause, qosPolicyCloudInvalid("found policy requires a nonempty member ID")))
	}
	member, cause := cloudfilter.PythonString(raw)
	if cause != nil {
		return result, qosPolicyCloudWrap(ctx, op, cause)
	}
	result.Delete, err = state.API.QoSPolicies.DeleteSourceRecord(ctx, member)
	if err = errors.Join(err, state.Check(ctx)); err != nil {
		return result, qosPolicyCloudWrap(ctx, op, err)
	}
	result.Deleted = true
	return result, nil
}
