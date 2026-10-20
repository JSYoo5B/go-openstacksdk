package image

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// ImageRecordCreateRequest supplies a formal name, optional filename or tagged
// data, and kwargs. The SDK owns defaults and branching; readers are borrowed.
type ImageRecordCreateRequest struct {
	Name, Filename string
	Data           ImageRecordCreateData
	Attributes     map[string]any
}

type preparedImageRecordCreateWorkflow struct {
	*preparedImageRecord
	input  ImageRecordCreateRequest
	policy ImageRecordCreateOpts
	cloud  ImageCreatePolicy
	attrs  map[string]json.RawMessage
}

func (s *Service) prepareImageRecordCreateWorkflow(ctx context.Context, input ImageRecordCreateRequest, options []ImageRecordCreateOption) (*preparedImageRecordCreateWorkflow, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	p, err := s.captureImageRecord(rest.WithOperationSources(ctx))
	if err != nil {
		return nil, err
	}
	if !utf8.ValidString(input.Name) || !utf8.ValidString(input.Filename) {
		return nil, uploadInvalid("image create name and filename must be UTF-8")
	}
	data, err := captureImageRecordCreateData(p.ctx, p.check, input.Data)
	if err != nil {
		return nil, err
	}
	input.Data = data
	attrs, err := captureImageRecordUploadAttributes(p.ctx, p.check, input.Attributes)
	if err != nil {
		return nil, err
	}
	input.Attributes = nil
	cloud, err := copyImageCreatePolicy(p.ctx, p.check, s.dependencies.CreatePolicy)
	if err != nil {
		return nil, err
	}
	if cloud.ImageFormat == nil {
		cloud.ImageFormat = json.RawMessage(`"qcow2"`)
	}
	if cloud.UseTasks == nil {
		cloud.UseTasks = json.RawMessage(`false`)
	}
	prepared := &preparedImageRecordCreateWorkflow{preparedImageRecord: p, input: input, cloud: cloud, attrs: make(map[string]json.RawMessage, len(attrs))}
	for key, value := range attrs {
		prepared.attrs[key] = bytes.Clone(value.(json.RawMessage))
	}
	err = p.prepare(func(ctx context.Context, check func(context.Context) error) (map[string]string, error) {
		policy, err := prepareImageRecordCreateOptions(ctx, check, options)
		if err != nil {
			return nil, err
		}
		prepared.policy = policy
		for key, value := range policy.Attributes {
			prepared.attrs[key] = bytes.Clone(value.(json.RawMessage))
		}
		return policy.Headers, check(ctx)
	})
	return prepared, errors.Join(err, p.check(p.ctx))
}

func imageCreateRawDefault(raw json.RawMessage, fallback string) json.RawMessage {
	if raw == nil {
		return json.RawMessage(fallback)
	}
	return raw
}
func imageCreateTruthy(raw json.RawMessage) (bool, error) {
	return cloudfilter.PythonTruthy(imageCreateRawDefault(raw, "null"))
}

func (p *preparedImageRecordCreateWorkflow) preface() error {
	if p.input.Filename != "" && p.input.Data.truthy() {
		return uploadInvalid("filename and data are mutually exclusive")
	}
	if p.policy.Container == nil {
		container := "images"
		p.policy.Container = &container
	}
	truthy, err := imageCreateTruthy(p.policy.DiskFormat)
	if err != nil {
		return err
	}
	if !truthy {
		p.policy.DiskFormat = bytes.Clone(p.cloud.ImageFormat)
	}
	truthy, err = imageCreateTruthy(p.policy.ContainerFormat)
	if err != nil {
		return err
	}
	if !truthy {
		p.policy.ContainerFormat = json.RawMessage(`"bare"`)
	}
	if p.input.Filename == "" && !p.input.Data.truthy() {
		// Preserve the pinned helper's actual extension inference: its second
		// branch tests isfile(name), not isfile(name+extension).
		if err := p.check(p.ctx); err != nil {
			return err
		}
		if info, statErr := os.Stat(p.input.Name); statErr == nil && info.Mode().IsRegular() {
			p.input.Filename = p.input.Name
			p.input.Name = imageCreateFilenameStem(p.input.Name)
		} else {
			format, err := decodeImageRecordString(p.cloud.ImageFormat, "cloud image format")
			if err != nil {
				return err
			}
			candidate := p.input.Name + "." + format
			if _, candidateErr := os.Stat(candidate); candidateErr == nil {
				if info, nameErr := os.Stat(p.input.Name); nameErr == nil && info.Mode().IsRegular() {
					p.input.Filename = candidate
					p.input.Name = filepath.Base(p.input.Name)
				}
			}
		}
		if err := p.check(p.ctx); err != nil {
			return err
		}
	}
	validate, err := imageCreateTruthy(p.policy.ValidateChecksum)
	if err != nil {
		return err
	}
	if validate && p.input.Data.truthy() && p.input.Data.kind != imageRecordCreateDataBytes {
		return uploadInvalid("checksum validation requires bytes or filename")
	}
	md5Truthy, err := imageCreateTruthy(p.policy.MD5)
	if err != nil {
		return err
	}
	shaTruthy, err := imageCreateTruthy(p.policy.SHA256)
	if err != nil {
		return err
	}
	if validate && !md5Truthy && !shaTruthy {
		var reader io.Reader
		var file *os.File
		if p.input.Filename != "" {
			file, err = os.Open(p.input.Filename)
			if err = errors.Join(err, p.check(p.ctx)); err != nil {
				if file != nil {
					err = errors.Join(err, file.Close())
				}
				return err
			}
			reader = file
		} else if p.input.Data.truthy() && p.input.Data.kind == imageRecordCreateDataBytes {
			reader = bytes.NewReader(p.input.Data.data)
		}
		if reader != nil {
			first, second := md5.New(), sha256.New()
			_, err = io.CopyBuffer(io.MultiWriter(first, second), imageBinaryReader{Reader: reader, check: func() error { return p.check(p.ctx) }}, make([]byte, 8192))
			if file != nil {
				err = errors.Join(err, file.Close())
			}
			if err = errors.Join(err, p.check(p.ctx)); err != nil {
				return err
			}
			p.policy.MD5, _ = json.Marshal(hex.EncodeToString(first.Sum(nil)))
			p.policy.SHA256, _ = json.Marshal(hex.EncodeToString(second.Sum(nil)))
		}
	}
	return p.check(p.ctx)
}

func (p *preparedImageRecordCreateWorkflow) existing() (*ImageRecord, error) {
	duplicates, err := imageCreateTruthy(p.policy.AllowDuplicates)
	if err != nil {
		return nil, err
	}
	if duplicates {
		return nil, p.check(p.ctx)
	}
	if err := validateImageRecordIdentity(p.input.Name); err != nil {
		return nil, err
	}
	parameters, err := prepareImageRecordList(p.ctx, p.check, nil)
	if err != nil {
		return nil, err
	}
	record, err := findPreparedImageRecord(p.preparedImageRecord, p.input.Name, FindImageRecordOpts{}, parameters)
	if err != nil || record == nil {
		return nil, err
	}
	props := record.Resource.Body["properties"]
	truthy, err := imageCreateTruthy(props)
	if err != nil {
		return nil, err
	}
	values := make(map[string]json.RawMessage)
	if truthy {
		if err := json.Unmarshal(props, &values); err != nil {
			return nil, errors.Join(uploadInvalid("existing image properties must be a mapping"), err)
		}
	}
	matching, hasExpected := true, false
	for _, pair := range []struct {
		expected       json.RawMessage
		modern, legacy string
	}{{p.policy.MD5, imageCreateMD5Key, "owner_specified.shade.md5"}, {p.policy.SHA256, imageCreateSHA256Key, "owner_specified.shade.sha256"}} {
		truthy, err := imageCreateTruthy(pair.expected)
		if err != nil {
			return nil, err
		}
		if !truthy {
			continue
		}
		hasExpected = true
		actual, present := values[pair.modern]
		if !present {
			actual, present = values[pair.legacy]
		}
		if !present {
			actual = json.RawMessage(`""`)
		}
		equal, err := jsonfilter.EqualPythonJSON(actual, pair.expected)
		if err != nil {
			return nil, err
		}
		matching = matching && equal
	}
	if hasExpected && matching {
		return record, p.check(p.ctx)
	}
	return nil, p.check(p.ctx)
}

const imageCreateMD5Key = "owner_specified.openstack.md5"
const imageCreateSHA256Key = "owner_specified.openstack.sha256"
const imageCreateObjectKey = "owner_specified.openstack.object"

func (p *preparedImageRecordCreateWorkflow) imageKwargs() (map[string]json.RawMessage, map[string]json.RawMessage, error) {
	attrs := copyTaskRawMap(p.attrs)
	enabled, err := cloudfilter.PythonTruthy(imageCreateRawDefault(p.policy.DisableVendorAgent, "true"))
	if err != nil {
		return nil, nil, err
	}
	if enabled {
		var vendor map[string]json.RawMessage
		if p.cloud.RawVendorAgent != nil {
			vendor, err = imageCreateMappingUpdate(p.cloud.RawVendorAgent)
		} else {
			vendor = make(map[string]json.RawMessage, len(p.cloud.DisableVendorAgent))
			for key, value := range p.cloud.DisableVendorAgent {
				raw, ok := value.(json.RawMessage)
				if !ok {
					return nil, nil, uploadInvalid("image vendor value was not captured")
				}
				vendor[key] = bytes.Clone(raw)
			}
		}
		if err != nil {
			return nil, nil, err
		}
		for key, value := range vendor {
			attrs[key] = bytes.Clone(value)
		}
	}
	properties := make(map[string]json.RawMessage)
	if raw, present := attrs["properties"]; present {
		if len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{' {
			return nil, nil, uploadInvalid("legacy image properties must be a mapping")
		}
		if err := json.Unmarshal(raw, &properties); err != nil {
			return nil, nil, err
		}
		delete(attrs, "properties")
	}
	for _, pair := range []struct {
		key string
		raw json.RawMessage
	}{{imageCreateMD5Key, p.policy.MD5}, {imageCreateSHA256Key, p.policy.SHA256}} {
		truthy, err := imageCreateTruthy(pair.raw)
		if err != nil {
			return nil, nil, err
		}
		if truthy {
			properties[pair.key] = bytes.Clone(pair.raw)
		} else {
			properties[pair.key] = json.RawMessage(`""`)
		}
	}
	properties[imageCreateObjectKey], _ = json.Marshal(*p.policy.Container + "/" + p.input.Name)
	for key, value := range properties {
		attrs[key] = bytes.Clone(value)
	}
	root := make(map[string]json.RawMessage)
	for _, pair := range []struct {
		key string
		raw json.RawMessage
	}{{"disk_format", p.policy.DiskFormat}, {"container_format", p.policy.ContainerFormat}, {"tags", p.policy.Tags}} {
		truthy, err := imageCreateTruthy(pair.raw)
		if err != nil {
			return nil, nil, err
		}
		if truthy {
			root[pair.key] = bytes.Clone(pair.raw)
		}
	}
	return root, attrs, p.check(p.ctx)
}

// dict.update accepts objects and JSON-domain lists of two-element pairs.
// Explicit null and malformed pairs fail at the Source vendor merge phase.
func imageCreateMappingUpdate(raw json.RawMessage) (map[string]json.RawMessage, error) {
	result := make(map[string]json.RawMessage)
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '{' {
		err := json.Unmarshal(raw, &result)
		return result, err
	}
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var pairs []json.RawMessage
		if err := json.Unmarshal(raw, &pairs); err != nil {
			return nil, err
		}
		for _, pair := range pairs {
			var values []json.RawMessage
			if err := json.Unmarshal(pair, &values); err != nil || len(values) != 2 {
				return nil, uploadInvalid("image vendor pairs must have two elements")
			}
			key, err := decodeImageRecordString(values[0], "image vendor key")
			if err != nil {
				return nil, err
			}
			result[key] = bytes.Clone(values[1])
		}
		return result, nil
	}
	return nil, uploadInvalid("image vendor configuration must be a mapping or pair list")
}

func (p *preparedImageRecordCreateWorkflow) metadata(root, properties map[string]json.RawMessage) (*preparedImageRecordCreate, error) {
	attrs := copyTaskRawMap(root)
	for key, value := range properties {
		converted, err := convertImageRecordProperty(key, value)
		if err != nil {
			return nil, err
		}
		attrs[key] = converted
	}
	for key, value := range p.policy.Meta {
		raw, ok := value.(json.RawMessage)
		if !ok {
			return nil, uploadInvalid("image meta value was not captured")
		}
		attrs[key] = bytes.Clone(raw)
	}
	attrs["name"], _ = json.Marshal(p.input.Name)
	return prepareImageRecordCreate(p.preparedImageRecord, attrs)
}

// Python splitext preserves leading-dot basenames, including repeated dots.
func imageCreateFilenameStem(name string) string {
	base := filepath.Base(name)
	dot := strings.LastIndexByte(base, '.')
	if dot > 0 && strings.Trim(base[:dot], ".") != "" {
		return base[:dot]
	}
	return base
}
