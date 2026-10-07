package v1

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"math"
	"net/netip"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	swift "github.com/gophercloud/gophercloud/v2/openstack/objectstorage/v1"
)

func cloneSigningKey(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte{}, value...)
}
func signingOptions(ctx context.Context, key []byte, digest TempURLDigest, timestamp *time.Time, captured int64) (TempURLDigest, int64, error) {
	if key != nil && len(key) == 0 {
		return "", 0, tempURLKeyContextError(ctx, tempURLKeyInvalid("explicit signing key is empty"))
	}
	if digest == "" {
		digest = TempURLDigestSHA1
	}
	switch digest {
	case TempURLDigestSHA1, TempURLDigestSHA256, TempURLDigestSHA512:
	default:
		return "", 0, tempURLKeyContextError(ctx, tempURLKeyInvalid("unsupported signing digest"))
	}
	if timestamp != nil {
		captured = timestamp.Unix()
	}
	if captured < 0 {
		return "", 0, tempURLKeyContextError(ctx, tempURLKeyInvalid("negative signing timestamp"))
	}
	return digest, captured, nil
}
func signingExpires(epoch, seconds int64, absolute bool) (int64, error) {
	if seconds < 0 {
		return 0, tempURLKeyInvalid("negative signing expiration")
	}
	if absolute {
		return seconds, nil
	}
	if seconds > math.MaxInt64-epoch {
		return 0, tempURLKeyInvalid("signing expiration overflows int64")
	}
	return epoch + seconds, nil
}
func signingText(value string) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for _, b := range []byte(value) {
		if b < 32 || b == 127 {
			return false
		}
	}
	return true
}
func signingPathText(value string) bool {
	if !signingText(value) || strings.ContainsRune(value, '\\') {
		return false
	}
	for _, part := range strings.Split(value, "/") {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}
func signingContainer(value string) error {
	if err := swift.CheckContainerName(value); err != nil {
		return fmt.Errorf("%w: %w", resource.ErrInvalidOption, err)
	}
	if !signingPathText(value) {
		return tempURLKeyInvalid("invalid literal signing container")
	}
	return nil
}
func signingFormPath(endpoint, container, prefix string) (string, *url.URL, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || !signingPathText(parsed.Path) || !strings.HasPrefix(parsed.Path, "/") {
		return "", nil, tempURLKeyInvalid("invalid form signing endpoint path")
	}
	// Remove one separator, then require all captured account-path slots.
	base := strings.TrimSuffix(parsed.Path, "/")
	parts := strings.Split(strings.TrimPrefix(base, "/"), "/")
	if len(parts) < 2 {
		return "", nil, tempURLKeyInvalid("form signing endpoint has no account path")
	}
	for _, part := range parts {
		if part == "" {
			return "", nil, tempURLKeyInvalid("form signing endpoint has an empty path slot")
		}
	}
	path := base + "/" + container + "/" + prefix
	parsed.Path, parsed.RawPath, parsed.RawQuery, parsed.Fragment = path, "", "", ""
	parsed.ForceQuery = false
	return path, parsed, nil
}
func signingTempPath(value string, prefix bool) error {
	if !signingPathText(value) {
		return tempURLKeyInvalid("invalid literal Temp URL path")
	}
	parts := strings.SplitN(value, "/", 5)
	if len(parts) != 5 || parts[0] != "" || parts[1] != "v1" || parts[2] == "" || parts[3] == "" || (!prefix && parts[4] == "") {
		return tempURLKeyInvalid("Temp URL requires /v1/account/container/object")
	}
	return nil
}
func signingIPRange(value string) error {
	if value == "" {
		return nil
	}
	if addr, err := netip.ParseAddr(value); err == nil && addr.Zone() == "" {
		return nil
	}
	if prefix, err := netip.ParsePrefix(value); err == nil && prefix.Addr().Zone() == "" && prefix == prefix.Masked() {
		return nil
	}
	return tempURLKeyInvalid("invalid literal Temp URL IP range")
}
func signingMAC(key []byte, digest TempURLDigest, body string) string {
	var constructor func() hash.Hash
	switch digest {
	case TempURLDigestSHA256:
		constructor = sha256.New
	case TempURLDigestSHA512:
		constructor = sha512.New
	default:
		constructor = sha1.New
	}
	mac := hmac.New(constructor, key)
	_, _ = mac.Write([]byte(body))
	return hex.EncodeToString(mac.Sum(nil))
}
