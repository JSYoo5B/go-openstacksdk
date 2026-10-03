package v1

import (
	"context"
	"fmt"
	"strings"
	"time"

	"gophercloudsdk/request"
)

// GenerateFormSignature signs literal FormPost fields. Automatic discovery reads
// container then account metadata; an explicit key performs no HTTP request.
func (s *Service) GenerateFormSignature(ctx context.Context, container string, input FormSignatureInput, options ...GenerateFormSignatureOption) (*FormSignatureResult, error) {
	captured := time.Now().Unix()
	p, err := s.captureTempURLKey(ctx)
	if err != nil {
		return nil, request.Wrap("GenerateFormSignature", "objectstorage", err)
	}
	fail := func(err error) (*FormSignatureResult, error) {
		return nil, request.Wrap("GenerateFormSignature", "objectstorage", tempURLKeyContextError(ctx, err))
	}
	if err = signingContainer(container); err != nil {
		return fail(err)
	}
	if !signingPathText(input.ObjectPrefix) {
		return fail(tempURLKeyInvalid("invalid literal form object prefix"))
	}
	if !signingText(input.RedirectURL) || len(input.RedirectURL) > 4096 || strings.HasSuffix(input.RedirectURL, "-") {
		return fail(tempURLKeyInvalid("invalid literal form redirect"))
	}
	if input.MaxFileSize < 1 || input.MaxUploadCount < 1 || input.Timeout < 1 {
		return fail(tempURLKeyInvalid("positive form limits and timeout are required"))
	}
	path, endpoint, err := signingFormPath(p.endpoint, container, input.ObjectPrefix)
	if err != nil {
		return fail(err)
	}
	cfg, optionErr := applyGenerateFormSignatureOptions(options)
	reads := GetTempURLKeyOpts{Headers: cfg.Headers, Newest: cfg.Newest}
	if cfg.Key == nil {
		reads.Container = container
	}
	if err = p.finish(ctx, reads, optionErr); err != nil {
		return fail(err)
	}
	digest, epoch, err := signingOptions(ctx, cfg.Key, cfg.Digest, cfg.Timestamp, captured)
	if err != nil {
		return fail(err)
	}
	expires, err := signingExpires(epoch, input.Timeout, false)
	if err != nil {
		return fail(err)
	}
	key := cfg.Key
	var discovery *TempURLKeyResult
	if key == nil {
		discovery, err = discoverSigningKey(ctx, p, reads)
		if err != nil {
			if discovery == nil {
				return fail(err)
			}
			return &FormSignatureResult{Discovery: discovery}, request.Wrap("GenerateFormSignature", "objectstorage", err)
		}
		key = discovery.Key
	}
	body := fmt.Sprintf("%s\n%s\n%d\n%d\n%d", path, input.RedirectURL, input.MaxFileSize, input.MaxUploadCount, expires)
	signature := signingMAC(key, digest, body)
	if err = observeSigningDiscovery(ctx, p, discovery, nil); err != nil {
		if discovery == nil {
			return fail(err)
		}
		return &FormSignatureResult{Discovery: discovery}, request.Wrap("GenerateFormSignature", "objectstorage", err)
	}
	return &FormSignatureResult{Path: path, URL: endpoint.String(), Signature: signature, Expires: expires, Digest: digest, Discovery: discovery}, nil
}
