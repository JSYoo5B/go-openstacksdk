package image

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/JSYoo5B/go-openstacksdk/image/v2/imageimport"
	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

// createUploaded is the complete non-Task branch of modern create_image. Its
// initial POST and import-method check intentionally precede upload cleanup.
func (p *preparedImageRecordCreateWorkflow) createUploaded(root, properties map[string]json.RawMessage, result *ImageRecordCreateResult) (err error) {
	policy := p.policy.Import
	all, err := imageCreateTruthy(policy.AllStores)
	if err != nil {
		return err
	}
	if all && len(policy.Stores) > 0 {
		return uploadInvalid("all_stores is mutually exclusive with stores")
	}
	must, err := imageCreateTruthy(policy.AllStoresMustSucceed)
	if err != nil {
		return err
	}
	useImport, err := imageCreateTruthy(p.policy.UseImport)
	if err != nil {
		return err
	}
	useImport = useImport || len(policy.Stores) > 0 || all || must
	methodTruthy, err := imageCreateTruthy(policy.Method)
	if err != nil {
		return err
	}
	if useImport && !methodTruthy {
		policy.Method = json.RawMessage(`"glance-direct"`)
	}

	data := p.input.Data.openReader()
	ownedFile := p.input.Filename != "" && !p.input.Data.truthy()
	if ownedFile {
		if err := p.check(p.ctx); err != nil {
			return err
		}
		file, openErr := os.Open(p.input.Filename)
		if file != nil {
			defer func() {
				if result.Record != nil {
					result.Record.data = nil
				}
				err = errors.Join(err, file.Close(), p.check(p.ctx))
			}()
		}
		if err := errors.Join(openErr, p.check(p.ctx)); err != nil {
			return err
		}
		info, statErr := file.Stat()
		if statErr == nil && info.IsDir() {
			statErr = &os.PathError{Op: "open", Path: p.input.Filename, Err: syscall.EISDIR}
		}
		if err := errors.Join(statErr, p.check(p.ctx)); err != nil {
			return err
		}
		data = file
	}
	flat, err := p.imageCreateUploadedFlatMetadata(root, properties)
	if err != nil {
		return err
	}
	prepared, err := prepareImageRecordCreate(p.preparedImageRecord, flat)
	if err != nil {
		return err
	}
	record, response, err := createPreparedImageRecord(prepared)
	result.Created, result.Record = imageUploadResponse(response), record
	if err != nil {
		return err
	}
	// Source assigns image.data unconditionally, including None and empty data.
	record.data = data
	if useImport {
		method, methodErr := decodeImageRecordString(policy.Method, "image import method")
		if methodErr != nil || !slices.Contains(record.ImportMethods, method) {
			return errors.Join(uploadInvalid("importing image was requested but the cloud does not support the image import method"), methodErr, p.check(p.ctx))
		}
	}
	// Only failures within the binary/import/checksum block invoke deletion.
	// File-open, metadata creation and unsupported methods leave no cleanup ACK.
	err = func() error {
		identity, identityErr := imageCreateUploadedIdentity(record)
		if identityErr != nil {
			return identityErr
		}
		if !useImport {
			size, sizeErr := p.imageCreateUploadedSize(data, ownedFile)
			if sizeErr != nil {
				return sizeErr
			}
			response, uploadErr := imageDataOnceSizeHeader(p.ctx, p.client, p.check,
				imageRecordEndpoint(p.preparedImageRecord, identity)+"/file", data, size, imageRecordCodes())
			result.Uploaded = imageUploadResponse(response)
			if uploadErr != nil {
				return uploadErr
			}
		} else {
			method, _ := decodeImageRecordString(policy.Method, "image import method")
			if method == "glance-direct" {
				size, sizeErr := p.imageCreateUploadedSize(data, ownedFile)
				if sizeErr != nil {
					return sizeErr
				}
				response, stageErr := imageDataOnceSizeHeader(p.ctx, p.client, p.check,
					imageRecordEndpoint(p.preparedImageRecord, identity)+"/stage", data, size, imageRecordCodes())
				result.Staged = imageUploadResponse(response)
				if stageErr != nil {
					return stageErr
				}
				if err := translateImageRecordHeaders(p.preparedImageRecord, record, response); err != nil {
					return err
				}
			}
			if policy.Stores != nil {
				policy.AllStores, policy.AllStoresMustSucceed = nil, nil
			}
			if method != "web-download" {
				policy.URI = nil
			}
			if method != "glance-download" {
				policy.RemoteRegion, policy.RemoteImageID, policy.RemoteServiceInterface = nil, nil, nil
			}
			ack, importErr := p.imageCreateUploadedImport(record, policy)
			result.Imported = ack
			if importErr != nil {
				return importErr
			}
		}
		validate, validateErr := imageCreateTruthy(p.policy.ValidateChecksum)
		if validateErr != nil {
			return validateErr
		}
		md5, sha256 := flat[imageCreateMD5Key], flat[imageCreateSHA256Key]
		md5Truthy, hashErr := imageCreateTruthy(md5)
		if hashErr != nil {
			return hashErr
		}
		shaTruthy, hashErr := imageCreateTruthy(sha256)
		if hashErr != nil {
			return hashErr
		}
		if validate && (md5Truthy || shaTruthy) {
			check := &ImageRecordCreateChecksum{MD5: bytes.Clone(md5), SHA256: bytes.Clone(sha256)}
			result.Checksum = check
			fetched, metadata, fetchErr := fetchPreparedImageRecord(p.preparedImageRecord, record, identity)
			result.ChecksumFetched = imageUploadResponse(metadata)
			if fetched != nil {
				record, result.Record = fetched, fetched
			}
			if fetchErr != nil {
				return fetchErr
			}
			check.Actual = bytes.Clone(record.bodyState.current["checksum"])
			actualTruthy, truthErr := imageCreateTruthy(check.Actual)
			if truthErr != nil {
				return truthErr
			}
			if actualTruthy {
				check.Compared = true
				for _, expected := range []json.RawMessage{md5, sha256} {
					if expected == nil {
						expected = json.RawMessage(`null`)
					}
					matched, equalErr := jsonfilter.EqualPythonJSON(check.Actual, expected)
					if equalErr != nil {
						return equalErr
					}
					check.Matched = check.Matched || matched
				}
				if !check.Matched {
					return uploadInvalid("image checksum verification failed")
				}
			}
		}
		return p.check(p.ctx)
	}()
	if err != nil {
		cleanup, cleanupErr := p.imageCreateUploadedCleanup(record)
		result.Cleanup = cleanup
		return errors.Join(err, cleanupErr, p.check(p.ctx))
	}
	if useImport {
		result.Outcome = "imported"
	} else {
		result.Outcome = "uploaded"
	}
	return p.check(p.ctx)
}

func (p *preparedImageRecordCreateWorkflow) imageCreateUploadedFlatMetadata(root, properties map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	flat := copyTaskRawMap(root)
	for key, raw := range properties {
		converted, err := convertImageRecordProperty(key, raw)
		if err != nil {
			return nil, err
		}
		flat[key] = converted
	}
	for key, value := range p.policy.Meta {
		raw, ok := value.(json.RawMessage)
		if !ok {
			return nil, uploadInvalid("image meta value was not captured")
		}
		flat[key] = bytes.Clone(raw)
	}
	flat["name"], _ = json.Marshal(p.input.Name)
	return flat, p.check(p.ctx)
}

func imageCreateUploadedIdentity(record *ImageRecord) (string, error) {
	if record == nil || record.bodyState == nil {
		return "", uploadInvalid("created image has no private body")
	}
	identity, err := decodeImageRecordString(record.bodyState.current["id"], "created image identity")
	if err == nil {
		err = validateImageRecordIdentity(identity)
	}
	return identity, err
}

// Source size accepts Python integers (including bool), retains signed arbitrary
// precision and infers only file-like values. Bytes/Text have no Source seek API.
func (p *preparedImageRecordCreateWorkflow) imageCreateUploadedSize(data io.Reader, ownedFile bool) (*string, error) {
	raw := bytes.TrimSpace(p.policy.Size)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		if !ownedFile && p.input.Data.kind != imageRecordCreateDataReader {
			return nil, p.check(p.ctx)
		}
		length, err := inferImageRecordStageSize(p.preparedImageRecord, data)
		if err != nil || length == nil {
			return nil, err
		}
		text := strconv.FormatInt(*length, 10)
		return &text, p.check(p.ctx)
	}
	var text string
	if bytes.Equal(raw, []byte("true")) {
		text = "True"
	} else if bytes.Equal(raw, []byte("false")) {
		text = "False"
	} else {
		if strings.ContainsAny(string(raw), ".eE") {
			return nil, uploadInvalid("image size must be an integer")
		}
		integer, ok := new(big.Int).SetString(string(raw), 10)
		if !ok || !json.Valid(raw) {
			return nil, uploadInvalid("image size must be an integer")
		}
		text = integer.String()
	}
	return &text, p.check(p.ctx)
}

func (p *preparedImageRecordCreateWorkflow) imageCreateUploadedImport(record *ImageRecord, policy ImageRecordImportOpts) (*imageimport.ImportResult, error) {
	// Proxy._get_resource(existing)._update() resets the plain method collector
	// before checking formats, even when no Body attribute was supplied.
	if err := projectImageRecordMutationSeed(record, p.location); err != nil {
		return nil, err
	}
	if err := validateImageRecordImportStores(policy); err != nil {
		return nil, err
	}
	for _, key := range []string{"container_format", "disk_format"} {
		present, err := cloudfilter.PythonTruthy(record.Resource.Body[key])
		if err != nil {
			return nil, err
		}
		if !present {
			return nil, uploadInvalid("both container_format and disk_format are required for importing an image")
		}
	}
	body, headers, err := compileImageRecordImport(policy)
	if err != nil {
		return nil, err
	}
	identity, err := imageCreateUploadedIdentity(record)
	if err != nil {
		return nil, err
	}
	headerPolicy := rest.RequestHeaderPolicy{Absent: []string{"X-Image-Meta-Store"}}
	if policy.Store != nil {
		headerPolicy.Values = map[string]string{"X-Image-Meta-Store": headers["X-Image-Meta-Store"]}
		headerPolicy.Absent = nil
	}
	// Header values are request-owned and do not contaminate the later fetch or
	// cleanup. The REST helper supplies actual native 400..599 rejection evidence.
	client := *p.client
	client.MoreHeaders = copyImageRecordHeaders(p.client.MoreHeaders)
	// Reuse the standalone owned import policy: configured legacy store headers
	// cannot silently add a second selector to this invocation.
	for key := range client.MoreHeaders {
		if strings.EqualFold(key, "X-Image-Meta-Store") {
			delete(client.MoreHeaders, key)
		}
	}
	for key, value := range headers {
		client.MoreHeaders[key] = value
	}
	response, err := rest.DoJSONGuardedRejectionsHeaderPolicy(p.ctx, &client, p.check, http.MethodPost,
		imageRecordEndpoint(p.preparedImageRecord, identity)+"/import", body, headerPolicy,
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	if guardErr := p.check(p.ctx); guardErr != nil {
		if response != nil {
			guardErr = response.Fail(guardErr)
		}
		err = errors.Join(err, guardErr)
	}
	if response == nil {
		return nil, err
	}
	return &imageimport.ImportResult{ImageID: identity, Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}, err
}

func (p *preparedImageRecordCreateWorkflow) imageCreateUploadedCleanup(record *ImageRecord) (*ImageRecordDeleteResult, error) {
	identity, err := imageCreateUploadedIdentity(record)
	if err != nil {
		return nil, errors.Join(err, p.check(p.ctx))
	}
	// Source calls delete_image(image.id), making an ID-only resource rather than
	// revalidating unrelated metadata of the image whose upload failed.
	seed, _, err := imageRecordMutationSeed(identity, nil)
	if err != nil {
		return nil, errors.Join(err, p.check(p.ctx))
	}
	if err := projectImageRecordMutationSeed(seed, p.location); err != nil {
		return nil, err
	}
	response, err := rest.DoJSONGuardedRejections(p.ctx, p.client, p.check, http.MethodDelete,
		imageRecordEndpoint(p.preparedImageRecord, identity), nil, nil,
		rest.RejectionPolicy{Codes: imageRecordTagRejectionCodes(), PreserveCleanRetry: true}, imageRecordCodes()...)
	if guardErr := p.check(p.ctx); guardErr != nil {
		if response != nil {
			guardErr = response.Fail(guardErr)
		}
		err = errors.Join(err, guardErr)
	}
	if response == nil {
		if native, clean := err.(gophercloud.ErrUnexpectedResponseCode); clean && native.Actual == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}
	result := &ImageRecordDeleteResult{Acknowledgement: &DeleteImageResult{ImageID: identity,
		Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode}}
	if err == nil {
		seed.ImportMethods = imageRecordHTTPImportMethods(response.Header)
		result.Record = seed
	}
	return result, err
}
