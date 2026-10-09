package image

import (
	"context"
	"encoding/json"
	"errors"
	"slices"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
)

// ImageRecordCreateResult retains the selected image, raw task, and independent
// phase receipts. Later failures preserve accepted work and cleanup evidence.
// A Task without Wait is not an active ImageRecord or a completed import.
type ImageRecordCreateResult struct {
	Outcome                                                      string
	Reused                                                       bool
	Record, Updated                                              *ImageRecord
	Created, Uploaded, Staged, ChecksumFetched, TaskImageFetched *ImageUploadResponse
	Imported                                                     *imageimport.ImportResult
	Swift                                                        *ImageRecordCreateSwiftResult
	Task, TaskDiagnostic                                         *ImageRecordCreateTask
	TaskWait                                                     *ImageRecordCreateTaskWaitResult
	TaskDiagnosticResponse                                       *ImageUploadResponse
	Cleanup                                                      *ImageRecordDeleteResult
	Checksum                                                     *ImageRecordCreateChecksum
	Warnings                                                     []string
}

// ImageRecordCreateChecksum reports the optional metadata checksum comparison.
// A falsey remote checksum is accepted without claiming content verification.
type ImageRecordCreateChecksum struct {
	MD5, SHA256, Actual json.RawMessage
	Compared, Matched   bool
}

// CreateImageRecord owns the complete modern Glance creation workflow: defaults,
// optional checksum/duplicate reuse, metadata creation, direct upload/import or
// configured Swift Task composition, and selected failure cleanup. Reader data
// remains borrowed. Wait applies only to the Task route, as in the pinned SDK.
func (s *Service) CreateImageRecord(ctx context.Context, input ImageRecordCreateRequest, options ...ImageRecordCreateOption) (*ImageRecordCreateResult, error) {
	fail := func(result *ImageRecordCreateResult, err error) (*ImageRecordCreateResult, error) {
		return result, wrapImageMutationError(ctx, "CreateImageRecord", err)
	}
	p, err := s.prepareImageRecordCreateWorkflow(ctx, input, slices.Clone(options))
	if err != nil {
		return fail(nil, err)
	}
	if err := p.preface(); err != nil {
		return fail(nil, err)
	}
	existing, err := p.existing()
	if err != nil {
		return fail(nil, err)
	}
	if existing != nil {
		return &ImageRecordCreateResult{Outcome: "reused", Reused: true, Record: existing}, nil
	}
	root, properties, err := p.imageKwargs()
	if err != nil {
		return fail(nil, err)
	}
	selected, err := imageCreateTruthy(p.policy.Import.Method)
	if err != nil {
		return fail(nil, err)
	}
	selected = selected || p.input.Filename != "" || p.input.Data.truthy()
	result := &ImageRecordCreateResult{}
	if !selected {
		metadata, err := p.metadata(root, properties)
		if err != nil {
			return fail(result, err)
		}
		record, response, err := createPreparedImageRecord(metadata)
		result.Record, result.Created = record, imageUploadResponse(response)
		if err == nil {
			result.Outcome = "metadata-only"
		}
		return fail(result, errors.Join(err, p.check(p.ctx)))
	}
	if legacy, present := properties["is_public"]; present {
		truthy, err := imageCreateTruthy(legacy)
		if err != nil {
			return fail(result, err)
		}
		delete(properties, "is_public")
		visibility := "private"
		if truthy {
			visibility = "public"
		}
		root["visibility"], _ = json.Marshal(visibility)
		result.Warnings = append(result.Warnings, "The is_public property is deprecated by Glance v2; use visibility instead")
	}
	tasks, err := imageCreateTruthy(p.cloud.UseTasks)
	if err != nil {
		return fail(result, err)
	}
	if tasks {
		explicitImport, importErr := imageCreateTruthy(p.policy.UseImport)
		if importErr != nil {
			return fail(result, importErr)
		}
		if explicitImport {
			return fail(result, uploadInvalid("Glance Task and Import APIs are mutually exclusive"))
		}
		err = p.createUsingTask(root, properties, result)
	} else {
		err = p.createUploaded(root, properties, result)
	}
	return fail(result, errors.Join(err, p.check(p.ctx)))
}
