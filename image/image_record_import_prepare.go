package image

import (
	"context"
	"errors"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

type preparedImageRecordImport struct {
	*preparedImageRecord
	seed         *ImageRecord
	identity     string
	body         map[string]any
	headerPolicy rest.RequestHeaderPolicy
}

func (s *Service) prepareImageRecordImport(ctx context.Context, input ImageRecordImportRequest, options []ImageRecordImportOption) (*preparedImageRecordImport, error) {
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return nil, err
	}
	seed, identity, err := imageRecordMutationSeed(input.ID, input.Record)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return nil, err
	}
	// Store selection belongs to this invocation. A captured configured legacy
	// header is discarded rather than silently becoming an additional selection.
	// The original source remains untouched and its ordinary headers stay frozen.
	for key := range p.client.MoreHeaders {
		if strings.EqualFold(key, "X-Image-Meta-Store") {
			delete(p.client.MoreHeaders, key)
		}
	}
	prepared := &preparedImageRecordImport{preparedImageRecord: p, seed: seed, identity: identity}
	if err := p.prepare(func(opctx context.Context, check func(context.Context) error) (map[string]string, error) {
		if err := errors.Join(projectImageRecordMutationSeed(seed, p.location), check(opctx)); err != nil {
			return nil, err
		}
		policy, err := prepareImageRecordImportOptions(opctx, check, options)
		if err != nil {
			return nil, err
		}
		if err := validateImageRecordImportStores(policy); err != nil {
			return nil, errors.Join(err, check(opctx))
		}
		// JSON truthiness reproduces the untyped Source descriptors. Do not impose
		// the legacy native-image profile's canonical nonempty string restriction.
		for _, field := range []string{"container_format", "disk_format"} {
			present, err := cloudfilter.PythonTruthy(seed.Resource.Body[field])
			if err != nil {
				return nil, errors.Join(err, check(opctx))
			}
			if !present {
				return nil, uploadInvalid("both container_format and disk_format are required for importing an image")
			}
		}
		body, headers, err := compileImageRecordImport(policy)
		prepared.body = body
		if policy.Store != nil {
			prepared.headerPolicy.Values = map[string]string{"X-Image-Meta-Store": headers["X-Image-Meta-Store"]}
		} else {
			prepared.headerPolicy.Absent = []string{"X-Image-Meta-Store"}
		}
		return headers, errors.Join(err, check(opctx))
	}); err != nil {
		return nil, err
	}
	return prepared, p.check(p.ctx)
}
