package cinderaction

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2"
	"gophercloudsdk/internal/cinderrequest"
	"gophercloudsdk/internal/rest"
)

// ImageMetadataDeletion records one key and its actual accepted response.
// Failed entries can lack Response when transmission or native admission fails.
type ImageMetadataDeletion struct {
	Key      string
	Response *rest.Response
}

// ImageMetadataDeleteResult separates member observation, discovery and action
// proofs. Completed means this helper finished, never remote metadata state.
type ImageMetadataDeleteResult struct {
	VolumeID     string
	Microversion string
	Discovery    []*rest.Response
	Observed     *rest.Response
	Deleted      []ImageMetadataDeletion
	Failed       *ImageMetadataDeletion
	Completed    bool
}

func SetImageMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string, options ...ImageMetadataOption) (*Result, error) {
	const operation = "SetVolumeImageMetadata"
	source, err := captureAction(ctx, client, id)
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	policy, err := prepareImageMetadata(options, func() error { return source.Guard(ctx) })
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	body, err := json.Marshal(map[string]any{"os-set_image_metadata": map[string]any{"metadata": policy.Metadata}})
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	return applyPrepared(ctx, source, id, operation, body)
}

func imageMetadataCodes() []int {
	codes := make([]int, 300)
	for i := range codes {
		codes[i] = 100 + i
	}
	return codes
}

func DeleteImageMetadata(ctx context.Context, client *gophercloud.ServiceClient, id string, options ...ImageMetadataDeleteOption) (*ImageMetadataDeleteResult, error) {
	const operation = "DeleteVolumeImageMetadata"
	source, err := captureAction(ctx, client, id)
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	policy, err := prepareImageMetadataDelete(options, func() error { return source.Guard(ctx) })
	if err != nil {
		return nil, WrapOperation(ctx, operation, err)
	}
	result := &ImageMetadataDeleteResult{VolumeID: id, Microversion: source.Client.Microversion}
	fail := func(err error) (*ImageMetadataDeleteResult, error) { return result, WrapOperation(ctx, operation, err) }
	if policy.Keys != nil && len(*policy.Keys) == 0 {
		err := source.Guard(ctx)
		result.Completed = err == nil
		return fail(err)
	}
	target := source.Client.ServiceURL("volumes", id, "action")
	if err := rest.ValidateTarget(&source.Client, target); err != nil {
		return fail(err)
	}
	result.Microversion, result.Discovery, err = cinderrequest.Negotiate(ctx, source, "3.71")
	if err == nil {
		err = source.WithPolicy(ctx, &result.Microversion, nil)
	}
	if err != nil {
		return fail(err)
	}
	var keys []string
	if policy.Keys == nil {
		member := source.Client.ServiceURL("volumes", id)
		result.Observed, err = cinderrequest.MemberGet(ctx, source, member, result.Microversion, imageMetadataCodes()...)
		if err == nil {
			keys, err = observedImageMetadataKeys(result.Observed)
		}
		if err != nil {
			return fail(err)
		}
	} else {
		keys = *policy.Keys
	}
	for _, key := range keys {
		if err := source.Guard(ctx); err != nil {
			return fail(err)
		}
		body, _ := json.Marshal(map[string]any{"os-unset_image_metadata": map[string]string{"key": key}})
		response, err := cinderrequest.Post(ctx, source, target, body, &result.Microversion, imageMetadataCodes()...)
		entry := ImageMetadataDeletion{Key: key, Response: response}
		if err != nil {
			result.Failed = &entry
			return fail(err)
		}
		result.Deleted = append(result.Deleted, entry)
	}
	err = source.Guard(ctx)
	result.Completed = err == nil
	return fail(err)
}

func observedImageMetadataKeys(response *rest.Response) ([]string, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(fmt.Errorf("volume member response must be UTF-8 JSON"))
	}
	body, err := response.Object("volume")
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return nil, response.Fail(err)
	}
	metadata, exists := fields["volume_image_metadata"]
	if !exists {
		return nil, nil
	}
	var values map[string]json.RawMessage
	if err := json.Unmarshal(metadata, &values); err != nil {
		return nil, response.Fail(fmt.Errorf("volume_image_metadata must be an object or null: %w", err))
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}
