package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudread"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

type preparedImageRecord struct {
	*preparedTaskSource
	ctx      context.Context
	check    func(context.Context) error
	location json.RawMessage
}

// Capture fixed routing, provider, service bindings, headers and an outer guard
// before caller data or location/options callbacks can run. Source and caller
// failures are sticky; a native request context owns only its own deadline.
func (s *Service) captureImageRecord(ctx context.Context) (*preparedImageRecord, error) {
	if err := cloudread.Context(ctx); err != nil {
		return nil, err
	}
	source, err := s.captureTaskSource(ctx)
	if err != nil {
		return nil, err
	}
	api, images := s.API, s.Images
	resourceBase := source.source.ResourceBase
	outer := rest.OperationGuard(ctx)
	var observed error
	var observedMu sync.Mutex
	check := func(checkCtx context.Context) error {
		observedMu.Lock()
		prior := observed
		observedMu.Unlock()
		if prior != nil {
			return cloudread.ContextError(checkCtx, prior)
		}
		var binding, ancestor error
		if s.API != api || s.Images != images || source.source.ResourceBase != resourceBase || api != nil && api.RawClient() != source.source {
			binding = uploadInvalid("image record service binding changed")
		}
		if outer != nil {
			ancestor = outer(ctx)
		}
		// Request-owned timeouts must not poison the parent workflow guard.
		// Genuine source drift and caller cancellation still remain sticky.
		fresh := errors.Join(source.check(ctx), binding, ancestor)
		// Upload reads can overlap response handling. Do not race the sticky
		// cause, or hold its lock while invoking an ancestor callback.
		observedMu.Lock()
		observed = errors.Join(observed, fresh)
		current := observed
		observedMu.Unlock()
		return cloudread.ContextError(checkCtx, current)
	}
	opctx := rest.WithOperationGuard(ctx, check)
	if err := check(opctx); err != nil {
		return nil, err
	}
	return &preparedImageRecord{preparedTaskSource: source, ctx: opctx, check: check, location: json.RawMessage("null")}, nil
}

func (s *Service) prepareImageRecord(ctx context.Context, apply func(context.Context, func(context.Context) error) (map[string]string, error)) (*preparedImageRecord, error) {
	p, err := s.captureImageRecord(ctx)
	if err != nil {
		return nil, err
	}
	if err := p.prepare(apply); err != nil {
		return nil, err
	}
	return p, nil
}

// Get captures caller seeds before this location callback. List/Find use the
// wrapper above. All options observe exactly one current-location snapshot.
func (p *preparedImageRecord) prepare(apply func(context.Context, func(context.Context) error) (map[string]string, error)) error {
	if err := p.check(p.ctx); err != nil {
		return err
	}
	var err error
	if p.service.dependencies.CloudLocation != nil {
		facts, readErr := p.service.dependencies.CloudLocation()
		err = readErr
		if err == nil {
			p.location, err = facts.ForResource(nil, facts.Zone)
		}
	}
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return err
	}
	if apply == nil {
		return uploadInvalid("image record option preparation is required")
	}
	headers, err := apply(p.ctx, p.check)
	if err = errors.Join(err, p.check(p.ctx)); err != nil {
		return err
	}
	return errors.Join(p.preparedTaskSource.finish(p.ctx, headers, nil), p.check(p.ctx))
}

func imageRecordCodes() []int {
	codes := make([]int, 200)
	for i := range codes {
		codes[i] = 200 + i
	}
	return codes
}

// Decode only complete UTF-8 JSON strings whose Unicode escapes preserve one
// valid string identity. encoding/json otherwise replaces unpaired UTF-16
// surrogates with U+FFFD and could select a different transport target. The
// existing tag pairing check requires a complete quoted JSON string first.
func decodeImageRecordString(raw json.RawMessage, label string) (string, error) {
	if !utf8.Valid(raw) {
		return "", uploadInvalid("%s must be UTF-8", label)
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", errors.Join(uploadInvalid("%s must be a complete JSON string", label), err)
	}
	trimmed := bytes.TrimSpace(raw)
	// JSON null decodes into a string without an error, but is not an identity
	// string. Successful parsing above also bounds the existing escape scanner.
	if len(trimmed) < 2 || trimmed[0] != '"' {
		return "", uploadInvalid("%s must be a JSON string", label)
	}
	if !imageRecordTagUnicodeString(trimmed) {
		return "", uploadInvalid("%s must not contain unpaired UTF-16 surrogates", label)
	}
	return value, nil
}

func validateImageRecordIdentity(value string) error {
	if strings.TrimSpace(value) == "" || !utf8.ValidString(value) || value == "." || value == ".." {
		return uploadInvalid("image identity must be nonblank valid UTF-8 literal text")
	}
	for _, char := range value {
		if unicode.IsControl(char) {
			return uploadInvalid("image identity must not contain controls")
		}
	}
	return nil
}
func imageRecordText(value *ImageRecord, field string) string {
	if value == nil || value.Resource == nil {
		return ""
	}
	text, err := decodeImageRecordString(value.Resource.Body[field], "image record "+field)
	if err != nil {
		return ""
	}
	return text
}
