package v1

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gophercloudsdk/request"
)

// GenerateTempURL signs a decoded literal Swift path. Automatic key discovery is
// account-only; native Objects.CreateTempURL retains its independent policy.
func (s *Service) GenerateTempURL(ctx context.Context, path string, seconds int64, method string, options ...GenerateTempURLOption) (*TempURLResult, error) {
	captured := time.Now().Unix()
	p, err := s.captureTempURLKey(ctx)
	if err != nil {
		return nil, request.Wrap("GenerateTempURL", "objectstorage", err)
	}
	fail := func(err error) (*TempURLResult, error) {
		return nil, request.Wrap("GenerateTempURL", "objectstorage", tempURLKeyContextError(ctx, err))
	}
	// Prefix changes only whether an empty object is meaningful; structural checks are early.
	if err = signingTempPath(path, true); err != nil {
		return fail(err)
	}
	if seconds < 0 || !tempURLKeyToken(method) {
		return fail(tempURLKeyInvalid("invalid expiration or HTTP method"))
	}
	cfg, optionErr := applyGenerateTempURLOptions(options)
	reads := GetTempURLKeyOpts{Headers: cfg.Headers, Newest: cfg.Newest}
	if err = p.finish(ctx, reads, optionErr); err != nil {
		return fail(err)
	}
	if err = signingTempPath(path, cfg.Prefix); err != nil {
		return fail(err)
	}
	if err = signingIPRange(cfg.IPRange); err != nil {
		return fail(err)
	}
	digest, epoch, err := signingOptions(ctx, cfg.Key, cfg.Digest, cfg.Timestamp, captured)
	if err != nil {
		return fail(err)
	}
	expires, err := signingExpires(epoch, seconds, cfg.Absolute)
	if err != nil {
		return fail(err)
	}
	expiration := strconv.FormatInt(expires, 10)
	if cfg.ISO8601 {
		instant := time.Unix(expires, 0).UTC()
		if instant.Year() < 1 || instant.Year() > 9999 {
			return fail(tempURLKeyInvalid("ISO8601 expiration outside year1..9999"))
		}
		expiration = instant.Format("2006-01-02T15:04:05Z")
	}
	key := cfg.Key
	var discovery *TempURLKeyResult
	if key == nil {
		discovery, err = discoverSigningKey(ctx, p, reads)
		if err != nil {
			if discovery == nil {
				return fail(err)
			}
			return &TempURLResult{Discovery: discovery}, request.Wrap("GenerateTempURL", "objectstorage", err)
		}
		key = discovery.Key
	}
	signedPath := path
	if cfg.Prefix {
		signedPath = "prefix:" + path
	}
	body := fmt.Sprintf("%s\n%d\n%s", strings.ToUpper(method), expires, signedPath)
	if cfg.IPRange != "" {
		body = "ip=" + cfg.IPRange + "\n" + body
	}
	signature := signingMAC(key, digest, body)
	query := url.Values{"temp_url_sig": {signature}, "temp_url_expires": {expiration}}
	if cfg.Prefix {
		query.Set("temp_url_prefix", strings.SplitN(path, "/", 5)[4])
	}
	if cfg.IPRange != "" {
		query.Set("temp_url_ip_range", cfg.IPRange)
	}
	output := (&url.URL{Path: path, RawQuery: query.Encode()}).String()
	if err = observeSigningDiscovery(ctx, p, discovery, nil); err != nil {
		if discovery == nil {
			return fail(err)
		}
		return &TempURLResult{Discovery: discovery}, request.Wrap("GenerateTempURL", "objectstorage", err)
	}
	return &TempURLResult{Path: path, URL: output, Signature: signature, Expires: expires, Digest: digest, Discovery: discovery}, nil
}
