package image

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha3"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"net/http"
	"reflect"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
)

// ErrImageRecordDownloadHashUnsupported lets a primary hash factory select the
// Source fallback to metadata or response MD5, or an unverified download.
// Other factory errors stop the operation after the accepted binary response.
var ErrImageRecordDownloadHashUnsupported = errors.New("image download hash algorithm unsupported")

// ErrImageRecordDownloadHashLengthRequired identifies SHAKE's missing digest
// length. Source selects SHAKE successfully but cannot call hexdigest without
// a length after consuming an output or memory response. Stream-only mode does
// not request a digest and does not fail for this reason.
var ErrImageRecordDownloadHashLengthRequired = errors.New("image download hash output length required")

// ImageRecordDownloadChecksumMismatchError retains the raw expected value.
// Source compares the lower-case hex digest to that value without converting
// it: truthy JSON numbers, booleans and containers therefore cannot match.
type ImageRecordDownloadChecksumMismatchError struct {
	Algorithm string
	Expected  json.RawMessage
	Actual    string
}

func (value *ImageRecordDownloadChecksumMismatchError) Error() string {
	expected, err := cloudfilter.PythonString(value.Expected)
	if err != nil {
		expected = string(value.Expected)
	}
	return fmt.Sprintf("image %s checksum mismatch: expected %s, got %s", value.Algorithm, expected, value.Actual)
}

func (*ImageRecordDownloadChecksumMismatchError) Unwrap() error { return ErrChecksumMismatch }

// The caller invokes selection only AFTER the binary GET. Hash fields have
// Source type=None and remain raw JSON; the algorithm alone uses Python str.
// A custom factory replaces the one primary hashlib.new attempt. Fallback MD5
// is the distinct SDK-owned hashlib.md5 constructor and never replays it.
func prepareImageRecordDownloadChecksum(record *ImageRecord, header http.Header, config ImageRecordDownloadOpts) (hash.Hash, *ImageRecordDownloadChecksum, error) {
	if config.VerifyChecksum != nil && !*config.VerifyChecksum {
		return nil, nil, nil
	}
	if record == nil || record.Resource == nil {
		return nil, nil, uploadInvalid("fetched image Record is required for checksum selection")
	}
	fields := record.Resource.Body
	expected, algorithm := bytes.Clone(fields["hash_value"]), bytes.Clone(fields["hash_algo"])
	expectedTruthy, err := cloudfilter.PythonTruthy(expected)
	if err != nil {
		return nil, nil, err
	}
	if expectedTruthy {
		algorithmTruthy, err := cloudfilter.PythonTruthy(algorithm)
		if err != nil {
			return nil, nil, err
		}
		if algorithmTruthy {
			name, err := cloudfilter.PythonString(algorithm)
			if err != nil {
				return nil, nil, err
			}
			factory := config.HashFactory
			if factory == nil {
				factory = imageRecordDownloadDefaultHash
			}
			digest, err := factory(name)
			if err == nil {
				if imageRecordDownloadNilHash(digest) {
					return nil, nil, uploadInvalid("image download hash factory returned a nil hash")
				}
				return digest, &ImageRecordDownloadChecksum{Algorithm: name, Expected: expected}, nil
			}
			if !errors.Is(err, ErrImageRecordDownloadHashUnsupported) {
				return nil, nil, err
			}
		}
	}
	expected = bytes.Clone(fields["checksum"])
	truthy, err := cloudfilter.PythonTruthy(expected)
	if err != nil {
		return nil, nil, err
	}
	if !truthy {
		checksum := imageRecordDownloadHeaderValue(header, "Content-MD5")
		if checksum == "" {
			return nil, nil, nil
		}
		expected, err = json.Marshal(checksum)
		if err != nil {
			return nil, nil, err
		}
	}
	return md5.New(), &ImageRecordDownloadChecksum{Algorithm: "md5", Expected: expected}, nil
}

// Complete means EOF with successful writes, independently of a later HTTP or
// file Close failure. SHAKE reaches that boundary but has no implicit hex size.
func finishImageRecordDownloadChecksum(digest hash.Hash, selected *ImageRecordDownloadChecksum) error {
	if selected == nil {
		return nil
	}
	if imageRecordDownloadNilHash(digest) {
		return uploadInvalid("selected image checksum requires a hash")
	}
	selected.Complete, selected.Actual, selected.Verified = true, "", false
	if _, needsLength := digest.(*imageRecordDownloadShakeHash); needsLength {
		return fmt.Errorf("%w: %s hexdigest needs a length", ErrImageRecordDownloadHashLengthRequired, selected.Algorithm)
	}
	selected.Actual = hex.EncodeToString(digest.Sum(nil))
	truthy, err := cloudfilter.PythonTruthy(selected.Expected)
	if err != nil {
		return err
	}
	if !truthy {
		return nil
	}
	raw := bytes.TrimSpace(selected.Expected)
	var expected string
	// json.Unmarshal accepts null into a string. Source requires an actual
	// string equal to hexdigest; all other truthy JSON kinds mismatch.
	if len(raw) > 0 && raw[0] == '"' {
		if err := json.Unmarshal(raw, &expected); err != nil {
			return err
		}
		selected.Verified = expected == selected.Actual
	}
	if selected.Verified {
		return nil
	}
	return &ImageRecordDownloadChecksumMismatchError{Algorithm: selected.Algorithm, Expected: bytes.Clone(selected.Expected), Actual: selected.Actual}
}

// The raw selected MD5 value and its string projection are compatibility-only
// channels. Actual binary headers remain independently owned and untouched.
func imageRecordDownloadCompatibility(selected *ImageRecordDownloadChecksum, response *ImageRecordDownloadResponse) error {
	if response == nil {
		return uploadInvalid("image download response is required")
	}
	response.CompatibilityHeader = response.Header.Clone()
	response.ContentMD5 = nil
	if selected == nil || selected.Algorithm != "md5" {
		return nil
	}
	truthy, err := cloudfilter.PythonTruthy(selected.Expected)
	if err != nil || !truthy {
		return err
	}
	expected, err := cloudfilter.PythonString(selected.Expected)
	if err != nil {
		return err
	}
	response.ContentMD5 = bytes.Clone(selected.Expected)
	if response.CompatibilityHeader == nil {
		response.CompatibilityHeader = make(http.Header)
	}
	// A custom Transport may return uncanonicalized aliases. requests treats
	// these case-insensitively; replace them only in the compatibility clone.
	for key := range response.CompatibilityHeader {
		if strings.EqualFold(key, "Content-MD5") {
			delete(response.CompatibilityHeader, key)
		}
	}
	response.CompatibilityHeader.Set("Content-MD5", expected)
	return nil
}

// This finite SDK registry handles Python's fixed-size SHA/MD5/SHA3/BLAKE2
// names and common case/hyphen aliases. It does not promise all algorithms
// registered by a deployment's OpenSSL. Optional factories extend that domain.
func imageRecordDownloadDefaultHash(name string) (hash.Hash, error) {
	switch strings.ToLower(name) {
	case "md5":
		return md5.New(), nil
	case "sha1", "sha-1":
		return sha1.New(), nil
	case "sha224", "sha-224":
		return sha256.New224(), nil
	case "sha256", "sha-256":
		return sha256.New(), nil
	case "sha384", "sha-384":
		return sha512.New384(), nil
	case "sha512", "sha-512":
		return sha512.New(), nil
	case "sha512_224", "sha512-224":
		return sha512.New512_224(), nil
	case "sha512_256", "sha512-256":
		return sha512.New512_256(), nil
	case "sha3_224", "sha3-224":
		return sha3.New224(), nil
	case "sha3_256", "sha3-256":
		return sha3.New256(), nil
	case "sha3_384", "sha3-384":
		return sha3.New384(), nil
	case "sha3_512", "sha3-512":
		return sha3.New512(), nil
	case "blake2b", "blake2b512", "blake2b-512":
		return blake2b.New512(nil)
	case "blake2s", "blake2s256", "blake2s-256":
		return blake2s.New256(nil)
	case "shake_128", "shake128", "shake-128":
		return &imageRecordDownloadShakeHash{SHAKE: sha3.NewSHAKE128()}, nil
	case "shake_256", "shake256", "shake-256":
		return &imageRecordDownloadShakeHash{SHAKE: sha3.NewSHAKE256()}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrImageRecordDownloadHashUnsupported, name)
	}
}

func imageRecordDownloadNilHash(digest hash.Hash) bool {
	if digest == nil {
		return true
	}
	value := reflect.ValueOf(digest)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	}
	return false
}

// hash.Hash permits the common copy engine to absorb SHAKE chunks. Source
// hexdigest lacks its required length, so finish detects this private wrapper
// before Sum and reports that error instead of inventing an output length.
type imageRecordDownloadShakeHash struct{ *sha3.SHAKE }

func (*imageRecordDownloadShakeHash) Size() int                { return 0 }
func (*imageRecordDownloadShakeHash) Sum(prefix []byte) []byte { return prefix }

// HTTP combines repeated header values in encounter order. A Go map lacks the
// order of differently cased aliases, so those rare custom-Transport keys use
// deterministic sorted order while each value slice keeps its original order.
func imageRecordDownloadHeaderValue(header http.Header, name string) string {
	keys := make([]string, 0)
	for key := range header {
		if strings.EqualFold(key, name) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	values := make([]string, 0)
	for _, key := range keys {
		values = append(values, header[key]...)
	}
	return strings.Join(values, ", ")
}
