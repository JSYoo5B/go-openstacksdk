package cloudsnapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// resolveMutationWait resolves the shared wait options of a snapshot or backup
// mutation: unlimited by default with two-second polling. Only the timeout,
// unlimited, poll interval and wait strategy controls are supported; a zero or
// negative timeout or interval and any other control fail with
// resource.ErrInvalidOption before any request.
func resolveMutationWait(options []resource.WaitOption) (resource.WaitSettings, error) {
	return resource.ResolveWait(resource.WaitDefaults{Unlimited: true, Strategy: resource.FixedWait(2 * time.Second)}, options...)
}

// CreateAttributes supplies the four cloud-helper name/description aliases.
// Concrete string pointers override the corresponding raw Fields entry.
// Fields owns ordinary JSON values; it is not a native options builder.
type CreateAttributes struct {
	Name, DisplayName, Description, DisplayDescription *string
	Fields                                             map[string]json.RawMessage
}

type CreateOptions struct {
	Force, Wait *bool
	WaitOptions []resource.WaitOption
	Attributes  CreateAttributes
	Location    *resource.CloudLocation
}

type DeleteOptions struct {
	Wait        *bool
	WaitOptions []resource.WaitOption
	Location    *resource.CloudLocation
}

type CreateOption = Option[CreateOptions]
type DeleteOption = Option[DeleteOptions]

func WithCreateOptions(value CreateOptions) CreateOption {
	return withValue(value, cloneCreateOptions)
}
func WithDeleteOptions(value DeleteOptions) DeleteOption {
	return withValue(value, cloneDeleteOptions)
}
func WithCreateForce(value bool) CreateOption {
	return func(target *CreateOptions) error { target.Force = clonePointer(&value); return nil }
}
func WithCreateWait(value bool) CreateOption {
	return func(target *CreateOptions) error { target.Wait = clonePointer(&value); return nil }
}
func WithDeleteWait(value bool) DeleteOption {
	return func(target *DeleteOptions) error { target.Wait = clonePointer(&value); return nil }
}
func WithCreateWaitOptions(options ...resource.WaitOption) CreateOption {
	owned := slices.Clone(options)
	return func(target *CreateOptions) error {
		target.WaitOptions = append(slices.Clone(target.WaitOptions), owned...)
		return nil
	}
}
func WithDeleteWaitOptions(options ...resource.WaitOption) DeleteOption {
	owned := slices.Clone(options)
	return func(target *DeleteOptions) error {
		target.WaitOptions = append(slices.Clone(target.WaitOptions), owned...)
		return nil
	}
}
func WithCreateAttributes(value CreateAttributes) CreateOption {
	owned := cloneCreateAttributes(value)
	return func(target *CreateOptions) error { target.Attributes = cloneCreateAttributes(owned); return nil }
}
func WithCreateFields(value map[string]json.RawMessage) CreateOption {
	owned := cloneMutationFields(value)
	return func(target *CreateOptions) error { target.Attributes.Fields = cloneMutationFields(owned); return nil }
}
func WithCreateName(value string) CreateOption {
	return func(target *CreateOptions) error { target.Attributes.Name = clonePointer(&value); return nil }
}
func WithCreateDisplayName(value string) CreateOption {
	return func(target *CreateOptions) error { target.Attributes.DisplayName = clonePointer(&value); return nil }
}
func WithCreateDescription(value string) CreateOption {
	return func(target *CreateOptions) error { target.Attributes.Description = clonePointer(&value); return nil }
}
func WithCreateDisplayDescription(value string) CreateOption {
	return func(target *CreateOptions) error {
		target.Attributes.DisplayDescription = clonePointer(&value)
		return nil
	}
}

// Preparation owns policy and executes originals once without service I/O.
// Runtime defaults and validation of reached fields belong to the workflow.
func PrepareCreate(ctx context.Context, options ...CreateOption) (CreateOptions, error) {
	return prepare(ctx, options, cloneCreateOptions, nil)
}
func PrepareDelete(ctx context.Context, options ...DeleteOption) (DeleteOptions, error) {
	return prepare(ctx, options, cloneDeleteOptions, nil)
}

func cloneMutationFields(value map[string]json.RawMessage) map[string]json.RawMessage {
	if value == nil {
		return nil
	}
	owned := make(map[string]json.RawMessage, len(value))
	for key, raw := range value {
		owned[key] = bytes.Clone(raw)
	}
	return owned
}
func cloneCreateAttributes(value CreateAttributes) CreateAttributes {
	value.Name = clonePointer(value.Name)
	value.DisplayName = clonePointer(value.DisplayName)
	value.Description = clonePointer(value.Description)
	value.DisplayDescription = clonePointer(value.DisplayDescription)
	value.Fields = cloneMutationFields(value.Fields)
	return value
}
func cloneCreateOptions(value CreateOptions) CreateOptions {
	value.Force = clonePointer(value.Force)
	value.Wait = clonePointer(value.Wait)
	value.WaitOptions = slices.Clone(value.WaitOptions)
	value.Attributes = cloneCreateAttributes(value.Attributes)
	value.Location = cloneLocation(value.Location)
	return value
}
func cloneDeleteOptions(value DeleteOptions) DeleteOptions {
	value.Wait = clonePointer(value.Wait)
	value.WaitOptions = slices.Clone(value.WaitOptions)
	value.Location = cloneLocation(value.Location)
	return value
}
