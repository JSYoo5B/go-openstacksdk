package waits

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type pollValue struct {
	Record   *Record
	State    string
	Progress int
}

func stateToken(state *string) string {
	if state == nil {
		return "null"
	}
	return "value:" + hex.EncodeToString([]byte(cloudfilter.PythonLower(*state)))
}
func validID(id string) error {
	if id == "" || !utf8.ValidString(id) {
		return invalid("a fixed nonempty request ID is required")
	}
	for _, v := range id {
		if unicode.IsControl(v) {
			return invalid("request ID contains controls")
		}
	}
	return nil
}

// WaitForStatus defaults to status, ERROR failure detection, two-second polling
// and no SDK timeout. A nil target status means Source's actual None, not the
// string "None". Matching cached attributes return without HTTP.
func (a *API) WaitForStatus(ctx context.Context, target Target, status *string, options ...Option) (*Record, error) {
	if status != nil {
		v := *status
		status = &v
	}
	return a.wait(ctx, target, status, false, options...)
}

// WaitForDelete polls GET, never DELETE. Only a clean metadata 404 or a fetched
// string status whose Python lower is deleted establishes completion. A payload
// 404, callback, read/Close, source or cancellation failure stays an error.
func (a *API) WaitForDelete(ctx context.Context, target Target, options ...Option) (*Record, error) {
	return a.wait(ctx, target, nil, true, options...)
}
func (a *API) wait(ctx context.Context, target Target, status *string, deleting bool, options ...Option) (*Record, error) {
	op := "WaitForStatus"
	if deleting {
		op = "WaitForDelete"
	}
	wrap := func(err error) error {
		if err == nil {
			return nil
		}
		return request.Wrap(op, target.kind, cloudread.ContextError(ctx, err))
	}
	var client *gophercloud.ServiceClient
	if a != nil {
		client = a.client
	}
	source, err := cloudread.Capture(ctx, client, "key-manager")
	if err != nil {
		return nil, wrap(err)
	}
	var observed error
	guard := func(ctx context.Context) error {
		return errors.Join(source.Guard(ctx), rest.CheckOperationGuard(ctx), observed)
	}
	target.seed = target.seed.Clone()
	if target.err != nil {
		return nil, wrap(target.err)
	}
	if target.seed == nil || target.seed.Resource == nil {
		return nil, wrap(invalid("a concrete owned target is required"))
	}
	if err := validID(target.id); err != nil {
		return nil, wrap(err)
	}
	if target.kind != "containers" && target.kind != "orders" && target.kind != "secrets" {
		return nil, wrap(resource.ErrUnsupported)
	}
	config := Options{}
	for _, apply := range slices.Clone(options) {
		if err := guard(ctx); err != nil {
			return nil, wrap(err)
		}
		if apply == nil {
			return nil, wrap(invalid("nil wait option"))
		}
		config = ownOptions(config)
		applied := apply(&config)
		config = ownOptions(config)
		if err := errors.Join(applied, guard(ctx)); err != nil {
			return nil, wrap(err)
		}
	}
	if err := source.WithPolicy(ctx, config.Microversion, config.Headers); err != nil {
		return nil, wrap(err)
	}
	attribute := "status"
	if config.Attribute != nil {
		attribute = *config.Attribute
	}
	if deleting && (config.Attribute != nil || config.Failures != nil) {
		return nil, wrap(fmt.Errorf("%w: deletion uses only the status descriptor", resource.ErrUnsupported))
	}
	if !validAttribute(target.kind, attribute) {
		return nil, wrap(fmt.Errorf("%w: unknown declared wait attribute %q", resource.ErrUnsupported, attribute))
	}
	seedState, err := text(target.seed.Resource.Body[attribute])
	// Deletion ignores cached status and always performs the first fetch.
	if !deleting {
		if err != nil {
			return nil, wrap(err)
		}
		if stateToken(seedState) == stateToken(status) {
			if err := guard(ctx); err != nil {
				return nil, wrap(err)
			}
			return target.seed.Clone(), nil
		}
	}
	interval := 2 * time.Second
	if config.Interval != nil {
		interval = *config.Interval
	}
	timeout := time.Duration(0)
	if deleting {
		timeout = 120 * time.Second
	}
	if config.Timeout != nil {
		timeout = *config.Timeout
	}
	if config.Unlimited {
		timeout = 0
	}
	// Source timeout zero/negative exhausts before the first observation. A nil
	// status timeout denotes infinity; Unlimited explicitly chooses it for delete.
	if config.Timeout != nil && !config.Unlimited && timeout <= 0 {
		return nil, wrap(context.DeadlineExceeded)
	}
	if interval < 0 {
		return nil, wrap(invalid("negative polling interval"))
	}
	if interval == 0 {
		interval = 100 * time.Millisecond
		if timeout > 0 && timeout < interval {
			interval = timeout
		}
	}
	waitOptions := []resource.WaitOption{resource.WithPollInterval(interval)}
	if timeout == 0 {
		waitOptions = append(waitOptions, resource.WithUnlimitedWait())
	} else {
		waitOptions = append(waitOptions, resource.WithTimeout(timeout))
	}
	failures := config.Failures
	if failures == nil {
		failures = []string{"ERROR"}
	}
	tokens := make([]string, len(failures))
	for i, v := range failures {
		tokens[i] = stateToken(&v)
	}
	waitOptions = append(waitOptions, resource.WithFailureStates(tokens...))
	if config.Callback != nil {
		waitOptions = append(waitOptions, resource.WithProgressCallback(func(progress int) { observed = errors.Join(observed, config.Callback(progress)) }))
	}
	policy, err := resource.PrepareWaitOptionsFor[pollValue](waitOptions...)
	if err != nil {
		return nil, wrap(err)
	}
	var receipt *Record
	current := target.seed.Clone()
	collection := resource.NewCollection(resource.Adapter[pollValue]{
		Kind: target.kind, ValidateID: validID, ID: func(v *pollValue) string { return target.id },
		WaitGuard: guard, Status: func(v *pollValue) string { return v.State },
		Get: func(ctx context.Context, id string) (*pollValue, error) {
			value, err := fetch(ctx, source, guard, target.kind, id, current)
			receipt = value
			if err != nil {
				return nil, err
			}
			current = value.Clone()
			state, stateErr := text(value.Resource.Body[attribute])
			if deleting && state == nil && stateErr == nil {
				stateErr = invalid("delete status cannot be null")
			}
			if stateErr != nil {
				return nil, proofError(value, stateErr)
			}
			token := stateToken(state)
			if deleting && state != nil && cloudfilter.PythonLower(*state) == "deleted" {
				token = "deleted"
			}
			return &pollValue{Record: value, State: token}, nil
		},
	})
	if deleting {
		err = collection.WaitDeleted(ctx, resource.ID(target.id), resource.WithWaitPolicy(policy))
		if err == nil {
			if receipt == nil {
				return target.seed.Clone(), nil
			}
			if receipt.Resource == nil {
				receipt.Resource = target.seed.Resource.Clone()
				receipt.Payload = target.seed.Clone().Payload
			}
			return receipt.Clone(), nil
		}
	} else {
		var value *pollValue
		value, err = collection.Wait(ctx, resource.ID(target.id), stateToken(status), resource.WithWaitPolicy(policy))
		if err == nil && value != nil {
			return value.Record.Clone(), nil
		}
	}
	var failed *resource.FailedStateError
	if errors.As(err, &failed) && current != nil && current.Resource != nil {
		state, _ := text(current.Resource.Body[attribute])
		failed.Status = "None"
		if state != nil {
			failed.Status = *state
		}
	}
	err = errors.Join(err, guard(ctx))
	if receipt != nil {
		receipt = receipt.Clone()
		receipt.Resource = nil
		err = proofError(receipt, err)
	}
	return receipt, wrap(err)
}

// Attach the last accepted observation only when the error has no actual
// response proof already. A payload rejection retains its own native receipt.
func proofError(record *Record, err error) error {
	if err == nil || record == nil || record.StatusCode == 0 {
		return err
	}
	var accepted *resource.ResponseError
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &accepted) || errors.As(err, &native) {
		return err
	}
	return &resource.ResponseError{Body: append([]byte(nil), record.Envelope...), Header: record.Header.Clone(), StatusCode: record.StatusCode, Cause: err}
}
func recordFromError(kind, id string, err error) *Record {
	var native gophercloud.ErrUnexpectedResponseCode
	if errors.As(err, &native) {
		r := &Record{Kind: kind, RequestID: id, Envelope: append([]byte(nil), native.Body...), Header: native.ResponseHeader.Clone(), StatusCode: native.Actual}
		var wire resource.RawResource
		if json.Unmarshal(native.Body, &wire) == nil {
			wire.Header = r.Header.Clone()
			wire.StatusCode = r.StatusCode
			r.Wire = wire.Clone()
		}
		return r
	}
	return nil
}
