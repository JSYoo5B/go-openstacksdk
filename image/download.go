package image

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/fixedrequest"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/image/v2/images"
)

// DownloadTo fetches metadata, then streams a full image to a borrowed writer.
// Available canonical hashes are verified by default. The fixed request ID,
// actual response headers and successful byte count survive partial failures.
// It never closes, seeks or truncates output, and never replays written bytes.
func (s *Service) DownloadTo(ctx context.Context, ref resource.Ref, output io.Writer, options ...DownloadImageOption) (*DownloadImageResult, error) {
	wrap := func(err error) error { return wrapDownloadError(ctx, err) }
	var source *gophercloud.ServiceClient
	if s != nil {
		source = s.client
	}
	headers, err := validateDownloadSource(ctx, source)
	if err != nil {
		return nil, wrap(err)
	}
	if err := ref.Validate(); err != nil {
		return nil, wrap(err)
	}
	if !utf8.ValidString(ref.String()) {
		return nil, wrap(uploadInvalid("image reference must be valid UTF-8"))
	}
	if !ref.IsName() {
		if err := createImportID(ref.String()); err != nil {
			return nil, wrap(err)
		}
	}
	if downloadNilWriter(output) {
		return nil, wrap(uploadInvalid("download output writer is required"))
	}
	client := *source
	client.MoreHeaders = headers
	collection := source.ServiceURL("images")
	policy, err := parseDownloadImageOptions(append([]DownloadImageOption(nil), options...))
	if err != nil {
		return nil, wrap(err)
	}
	check := func() error {
		if source.ProviderClient != client.ProviderClient {
			return uploadInvalid("image download provider changed")
		}
		if _, err := validateDownloadSource(ctx, source); err != nil {
			return err
		}
		return rest.ValidateTarget(source, collection)
	}
	if err := check(); err != nil {
		return nil, wrap(err)
	}
	id := ref.String()
	if ref.IsName() {
		id, err = New(&client).Images.ResolveID(ctx, ref)
		if err != nil {
			return nil, wrap(err)
		}
		if err := createImportID(id); err != nil {
			return nil, wrap(err)
		}
		if !utf8.ValidString(id) {
			return nil, wrap(uploadInvalid("resolved image ID must be valid UTF-8"))
		}
	}
	if err := check(); err != nil {
		return nil, wrap(err)
	}
	endpoint := collection + "/" + url.PathEscape(id)
	metadata, err := downloadMetadata(ctx, &client, endpoint)
	if metadata == nil {
		return nil, wrap(err)
	}
	result := &DownloadImageResult{ImageID: id, Metadata: &DownloadMetadataResponse{
		Body: append(json.RawMessage(nil), metadata.Body...), Header: metadata.Header.Clone(), StatusCode: metadata.StatusCode,
	}}
	if err != nil {
		return result, wrap(err)
	}
	fields, err := createImportObject(metadata)
	if err != nil {
		return result, wrap(err)
	}
	canonicalID, err := createImportString(metadata, fields, "id")
	if err != nil {
		return result, wrap(err)
	}
	if canonicalID != id {
		return result, wrap(metadata.Fail(uploadInvalid("canonical image ID does not match requested image ID")))
	}
	result.Image, err = decodeDownloadImage(metadata)
	if err != nil {
		return result, wrap(err)
	}
	var digest hash.Hash
	var selected *DownloadChecksumResult
	if *policy.VerifyChecksum {
		digest, selected, err = prepareDownloadChecksum(metadata, fields)
		if err != nil {
			return result, wrap(err)
		}
		result.Checksum = selected
	}
	if err := check(); err != nil {
		return result, wrap(err)
	}
	fileURL := endpoint + "/file"
	if len(policy.StorePreferences) != 0 {
		fileURL += "?" + url.Values{"prefer": {strings.Join(policy.StorePreferences, ",")}}.Encode()
	}
	streamClient, err := fixedrequest.New(&client, http.MethodGet, fileURL)
	if err != nil {
		return result, wrap(err)
	}
	// Pre-body authentication/retry policy remains safe for this read-only GET.
	// A redirect is not the captured file route and must never receive its token.
	streamClient.ProviderClient.HTTPClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	wire, err := streamClient.Request(ctx, http.MethodGet, fileURL, &gophercloud.RequestOpts{KeepResponseBody: true, OkCodes: []int{http.StatusOK, http.StatusNoContent}})
	if err != nil {
		// Native Request already consumes and closes a rejected response body.
		return result, wrap(err)
	}
	result.Header, result.StatusCode = wire.Header.Clone(), wire.StatusCode
	if wire.StatusCode == http.StatusNoContent {
		result.Checksum = nil
		return result, wrap(errors.Join(ctx.Err(), wire.Body.Close()))
	}
	if wire.Header.Get("Content-Range") != "" {
		return result, wrap(errors.Join(uploadInvalid("full image response must not include Content-Range"), wire.Body.Close(), ctx.Err()))
	}
	if *policy.VerifyChecksum && selected == nil {
		if expected := wire.Header.Get("Content-MD5"); expected != "" {
			digest = md5.New()
			selected = &DownloadChecksumResult{Algorithm: "md5", Expected: expected}
		}
	}
	result.Checksum = selected
	var complete bool
	result.BytesWritten, complete, err = copyDownload(ctx, output, wire.Body, *policy.ChunkSize, digest)
	if complete && selected != nil {
		selected.Complete, selected.Actual = true, hex.EncodeToString(digest.Sum(nil))
		selected.Verified = selected.Actual == selected.Expected
		if !selected.Verified {
			err = errors.Join(err, &DownloadChecksumMismatchError{Algorithm: selected.Algorithm, Expected: selected.Expected, Actual: selected.Actual})
		}
	}
	// Complete/Verified describe the bytes, even if the response fails to close.
	err = errors.Join(err, wire.Body.Close(), ctx.Err())
	return result, wrap(err)
}

func validateDownloadSource(ctx context.Context, client *gophercloud.ServiceClient) (map[string]string, error) {
	if ctx == nil {
		return nil, uploadInvalid("context is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if client == nil || client.ProviderClient == nil {
		return nil, uploadInvalid("image service client is required")
	}
	if client.Type != "image" {
		return nil, fmt.Errorf("%w: image service client type is required", resource.ErrUnsupported)
	}
	base := client.ServiceURL()
	if err := rest.ValidateTarget(client, base); err != nil {
		return nil, err
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.RawQuery != "" || !strings.HasSuffix(parsed.Path, "/") {
		return nil, uploadInvalid("image service base must be query-free and end in a slash")
	}
	headers := make(map[string]string, len(client.MoreHeaders))
	for key, value := range client.MoreHeaders {
		if key == "" || !utf8.ValidString(value) {
			return nil, uploadInvalid("invalid image download source header")
		}
		for _, char := range []byte(key) {
			if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(char)) {
				continue
			}
			return nil, uploadInvalid("invalid image download header %q", key)
		}
		if !downloadHeaderValue(value) {
			return nil, uploadInvalid("invalid image download header value")
		}
		name := http.CanonicalHeaderKey(key)
		if old, exists := headers[name]; exists && old != value {
			return nil, uploadInvalid("conflicting image download header aliases %q", key)
		}
		switch strings.ToLower(key) {
		case "range", "if-range":
			return nil, uploadInvalid("partial image download headers are unsupported")
		case "x-auth-token", "x-service-token", "authorization", "host", "cookie", "content-length", "transfer-encoding", "connection", "trailer", "te", "upgrade", "x-openstack-glance-api-version", "x-openstack-image-size":
			return nil, uploadInvalid("download header %q is owned by the SDK", key)
		case "openstack-api-version":
			if client.Microversion == "" || value != "image "+client.Microversion {
				return nil, uploadInvalid("image version header conflicts with selected microversion")
			}
		}
		headers[name] = value
	}
	if !utf8.ValidString(client.Microversion) || !downloadHeaderValue(client.Microversion) {
		return nil, uploadInvalid("invalid image microversion")
	}
	return headers, nil
}

func downloadHeaderValue(value string) bool {
	for _, char := range []byte(value) {
		if char < 32 && char != '\t' || char == 127 {
			return false
		}
	}
	return true
}

func downloadNilWriter(output io.Writer) bool {
	if output == nil {
		return true
	}
	value := reflect.ValueOf(output)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	}
	return false
}

func downloadMetadata(ctx context.Context, source *gophercloud.ServiceClient, endpoint string) (*rest.Response, error) {
	client, err := fixedrequest.New(source, http.MethodGet, endpoint)
	if err != nil {
		return nil, err
	}
	wire, err := client.Request(ctx, http.MethodGet, endpoint, &gophercloud.RequestOpts{KeepResponseBody: true, OkCodes: []int{http.StatusOK}})
	if err != nil {
		return nil, err
	}
	response := &rest.Response{Header: wire.Header.Clone(), StatusCode: wire.StatusCode}
	response.Body, err = io.ReadAll(wire.Body)
	err = errors.Join(err, wire.Body.Close(), ctx.Err())
	if err != nil {
		return response, response.Fail(err)
	}
	return response, nil
}

func decodeDownloadImage(response *rest.Response) (*Image, error) {
	var body map[string]any
	decoder := json.NewDecoder(bytes.NewReader(response.Body))
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return nil, response.Fail(err)
	}
	var native images.GetResult
	native.Body, native.Header = body, response.Header.Clone()
	image, err := native.Extract()
	if err != nil {
		return nil, response.Fail(err)
	}
	return image, nil
}

func prepareDownloadChecksum(response *rest.Response, fields map[string]json.RawMessage) (hash.Hash, *DownloadChecksumResult, error) {
	values := make(map[string]string, 3)
	for _, key := range []string{"os_hash_algo", "os_hash_value", "checksum"} {
		body, exists := fields[key]
		if !exists || bytes.Equal(bytes.TrimSpace(body), []byte("null")) {
			continue
		}
		var value string
		if err := json.Unmarshal(body, &value); err != nil {
			return nil, nil, response.Fail(err)
		}
		values[key] = value
	}
	algorithm, expected := values["os_hash_algo"], values["os_hash_value"]
	if algorithm == "" || expected == "" {
		algorithm, expected = "md5", values["checksum"]
	}
	if expected == "" {
		return nil, nil, nil
	}
	var digest hash.Hash
	switch algorithm {
	case "md5":
		digest = md5.New()
	case "sha1":
		digest = sha1.New()
	case "sha224":
		digest = sha256.New224()
	case "sha256":
		digest = sha256.New()
	case "sha384":
		digest = sha512.New384()
	case "sha512":
		digest = sha512.New()
	case "sha512_224":
		digest = sha512.New512_224()
	case "sha512_256":
		digest = sha512.New512_256()
	default:
		return nil, nil, response.Fail(fmt.Errorf("%w: image hash algorithm %q", resource.ErrUnsupported, algorithm))
	}
	return digest, &DownloadChecksumResult{Algorithm: algorithm, Expected: expected}, nil
}

func copyDownload(ctx context.Context, output io.Writer, input io.Reader, chunkSize int, digest hash.Hash) (int64, bool, error) {
	buffer := make([]byte, chunkSize)
	var written int64
	var emptyReads int
	for {
		if err := ctx.Err(); err != nil {
			return written, false, err
		}
		n, readErr := input.Read(buffer)
		terminalReadErr := readErr
		if readErr == io.EOF {
			terminalReadErr = nil
		}
		if n < 0 || n > len(buffer) {
			return written, false, errors.Join(fmt.Errorf("invalid image response Read count %d", n), terminalReadErr, ctx.Err())
		}
		if err := ctx.Err(); err != nil {
			return written, false, errors.Join(terminalReadErr, err)
		}
		if n > 0 {
			emptyReads = 0
			count, writeErr := output.Write(buffer[:n])
			if count < 0 || count > n {
				return written, false, errors.Join(fmt.Errorf("invalid image output Write count %d", count), writeErr, terminalReadErr, ctx.Err())
			}
			written += int64(count)
			if digest != nil {
				_, _ = digest.Write(buffer[:count])
			}
			if count != n && writeErr == nil {
				writeErr = io.ErrShortWrite
			}
			if writeErr != nil {
				return written, false, errors.Join(writeErr, terminalReadErr, ctx.Err())
			}
		} else if readErr == nil {
			emptyReads++
			if emptyReads >= 100 {
				return written, false, io.ErrNoProgress
			}
		}
		if err := ctx.Err(); err != nil {
			return written, false, errors.Join(terminalReadErr, err)
		}
		if readErr != nil {
			return written, readErr == io.EOF, terminalReadErr
		}
	}
}

func wrapDownloadError(ctx context.Context, err error) error {
	if err != nil && ctx != nil && ctx.Err() != nil {
		for _, cause := range []error{ctx.Err(), context.Cause(ctx)} {
			if cause != nil && !errors.Is(err, cause) {
				err = errors.Join(err, cause)
			}
		}
	}
	return request.Wrap("DownloadTo", "image", err)
}
