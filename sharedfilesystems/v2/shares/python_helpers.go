package shares

import (
	"context"
	"fmt"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

// ForceRemove sends the force_delete share action for one share ID. A 404 is
// ignored by default, like Remove; pass resource.WithMissingError to report
// resource.ErrNotFound. The ID is never resolved by name, nothing is retried and
// acknowledgement does not imply the share is gone: use WaitForDeletion.
func (a *API) ForceRemove(ctx context.Context, id string, options ...resource.LookupOption) error {
	ignore, err := forceRemovePolicy(id, options)
	if err != nil {
		return request.Wrap("ForceRemove", "shares", err)
	}
	err = a.ForceDelete(ctx, id)
	if err != nil && ctx.Err() == nil && gophercloud.ResponseCodeIs(err, 404) {
		if ignore {
			return nil
		}
		return request.Wrap("ForceRemove", "shares", &resource.NotFoundError{Resource: "shares", Reference: id, Cause: err})
	}
	return err
}

func forceRemovePolicy(id string, options []resource.LookupOption) (bool, error) {
	if err := resource.ID(id).Validate(); err != nil {
		return false, err
	}
	return resource.IgnoresMissing(true, options...)
}

// ReplaceMetadata replaces all metadata of a share with PUT
// /shares/{id}/metadata and returns the metadata echoed by the service. A nil
// or empty map sends {"metadata":{}} and therefore clears the metadata. Only
// HTTP 200 succeeds. Use SetMetadata to merge keys instead.
func (a *API) ReplaceMetadata(ctx context.Context, id string, metadata map[string]string) (map[string]string, error) {
	if err := resource.ID(id).Validate(); err != nil {
		return nil, request.Wrap("ReplaceMetadata", "shares", err)
	}
	if metadata == nil {
		metadata = map[string]string{}
	}
	var out struct {
		Metadata map[string]string `json:"metadata"`
	}
	_, err := a.client.Put(ctx, a.client.ServiceURL("shares", id, "metadata"), map[string]any{"metadata": metadata}, &out,
		&gophercloud.RequestOpts{OkCodes: []int{200}})
	if err != nil {
		return nil, request.Wrap("ReplaceMetadata", "shares", err)
	}
	return out.Metadata, nil
}

// MetadataDeleteError reports keys whose individual DELETE did not succeed.
// Attempted keys not listed in Failed succeeded or were ignored as missing.
// Unattempted keys exist only when the context ended the loop.
type MetadataDeleteError struct {
	Failed      []string
	Unattempted []string
	Causes      []error
}

func (e *MetadataDeleteError) Error() string {
	return fmt.Sprintf("share metadata keys failed to delete: %s (unattempted: %s)", strings.Join(e.Failed, ","), strings.Join(e.Unattempted, ","))
}

func (e *MetadataDeleteError) Unwrap() []error { return e.Causes }

// DeleteMetadata deletes each key with its own DELETE /shares/{id}/metadata/{key},
// in order and without replay. Missing keys (404) are ignored by default; pass
// resource.WithMissingError to count them as failures. Every other failure is
// recorded and the loop continues, and the collected failures are returned as a
// *MetadataDeleteError. Context cancellation stops the loop at once and the
// remaining keys are reported as Unattempted. Empty keys fail before any request.
func (a *API) DeleteMetadata(ctx context.Context, id string, keys []string, options ...resource.LookupOption) error {
	ignore, err := forceRemovePolicy(id, options)
	if err != nil {
		return request.Wrap("DeleteMetadata", "shares", err)
	}
	for _, key := range keys {
		if key == "" {
			return request.Wrap("DeleteMetadata", "shares", fmt.Errorf("%w: metadata key must not be empty", resource.ErrInvalidOption))
		}
	}
	var failure *MetadataDeleteError
	for index, key := range keys {
		err := a.DeleteMetadatum(ctx, id, key)
		if err == nil {
			continue
		}
		if ctx.Err() != nil {
			if failure == nil {
				failure = &MetadataDeleteError{}
			}
			failure.Failed = append(failure.Failed, key)
			failure.Causes = append(failure.Causes, err)
			failure.Unattempted = append(failure.Unattempted, keys[index+1:]...)
			break
		}
		if ignore && gophercloud.ResponseCodeIs(err, 404) {
			continue
		}
		if failure == nil {
			failure = &MetadataDeleteError{}
		}
		failure.Failed = append(failure.Failed, key)
		failure.Causes = append(failure.Causes, err)
	}
	if failure == nil {
		return nil
	}
	return request.Wrap("DeleteMetadata", "shares", failure)
}
