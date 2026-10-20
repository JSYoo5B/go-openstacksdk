// Package cinderaction owns direct-ID Cinder volume actions.
package cinderaction

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/JSYoo5B/go-openstacksdk/internal/cinderrequest"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"unicode"
	"unicode/utf8"
)

type State uint8

const (
	Reserve State = iota
	Unreserve
	BeginDetaching
	AbortDetaching
)

func (s State) Operation() string {
	switch s {
	case Reserve:
		return "ReserveVolume"
	case Unreserve:
		return "UnreserveVolume"
	case BeginDetaching:
		return "BeginVolumeDetaching"
	case AbortDetaching:
		return "AbortVolumeDetaching"
	}
	return "VolumeStateAction"
}
func (s State) action() string {
	switch s {
	case Reserve:
		return "os-reserve"
	case Unreserve:
		return "os-unreserve"
	case BeginDetaching:
		return "os-begin_detaching"
	case AbortDetaching:
		return "os-roll_detaching"
	}
	return ""
}

// Result reports helper acknowledgement; Completed never proves server state.
type Result struct {
	VolumeID     string
	Microversion string
	Discovery    []*rest.Response
	Applied      *rest.Response
	Completed    bool
}

func ValidateID(id string) error {
	if err := resource.ID(id).Validate(); err != nil {
		return err
	}
	if !utf8.ValidString(id) {
		return fmt.Errorf("%w: volume ID must be UTF-8", resource.ErrInvalidOption)
	}
	for _, c := range id {
		if unicode.IsControl(c) || unicode.IsSpace(c) {
			return fmt.Errorf("%w: volume ID must not contain whitespace or controls", resource.ErrInvalidOption)
		}
	}
	return nil
}
func Wrap(ctx context.Context, state State, err error) error {
	return WrapOperation(ctx, state.Operation(), err)
}

func WrapOperation(ctx context.Context, operation string, err error) error {
	return request.Wrap(operation, "volumes", cloudread.ContextError(ctx, err))
}

// Apply captures the selected service, negotiates only when necessary, and
// posts the exact source null action. No lookup, model conversion or wait.
func Apply(ctx context.Context, client *gophercloud.ServiceClient, id string, state State) (*Result, error) {
	source, err := cloudread.Capture(ctx, client, "volume")
	if err == nil {
		err = ValidateID(id)
	}
	action := state.action()
	if err == nil && action == "" {
		err = fmt.Errorf("%w: unknown volume state action", resource.ErrInvalidOption)
	}
	if err != nil {
		return nil, Wrap(ctx, state, err)
	}
	body, _ := json.Marshal(map[string]any{action: nil})
	return applyPrepared(ctx, source, id, state.Operation(), body)
}

// applyPrepared keeps the fixed route and Cinder policy shared by direct actions.
func applyPrepared(ctx context.Context, source *cloudread.Source, id, operation string, body json.RawMessage) (*Result, error) {
	return applyRequired(ctx, source, id, operation, body, "")
}

func applyRequired(ctx context.Context, source *cloudread.Source, id, operation string, body json.RawMessage, required string) (*Result, error) {
	target := source.Client.ServiceURL("volumes", id, "action")
	if err := rest.ValidateTarget(&source.Client, target); err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	result := &Result{VolumeID: id}
	var err error
	if required == "" {
		result.Microversion, result.Discovery, err = cinderrequest.Negotiate(ctx, source, "3.71")
	} else {
		result.Microversion, result.Discovery, err = cinderrequest.NegotiateRequired(ctx, source, "3.71", required)
	}
	if err == nil {
		err = source.WithPolicy(ctx, &result.Microversion, nil)
	}
	if err != nil {
		return result, WrapOperation(ctx, operation, err)
	}
	codes := make([]int, 300)
	for i := range codes {
		codes[i] = 100 + i
	}
	result.Applied, err = cinderrequest.Post(ctx, source, target, body, &result.Microversion, codes...)
	result.Completed = err == nil
	return result, WrapOperation(ctx, operation, err)
}
