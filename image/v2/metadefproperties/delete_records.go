package metadefproperties

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// DeleteRecord deletes one fixed property identity and preserves the opaque
// actual response independently of handling errors. It accepts actual200..399
// and, by default, a physical404 without invoking the native retry callback.
// Resource input is snapshotted before options; only id/name select the route.
func (s *NamespaceScope) DeleteRecord(ctx context.Context, input RecordRequest, options ...DeleteOption) (*Acknowledgement, error) {
	fail := func(err error) (*Acknowledgement, error) { return nil, wrap(ctx, "DeleteRecord", err) }
	p, err := s.capture(ctx, nil)
	if err != nil {
		return fail(err)
	}
	namespace := s.namespace
	check := p.operationGuard(ctx)
	opctx := rest.WithOperationGuard(ctx, check)
	if err := check(opctx); err != nil {
		return fail(err)
	}
	identity, err := recordDeletionIdentity(input)
	if err = errors.Join(err, check(opctx)); err != nil {
		return fail(err)
	}
	endpoint := p.url(&identity)
	policy, err := prepareDelete(guardRecordDeleteOptions(opctx, check, slices.Clone(options)))
	if err = errors.Join(p.finish(opctx, policy.Headers, err), check(opctx)); err != nil {
		return fail(err)
	}
	codes := recordDeletionCodes()
	if policy.IgnoreMissing == nil || *policy.IgnoreMissing {
		codes = append(codes, http.StatusNotFound)
	}
	response, err := rest.DoJSONGuarded(opctx, p.client, check, http.MethodDelete, endpoint, nil, nil, codes...)
	if guardErr := check(opctx); guardErr != nil {
		if response != nil {
			guardErr = response.Fail(guardErr)
		}
		err = errors.Join(err, guardErr)
	}
	return recordDeletionAcknowledgement(namespace, &identity, response), wrap(ctx, "DeleteRecord", err)
}

// DeleteAllRecords deletes the fixed namespace's property collection once.
// A missing collection remains a native error. Actual200..399 has an opaque
// receipt without enumerating children or inferring a deleted-item count.
func (s *NamespaceScope) DeleteAllRecords(ctx context.Context, options ...DeleteAllOption) (*Acknowledgement, error) {
	fail := func(err error) (*Acknowledgement, error) { return nil, wrap(ctx, "DeleteAllRecords", err) }
	p, err := s.capture(ctx, nil)
	if err != nil {
		return fail(err)
	}
	namespace := s.namespace
	check := p.operationGuard(ctx)
	opctx := rest.WithOperationGuard(ctx, check)
	if err := check(opctx); err != nil {
		return fail(err)
	}
	endpoint := p.url(nil)
	policy, err := prepareDeleteAll(guardRecordDeleteAllOptions(opctx, check, slices.Clone(options)))
	if err = errors.Join(p.finish(opctx, policy.Headers, err), check(opctx)); err != nil {
		return fail(err)
	}
	response, err := rest.DoJSONGuarded(opctx, p.client, check, http.MethodDelete, endpoint, nil, nil, recordDeletionCodes()...)
	if guardErr := check(opctx); guardErr != nil {
		if response != nil {
			guardErr = response.Fail(guardErr)
		}
		err = errors.Join(err, guardErr)
	}
	return recordDeletionAcknowledgement(namespace, nil, response), wrap(ctx, "DeleteAllRecords", err)
}

// recordDeletionIdentity copies Resource before any caller option runs. An
// explicitly present id, including null, takes precedence over name. Other
// definition fields neither constrain deletion nor enter the request body.
func recordDeletionIdentity(input RecordRequest) (string, error) {
	if input.ID != "" && input.Resource != nil {
		return "", invalid("select ID or Resource, not both")
	}
	if input.ID == "" && input.Resource == nil {
		return "", invalid("property input is required")
	}
	identity := input.ID
	if input.Resource != nil {
		fields := input.Resource.Clone().Body
		raw, exists := fields["id"]
		if !exists {
			raw = fields["name"]
		}
		if !utf8.Valid(raw) {
			return "", invalid("property identity must be UTF-8")
		}
		if err := json.Unmarshal(raw, &identity); err != nil {
			return "", errors.Join(invalid("property identity must be a string"), err)
		}
	}
	if err := literal(identity); err != nil {
		return "", err
	}
	return identity, nil
}

// Existing preparation owns every config/header/pointer snapshot. These
// wrappers add sticky operation checks around every individual callback so a
// later callback cannot undo an observed source or namespace change.
func guardRecordDeleteOptions(ctx context.Context, check func(context.Context) error, options []DeleteOption) []DeleteOption {
	guarded := make([]DeleteOption, 0, len(options))
	for _, apply := range options {
		guarded = append(guarded, func(config *DeleteOpts) error {
			if err := check(ctx); err != nil {
				return err
			}
			if apply == nil {
				return invalid("nil property option")
			}
			return errors.Join(apply(config), check(ctx))
		})
	}
	return guarded
}
func guardRecordDeleteAllOptions(ctx context.Context, check func(context.Context) error, options []DeleteAllOption) []DeleteAllOption {
	guarded := make([]DeleteAllOption, 0, len(options))
	for _, apply := range options {
		guarded = append(guarded, func(config *DeleteAllOpts) error {
			if err := check(ctx); err != nil {
				return err
			}
			if apply == nil {
				return invalid("nil property option")
			}
			return errors.Join(apply(config), check(ctx))
		})
	}
	return guarded
}
func recordDeletionCodes() []int {
	codes := make([]int, 200)
	for index := range codes {
		codes[index] = 200 + index
	}
	return codes
}
func recordDeletionAcknowledgement(namespace string, name *string, response *rest.Response) *Acknowledgement {
	if response == nil {
		return nil
	}
	return &Acknowledgement{Namespace: namespace, Name: copyPointer(name), Body: bytes.Clone(response.Body),
		Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
