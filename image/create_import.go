package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
	"gophercloudsdk/image/v2/imagedata"
	"gophercloudsdk/image/v2/imageimport"
	"gophercloudsdk/internal/rest"
	"gophercloudsdk/request"
	"gophercloudsdk/resource"
)

// CreateAndImportRequest names a new image and, for glance-direct, supplies
// borrowed data from its current cursor. Remote imports require Data nil.
type CreateAndImportRequest struct {
	Name string
	Data io.Reader
}

// CreateAndImport creates metadata, stages direct data, submits import and
// optionally waits for active. All caller options are frozen before POST or
// reader use. It retains completed phases on failure and never deletes images.
func (s *Service) CreateAndImport(ctx context.Context, input CreateAndImportRequest, options ...CreateImportOption) (*CreateImportResult, error) {
	wrap := func(err error) error { return wrapCreateImportError(ctx, err) }
	if s == nil || s.client == nil {
		return nil, wrap(uploadInvalid("image service is required"))
	}
	source := s.client
	// Validate the original source before any caller callback. Both phases use
	// the same selected collection, while authentication stays on its provider.
	if _, err := imagedata.New(source).PrepareStageOptions(ctx); err != nil {
		return nil, wrap(err)
	}
	if _, err := imageimport.New(source).PrepareImportOptions(ctx); err != nil {
		return nil, wrap(err)
	}
	if strings.TrimSpace(input.Name) == "" || !utf8.ValidString(input.Name) {
		return nil, wrap(uploadInvalid("image name must be nonempty valid UTF-8"))
	}
	if input.Data != nil && isNilReader(input.Data) {
		return nil, wrap(uploadInvalid("image data reader must not be typed nil"))
	}
	client := *source
	client.MoreHeaders = maps.Clone(source.MoreHeaders)
	collection := source.ServiceURL("images")
	policy, err := parseCreateImportOptions(append([]CreateImportOption(nil), options...))
	if err != nil {
		return nil, wrap(err)
	}
	stageAPI, importAPI := imagedata.New(&client), imageimport.New(&client)
	stageOptions := append([]imagedata.StageOption{imagedata.WithStageOpts(policy.Stage)}, policy.stageOptions...)
	policy.Stage, err = stageAPI.PrepareStageOptions(ctx, stageOptions...)
	if err != nil {
		return nil, wrap(err)
	}
	importOptions := append([]imageimport.ImportOption{imageimport.WithImportOpts(policy.Import)}, policy.importOptions...)
	policy.Import, err = importAPI.PrepareImportOptions(ctx, importOptions...)
	if err != nil {
		return nil, wrap(err)
	}
	switch policy.Import.Method {
	case imageimport.GlanceDirectMethod:
		if input.Data == nil {
			return nil, wrap(uploadInvalid("glance-direct requires data"))
		}
	case imageimport.WebDownloadMethod, imageimport.GlanceDownloadMethod:
		if input.Data != nil {
			return nil, wrap(uploadInvalid("remote import requires data to be nil"))
		}
		if policy.Stage.Size != nil || len(policy.Stage.Headers) != 0 {
			return nil, wrap(uploadInvalid("stage settings require glance-direct"))
		}
	default:
		return nil, wrap(fmt.Errorf("%w: fresh-image import method %q is unsupported", resource.ErrUnsupported, policy.Import.Method))
	}
	metadata, err := prepareCreateImportMetadata(input.Name, policy.Metadata)
	if err != nil {
		return nil, wrap(err)
	}
	metadataPolicy, err := importAPI.PrepareImportOptions(ctx, imageimport.WithImportOpts(imageimport.ImportOpts{Headers: policy.Metadata.Headers}))
	if err != nil {
		return nil, wrap(err)
	}
	waitOptions, err := prepareCreateImportWait(policy.Wait)
	if err != nil {
		return nil, wrap(err)
	}
	check := func() error {
		if source.ProviderClient != client.ProviderClient {
			return uploadInvalid("image workflow provider changed")
		}
		if _, err := imagedata.New(source).PrepareStageOptions(ctx); err != nil {
			return err
		}
		if _, err := imageimport.New(source).PrepareImportOptions(ctx, imageimport.WithImportOpts(policy.Import)); err != nil {
			return err
		}
		return rest.ValidateTarget(source, collection)
	}
	if err := check(); err != nil {
		return nil, wrap(err)
	}
	response, err := rest.DoJSON(ctx, &client, http.MethodPost, collection, metadata, metadataPolicy.Headers, http.StatusCreated)
	if response == nil {
		return nil, wrap(err)
	}
	result := &CreateImportResult{Created: &CreatedImageResponse{Body: append(json.RawMessage(nil), response.Body...), Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	if err != nil {
		return result, wrap(err)
	}
	fields, err := createImportObject(response)
	if err != nil {
		return result, wrap(err)
	}
	result.ImageID, err = createImportString(response, fields, "id")
	if err != nil {
		return result, wrap(err)
	}
	if err := createImportID(result.ImageID); err != nil {
		return result, wrap(response.Fail(err))
	}
	result.Created.Image, err = decodeCreatedImportImage(response)
	if err != nil {
		return result, wrap(err)
	}
	formats := response
	if policy.Import.Method == imageimport.GlanceDirectMethod {
		status, err := createImportString(response, fields, "status")
		if err != nil {
			return result, wrap(err)
		}
		if status != "queued" {
			return result, wrap(response.Fail(uploadInvalid("created image must have exact queued status")))
		}
		if err := check(); err != nil {
			return result, wrap(err)
		}
		// Canonical ID and status are independent of case aliases or incidental
		// IDs in the native model returned by creation or staging.
		seed := &Image{ID: result.ImageID, Status: images.ImageStatus(status)}
		result.Staged, err = stageAPI.StageKnownImage(ctx, seed, input.Data, imagedata.WithStageOpts(policy.Stage))
		if err != nil {
			return result, wrap(err)
		}
		formats = &rest.Response{Body: result.Staged.Body, Header: result.Staged.Header, StatusCode: result.Staged.StatusCode}
		fields, err = createImportObject(formats)
		if err != nil {
			return result, wrap(err)
		}
	}
	container, err := createImportString(formats, fields, "container_format")
	if err != nil {
		return result, wrap(err)
	}
	disk, err := createImportString(formats, fields, "disk_format")
	if err != nil {
		return result, wrap(err)
	}
	if err := check(); err != nil {
		return result, wrap(err)
	}
	seed := &Image{ID: result.ImageID, ContainerFormat: container, DiskFormat: disk}
	result.Imported, err = importAPI.ImportKnownImage(ctx, seed, imageimport.WithImportOpts(policy.Import))
	if err != nil {
		return result, wrap(err)
	}
	if policy.Wait != nil {
		if err := check(); err != nil {
			return result, wrap(err)
		}
		collection := resource.NewCollection(resource.Adapter[Image]{
			Kind: "image",
			Get: func(waitCtx context.Context, id string) (*Image, error) {
				if err := check(); err != nil {
					return nil, err
				}
				return images.Get(waitCtx, &client, id).Extract()
			},
			ID:     func(value *Image) string { return value.ID },
			Status: func(value *Image) string { return string(value.Status) },
		})
		result.Ready, err = collection.Wait(ctx, resource.ID(result.ImageID), "active", waitOptions...)
	}
	return result, wrap(err)
}

func prepareCreateImportMetadata(name string, value CreateImportMetadataOpts) (map[string]any, error) {
	visibility := VisibilityPrivate
	options := uploadImageOptions{base: images.CreateOpts{Name: name, DiskFormat: "qcow2", ContainerFormat: "bare", Visibility: &visibility}, properties: make(map[string]json.RawMessage)}
	var apply []UploadImageOption
	if value.DiskFormat != "" {
		apply = append(apply, WithDiskFormat(value.DiskFormat))
	}
	if value.ContainerFormat != "" {
		apply = append(apply, WithContainerFormat(value.ContainerFormat))
	}
	if value.Visibility != nil {
		apply = append(apply, WithVisibility(*value.Visibility))
	}
	if value.Protected != nil {
		apply = append(apply, WithProtected(*value.Protected))
	}
	if value.Hidden != nil {
		apply = append(apply, WithHidden(*value.Hidden))
	}
	if value.MinDisk != nil {
		apply = append(apply, WithMinDisk(*value.MinDisk))
	}
	if value.MinRAM != nil {
		apply = append(apply, WithMinRAM(*value.MinRAM))
	}
	apply = append(apply, WithTags(value.Tags...), WithProperties(value.Properties))
	for _, value := range append([]string{value.DiskFormat, value.ContainerFormat}, value.Tags...) {
		if !utf8.ValidString(value) {
			return nil, uploadInvalid("image metadata must be valid UTF-8")
		}
	}
	for _, option := range apply {
		if err := option(&options); err != nil {
			return nil, err
		}
	}
	return prepareUploadMetadata(options)
}

func createImportObject(response *rest.Response) (map[string]json.RawMessage, error) {
	if !utf8.Valid(response.Body) {
		return nil, response.Fail(fmt.Errorf("image response must be valid UTF-8"))
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(response.Body, &fields); err != nil {
		return nil, response.Fail(err)
	}
	if fields == nil {
		return nil, response.Fail(fmt.Errorf("image response must be a JSON object"))
	}
	return fields, nil
}

func createImportString(response *rest.Response, fields map[string]json.RawMessage, key string) (string, error) {
	body, ok := fields[key]
	if !ok || string(body) == "null" {
		return "", response.Fail(uploadInvalid("canonical %s is required", key))
	}
	var value string
	if err := json.Unmarshal(body, &value); err != nil {
		return "", response.Fail(err)
	}
	if value == "" {
		return "", response.Fail(uploadInvalid("canonical %s must not be empty", key))
	}
	return value, nil
}

func createImportID(value string) error {
	if err := resource.ID(value).Validate(); err != nil {
		return err
	}
	for _, char := range value {
		if unicode.IsSpace(char) || unicode.IsControl(char) || strings.ContainsRune(":\\", char) {
			return uploadInvalid("created image ID must be a single safe URL segment")
		}
	}
	return nil
}

func decodeCreatedImportImage(response *rest.Response) (*Image, error) {
	var body map[string]any
	decoder := json.NewDecoder(bytes.NewReader(response.Body))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return nil, response.Fail(err)
	}
	var native images.CreateResult
	native.Body, native.Header = body, response.Header.Clone()
	image, err := native.Extract()
	if err != nil {
		return nil, response.Fail(err)
	}
	return image, nil
}

func wrapCreateImportError(ctx context.Context, err error) error {
	if err != nil && ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return request.Wrap("CreateAndImport", "image", err)
}
