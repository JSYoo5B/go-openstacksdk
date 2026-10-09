package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	objectapi "github.com/JSYoo5B/go-openstacksdk/objectstorage/v1"
	"github.com/JSYoo5B/go-openstacksdk/objectstorage/v1/objects"
	"github.com/gophercloud/gophercloud/v2"
	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
)

// ImageRecordCreateSwiftResult preserves container establishment, object upload
// and the optional wait-finally cleanup as separate actual HTTP phases. It does
// not imply that a container, object or image was rolled back after a failure.
type ImageRecordCreateSwiftResult struct {
	ContainerDiscovery, ContainerCreated, ContainerFetched *ImageUploadResponse
	Object                                                 *objects.ImageImportObjectResult
	Cleanup                                                *objects.DeleteObjectResult
}

// createUsingTask follows the whole Source Task branch, including its late
// object-store selection and cleanup boundary. The object is retained when
// upload/Task creation fails or wait is false; every entered wait cleans it.
func (p *preparedImageRecordCreateWorkflow) createUsingTask(root, properties map[string]json.RawMessage, result *ImageRecordCreateResult) (err error) {
	service, err := p.createSwiftService()
	if err != nil {
		return err
	}
	objectKey, err := decodeImageRecordString(properties[imageCreateObjectKey], "image task object key")
	if err != nil {
		return err
	}
	container := strings.SplitN(objectKey, "/", 2)[0]
	root = copyTaskRawMap(root)
	delete(root, "disk_format")
	delete(root, "container_format")
	result.Swift = &ImageRecordCreateSwiftResult{}
	if err := p.ensureCreateImageContainer(service, container, result.Swift); err != nil {
		return err
	}
	md5, sha256 := "", ""
	if !p.input.Data.present() {
		// Ordinary data ignores supplied hashes in Source. Filename hashes cross
		// the concrete Swift string bridge only when that branch is selected,
		// after container establishment and before its file upload phases.
		md5, err = decodeImageRecordString(properties[imageCreateMD5Key], "image task MD5")
		if err != nil {
			return err
		}
		sha256, err = decodeImageRecordString(properties[imageCreateSHA256Key], "image task SHA256")
		if err != nil {
			return err
		}
	}
	result.Swift.Object, err = service.Objects.CreateImageImportObject(p.ctx, objects.ImageImportObjectRequest{
		Container: container, Name: p.input.Name, Filename: p.input.Filename,
		Data: p.input.Data.openReader(), DataPresent: p.input.Data.present(),
		MD5: md5, SHA256: sha256, Headers: maps.Clone(p.policy.SwiftHeaders),
	})
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return err
	}
	input, err := json.Marshal(map[string]any{
		"import_from":      container + "/" + p.input.Name,
		"image_properties": map[string]any{"name": p.input.Name},
	})
	if err != nil {
		return err
	}
	result.Task, result.Created, err = createImageRecordTask(p.preparedImageRecord, map[string]json.RawMessage{"type": json.RawMessage(`"import"`), "input": input})
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return err
	}
	result.Outcome = "task"
	wait, err := imageCreateTruthy(p.policy.Wait)
	if err != nil || !wait {
		return err
	}
	// Source's finally starts only after a successful create_task and a truthy
	// wait flag. Retain all cleanup evidence and preserve both failure causes.
	defer func() {
		var cleanupErr error
		result.Swift.Cleanup, cleanupErr = service.Objects.DeleteObject(p.ctx, container, p.input.Name, objects.WithDeleteObjectHeaders(p.policy.SwiftHeaders))
		err = errors.Join(err, cleanupErr, p.check(p.ctx))
	}()
	result.TaskWait, err = waitCreatedImageRecordTask(p.preparedImageRecord, result.Task, p.policy.Timeout)
	if err != nil {
		var failed *ImageRecordCreateTaskFailureError
		if errors.As(err, &failed) {
			original := result.Task
			if result.TaskWait != nil && result.TaskWait.OriginalTask != nil {
				original = result.TaskWait.OriginalTask
			}
			var diagnosticErr error
			result.TaskDiagnostic, result.TaskDiagnosticResponse, diagnosticErr = getCreatedImageRecordTask(p.preparedImageRecord, original)
			err = errors.Join(fmt.Errorf("image creation failed: %w", err), diagnosticErr)
		}
		return err
	}
	if result.TaskWait == nil || result.TaskWait.Task == nil {
		return uploadInvalid("image task wait requires the completed Task")
	}
	var taskResult map[string]json.RawMessage
	rawResult := imageRecordCreateTaskRaw(result.TaskWait.Task, "result")
	if len(bytes.TrimSpace(rawResult)) == 0 || bytes.TrimSpace(rawResult)[0] != '{' {
		return uploadInvalid("completed image Task result must be a mapping")
	}
	if err := json.Unmarshal(rawResult, &taskResult); err != nil {
		return err
	}
	id, err := imageRecordDeleteLiteral(taskResult["image_id"], "completed image Task")
	if err != nil {
		return err
	}
	seed, _, err := imageRecordMutationSeed(id, nil)
	if err != nil {
		return err
	}
	if err := projectImageRecordMutationSeed(seed, p.location); err != nil {
		return err
	}
	var imageResponse *rest.Response
	result.Record, imageResponse, err = fetchPreparedImageRecord(p.preparedImageRecord, seed, id)
	result.TaskImageFetched = imageUploadResponse(imageResponse)
	if err != nil {
		return err
	}
	var existing map[string]json.RawMessage
	rawProperties := result.Record.Resource.Body["properties"]
	if len(bytes.TrimSpace(rawProperties)) == 0 || bytes.TrimSpace(rawProperties)[0] != '{' {
		return uploadInvalid("completed image properties must be a mapping")
	}
	if err := json.Unmarshal(rawProperties, &existing); err != nil {
		return err
	}
	for key, raw := range properties {
		existing[key] = bytes.Clone(raw)
	}
	root["properties"], err = imageRecordObject(existing)
	if err != nil {
		return err
	}
	if err := validateImageRecordUpdateAttributes(root, false); err != nil {
		return err
	}
	updates, methods, err := normalizeImageRecord(root, nil, false, true)
	if err != nil {
		return err
	}
	prepared, err := prepareImageRecordNormalizedUpdate(p.preparedImageRecord, cloneImageRecord(result.Record), updates, methods)
	if err != nil {
		return err
	}
	installImageRecordUpdateHeaders(p.preparedImageRecord)
	result.Updated, err = commitImageRecordUpdate(prepared)
	if result.Updated != nil {
		result.Record = result.Updated
	}
	if err == nil {
		result.Outcome = "task-completed"
	}
	return err
}

func (p *preparedImageRecordCreateWorkflow) createSwiftService() (*objectapi.Service, error) {
	if err := p.check(p.ctx); err != nil {
		return nil, err
	}
	enabled := true
	if p.cloud.RawObjectStoreEnabled != nil {
		raw := string(bytes.TrimSpace(p.cloud.RawObjectStoreEnabled))
		if raw != "true" && raw != "false" {
			return nil, uploadInvalid("has_object_store must be a bool")
		}
		enabled = raw == "true"
	} else if p.cloud.ObjectStoreEnabled != nil {
		enabled = *p.cloud.ObjectStoreEnabled
	}
	unavailable := func() error {
		return uploadInvalid("the cloud %s is configured to use tasks for image upload, but no object-store service is available", p.cloud.CloudName)
	}
	if !enabled || p.service.dependencies.ObjectStorage == nil {
		return nil, unavailable()
	}
	service, err := p.service.dependencies.ObjectStorage(p.ctx)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, err
	}
	if service == nil || service.RawClient() == nil || service.Containers == nil || service.Objects == nil {
		return nil, unavailable()
	}
	source, containers, objectsAPI := service.RawClient(), service.Containers, service.Objects
	provider, endpoint, base, kind, version := source.ProviderClient, source.Endpoint, source.ResourceBase, source.Type, source.Microversion
	var observed error
	var mu sync.Mutex
	guard := func(ctx context.Context) error {
		var current error
		if service.RawClient() != source || service.Containers != containers || service.Objects != objectsAPI ||
			containers.RawClient() != source || objectsAPI.RawClient() != source || source.ProviderClient != provider ||
			source.Endpoint != endpoint || source.ResourceBase != base || source.Type != kind || source.Microversion != version {
			current = uploadInvalid("image task object-store service binding changed")
		}
		_, validationErr := imageCreateSwiftSourceHeaders(ctx, source)
		mu.Lock()
		defer mu.Unlock()
		observed = errors.Join(observed, current, validationErr)
		return observed
	}
	// Only this pure dependency guard is registered. Calling the outer image
	// check here would recursively re-enter the operation's source registry.
	rest.RegisterOperationSource(p.ctx, guard)
	return service, errors.Join(guard(p.ctx), p.check(p.ctx))
}

// Container establishment shares the guarded REST engine with the Swift APIs,
// while retaining Source's 200..399 profile and a clean physical missing HEAD.
func (p *preparedImageRecordCreateWorkflow) ensureCreateImageContainer(service *objectapi.Service, container string, result *ImageRecordCreateSwiftResult) error {
	if err := swift.CheckContainerName(container); err != nil {
		return errors.Join(uploadInvalid("invalid image task container"), err)
	}
	if !utf8.ValidString(container) || container == "." || container == ".." || strings.ContainsRune(container, '\\') {
		return uploadInvalid("invalid literal image task container")
	}
	for _, b := range []byte(container) {
		if b < 32 || b == 127 {
			return uploadInvalid("image task container must not contain controls")
		}
	}
	source := service.RawClient()
	headers, err := imageCreateSwiftSourceHeaders(p.ctx, source)
	if err != nil {
		return err
	}
	client := *source
	client.MoreHeaders = headers
	for key, value := range p.policy.SwiftHeaders {
		client.MoreHeaders[key] = value
	}
	target := client.ServiceURL(url.PathEscape(container))
	codes := append(imageRecordCodes(), http.StatusNotFound)
	response, err := rest.DoJSONGuarded(p.ctx, &client, p.check, http.MethodHead, target, nil, nil, codes...)
	result.ContainerDiscovery = imageUploadResponse(response)
	if err != nil || response == nil || response.StatusCode != http.StatusNotFound {
		return err
	}
	response, err = rest.DoJSONGuarded(p.ctx, &client, p.check, http.MethodPut, target, nil, nil, imageRecordCodes()...)
	result.ContainerCreated = imageUploadResponse(response)
	if err != nil {
		return err
	}
	response, err = rest.DoJSONGuarded(p.ctx, &client, p.check, http.MethodHead, target, nil, nil, codes...)
	result.ContainerFetched = imageUploadResponse(response)
	return err
}

func imageCreateSwiftSourceHeaders(ctx context.Context, source *gophercloud.ServiceClient) (map[string]string, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	if source == nil || source.ProviderClient == nil {
		return nil, uploadInvalid("image task object-store client is required")
	}
	if source.Type != "" && source.Type != "object-store" {
		return nil, uploadInvalid("image task object-store client type is required")
	}
	base := source.ResourceBaseURL()
	parsed, err := url.Parse(base)
	if err != nil || !utf8.ValidString(source.Endpoint) || !utf8.ValidString(base) || parsed.RawQuery != "" || parsed.ForceQuery || !strings.HasSuffix(base, "/") {
		return nil, uploadInvalid("invalid image task object-store endpoint or resource base")
	}
	if err := rest.ValidateTarget(source, base); err != nil {
		return nil, err
	}
	if !downloadHeaderValue(source.Microversion) || !utf8.ValidString(source.Microversion) {
		return nil, uploadInvalid("invalid image task object-store microversion")
	}
	headers := make(map[string]string, len(source.MoreHeaders))
	for key, value := range source.MoreHeaders {
		if key == "" || !downloadHeaderValue(value) || !utf8.ValidString(value) {
			return nil, uploadInvalid("invalid image task object-store header")
		}
		for _, b := range []byte(key) {
			if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(b)) {
				continue
			}
			return nil, uploadInvalid("invalid image task object-store header %q", key)
		}
		canonical := http.CanonicalHeaderKey(key)
		if _, exists := headers[canonical]; exists {
			return nil, uploadInvalid("aliased image task object-store header %q", key)
		}
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "x-container-meta-") || strings.HasPrefix(lower, "x-remove-container-meta-") {
			return nil, uploadInvalid("image task container metadata header %q is SDK owned", key)
		}
		switch lower {
		case "authorization", "x-auth-token", "host", "content-length", "transfer-encoding", "connection", "proxy-connection", "proxy-authorization", "upgrade", "trailer", "te", "x-newest":
			return nil, uploadInvalid("image task object-store header %q is SDK owned", key)
		}
		headers[canonical] = value
	}
	return headers, nil
}
