package objects

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/ianaindex"
	textunicode "golang.org/x/text/encoding/unicode"
	"golang.org/x/text/encoding/unicode/utf32"
	"golang.org/x/text/transform"
)

// CloudGetOpts configures the fixed cloud get_object string-name call. Nil
// Outfile/Writer and OmitContent=false select the default response.text branch.
// Empty Outfile or OmitContent=true selects Source's falsey non-None outfile
// branch: headers but no data, write, or text read. QueryString and Stream are
// intentionally ignored. ChunkSize only applies to a nonempty path or writer.
type CloudGetOpts struct {
	ChunkSize   *int
	QueryString string
	Stream      bool
	Outfile     *string
	Writer      io.Writer
	OmitContent bool
}
type CloudGetOption func(*CloudGetOpts) error

func cloneCloudGetOpts(value CloudGetOpts) CloudGetOpts {
	if value.ChunkSize != nil {
		size := *value.ChunkSize
		value.ChunkSize = &size
	}
	if value.Outfile != nil {
		path := *value.Outfile
		value.Outfile = &path
	}
	return value
}
func WithCloudGetOpts(value CloudGetOpts) CloudGetOption {
	value = cloneCloudGetOpts(value)
	return func(cfg *CloudGetOpts) error { *cfg = cloneCloudGetOpts(value); return nil }
}
func WithCloudGetChunkSize(value int) CloudGetOption {
	return func(cfg *CloudGetOpts) error { size := value; cfg.ChunkSize = &size; return nil }
}
func WithCloudGetQueryString(value string) CloudGetOption {
	return func(cfg *CloudGetOpts) error { cfg.QueryString = value; return nil }
}
func WithCloudGetStream(value bool) CloudGetOption {
	return func(cfg *CloudGetOpts) error { cfg.Stream = value; return nil }
}
func WithCloudGetOutfile(value string) CloudGetOption {
	return func(cfg *CloudGetOpts) error {
		path := value
		cfg.Outfile = &path
		cfg.Writer = nil
		cfg.OmitContent = false
		return nil
	}
}
func WithCloudGetWriter(value io.Writer) CloudGetOption {
	return func(cfg *CloudGetOpts) error {
		if cloudGetNilWriter(value) {
			return metadataInvalid("borrowed get_object writer required")
		}
		cfg.Writer = value
		cfg.Outfile = nil
		cfg.OmitContent = false
		return nil
	}
}
func WithoutCloudGetContent() CloudGetOption {
	return func(cfg *CloudGetOpts) error { cfg.OmitContent = true; cfg.Writer = nil; cfg.Outfile = nil; return nil }
}
func WithoutCloudGetOutfile() CloudGetOption {
	return func(cfg *CloudGetOpts) error {
		cfg.OmitContent = false
		cfg.Writer = nil
		cfg.Outfile = nil
		return nil
	}
}

// CloudGetResult keeps the lowercase Source tuple headers and optional text
// separate from actual HTTP receipt. Missing represents the clean Source None
// result; its Receipt still retains actual404. Writer/path results have nil
// Body, and BytesWritten counts only accepted destination writes.
type CloudGetResult struct {
	Headers      map[string]string
	Body         *string
	Receipt      *CloudStreamReceipt
	BytesWritten int64
	Missing      bool
}

func cloudGetNilWriter(value io.Writer) bool {
	if value == nil {
		return true
	}
	item := reflect.ValueOf(value)
	switch item.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return item.IsNil()
	}
	return false
}
func cloudGetOutput(cfg CloudGetOpts) (memory, discard bool, errorValue error) {
	if cfg.Outfile != nil && cfg.Writer != nil || cfg.OmitContent && (cfg.Outfile != nil || cfg.Writer != nil) {
		return false, false, metadataInvalid("get_object output controls conflict")
	}
	if cfg.Writer != nil && cloudGetNilWriter(cfg.Writer) {
		return false, false, metadataInvalid("borrowed get_object writer is nil")
	}
	if cfg.OmitContent || cfg.Outfile != nil && *cfg.Outfile == "" {
		return false, true, nil
	}
	return cfg.Outfile == nil && cfg.Writer == nil, false, nil
}
func cloudGetLowerHeaders(header http.Header) map[string]string {
	result := make(map[string]string, len(header))
	for name, values := range header {
		result[strings.ToLower(name)] = strings.Join(values, ", ")
	}
	return result
}
func cloudGetValidateTranslatedHeaders(header http.Header) error {
	// Resource._translate_response(has_body=False) still materializes all
	// declared Object Header descriptors before the path is opened.
	for _, decl := range objectListHeaderFields {
		values, present := header[http.CanonicalHeaderKey(decl.wire)]
		if !present {
			continue
		}
		raw, _ := json.Marshal(strings.Join(values, ", "))
		if _, err := objectListConvert(raw, decl.kind); err != nil {
			return err
		}
	}
	return nil
}

// GetCloudObjectText owns the closed cloud get_object workflow. It reuses the
// guarded Source stream opener/codecs while buffering only the text branch;
// writers are borrowed, path handles and the physical HTTP body are SDK-owned.
// Requests' statistical apparent_encoding remains explicitly unsupported for
// unresolved unmarked non-UTF8 content, rather than silently guessing a codec.
func (a *API) GetCloudObjectText(ctx context.Context, container, object string, options ...CloudGetOption) (result *CloudGetResult, terminal error) {
	cfg := CloudGetOpts{}
	memory, discard := true, false
	owned := slices.Clone(options)
	var stream *cloudObjectStream
	configure := func(target *CloudStreamOpts) error {
		for _, option := range owned {
			if err := stream.source.check(ctx); err != nil {
				return err
			}
			if option == nil {
				return metadataInvalid("nil cloud get_object option")
			}
			next := cloneCloudGetOpts(cfg)
			if err := errors.Join(option(&next), stream.source.check(ctx)); err != nil {
				return metadataContextError(ctx, err)
			}
			cfg = cloneCloudGetOpts(next)
		}
		var err error
		memory, discard, err = cloudGetOutput(cfg)
		if err != nil {
			return err
		}
		// Source only uses resp_chunk_size inside its truthy outfile branch.
		if !memory && !discard {
			target.ChunkSize = cloneCloudGetOpts(cfg).ChunkSize
		}
		return stream.source.check(ctx)
	}
	stream = &cloudObjectStream{api: a, ctx: ctx, container: container, object: object, options: []CloudStreamOption{configure}}
	var pathHandle *os.File
	var proofBytes []byte
	attach := func(cause error, payload []byte) error {
		if cause == nil {
			return nil
		}
		cause = metadataContextError(ctx, cause)
		var response *resource.ResponseError
		if result != nil && result.Receipt != nil && !errors.As(cause, &response) {
			cause = &resource.ResponseError{StatusCode: result.Receipt.StatusCode, Header: result.Receipt.Header.Clone(), Body: bytes.Clone(payload), Cause: cause}
		}
		return cause
	}
	defer func() {
		// Preserve each phase before adding cleanup faults: no cleanup error can
		// replace a writer/read/decoder cause or the already accepted byte count.
		terminal = attach(terminal, proofBytes)
		var pathClose error
		if pathHandle != nil {
			pathClose = pathHandle.Close()
		}
		closeErr := stream.Close()
		terminal = request.Wrap("GetCloudObjectText", "objects", metadataContextError(ctx, errors.Join(terminal, attach(pathClose, nil), closeErr)))
	}()
	reader, size, err := stream.open()
	receipt := stream.Receipt()
	if receipt != nil {
		result = &CloudGetResult{Receipt: receipt, Headers: cloudGetLowerHeaders(receipt.Header)}
	}
	if err != nil {
		return result, err
	}
	if reader == nil {
		if result != nil && receipt.StatusCode == http.StatusNotFound {
			result.Headers = nil
			result.Missing = true
		}
		return result, nil
	}
	if !memory {
		if err := cloudGetValidateTranslatedHeaders(receipt.Header); err != nil {
			return result, err
		}
	}
	if err := stream.source.check(ctx); err != nil {
		return result, err
	}
	if discard {
		return result, nil
	}
	var destination io.Writer
	if !memory {
		if cfg.Outfile != nil {
			if err := stream.source.check(ctx); err != nil {
				return result, err
			}
			pathHandle, err = os.OpenFile(*cfg.Outfile, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o666)
			if err != nil {
				return result, err
			}
			if err := stream.source.check(ctx); err != nil {
				return result, err
			}
			destination = pathHandle
		} else {
			destination = cfg.Writer
		}
	}
	var content bytes.Buffer
	buffer := make([]byte, size)
	for {
		if err := stream.source.check(ctx); err != nil {
			return result, err
		}
		n, readErr := reader.Read(buffer)
		if n < 0 || n > len(buffer) {
			return result, metadataInvalid("invalid get_object decoded read count")
		}
		chunk := buffer[:n]
		proofBytes = bytes.Clone(chunk)
		guardErr := stream.source.check(ctx)
		if readErr != nil && readErr != io.EOF || guardErr != nil {
			if memory && n > 0 {
				_, _ = content.Write(chunk)
				proofBytes = bytes.Clone(content.Bytes())
			}
			return result, errors.Join(readErr, guardErr)
		}
		if n > 0 {
			if memory {
				_, _ = content.Write(chunk)
			} else {
				if err := stream.source.check(ctx); err != nil {
					return result, err
				}
				written, writeErr := destination.Write(chunk)
				if written < 0 || written > len(chunk) {
					writeErr = errors.Join(writeErr, metadataInvalid("invalid get_object writer count"))
					written = 0
				}
				result.BytesWritten += int64(written)
				if written != len(chunk) && writeErr == nil {
					writeErr = io.ErrShortWrite
				}
				if err := errors.Join(writeErr, stream.source.check(ctx)); err != nil {
					return result, err
				}
			}
		}
		if readErr == io.EOF {
			break
		}
	}
	if err := stream.Close(); err != nil {
		return result, err
	}
	if memory {
		proofBytes = bytes.Clone(content.Bytes())
		if err := cloudGetValidateTranslatedHeaders(receipt.Header); err != nil {
			return result, err
		}
		text, err := cloudGetResponseText(content.Bytes(), receipt.Header)
		if err != nil {
			return result, err
		}
		if err := stream.source.check(ctx); err != nil {
			return result, err
		}
		result.Body = &text
	} else if flush, ok := destination.(interface{ Flush() error }); ok {
		if err := stream.source.check(ctx); err != nil {
			return result, err
		}
		if err := errors.Join(flush.Flush(), stream.source.check(ctx)); err != nil {
			return result, err
		}
	}
	return result, stream.source.check(ctx)
}

// Match Requests' permissive header splitting, case-sensitive media substring
// checks and last charset wins; a strict MIME parser changes this behavior.
func cloudGetHeaderEncoding(header http.Header) (string, bool) {
	value := header.Get("Content-Type")
	if value == "" {
		return "", false
	}
	tokens := strings.Split(value, ";")
	media := strings.TrimSpace(tokens[0])
	encodingName := ""
	present := false
	for _, param := range tokens[1:] {
		param = strings.TrimSpace(param)
		index := strings.IndexByte(param, '=')
		if index < 0 {
			continue
		}
		key := strings.Trim(strings.TrimSpace(param[:index]), "\"' ")
		if strings.EqualFold(key, "charset") {
			encodingName = strings.Trim(param[index+1:], "\"' ")
			present = true
		}
	}
	if present {
		return strings.Trim(encodingName, "'\""), true
	}
	if strings.Contains(media, "text") {
		return "iso-8859-1", true
	}
	if strings.Contains(media, "application/json") {
		return "utf-8", true
	}
	return "", false
}
func cloudGetResponseText(payload []byte, header http.Header) (string, error) {
	if len(payload) == 0 {
		return "", nil
	}
	name, present := cloudGetHeaderEncoding(header)
	if !present {
		switch {
		case bytes.HasPrefix(payload, []byte{0xef, 0xbb, 0xbf}):
			name = "utf-8-sig"
		case bytes.HasPrefix(payload, []byte{0xff, 0xfe, 0, 0}), bytes.HasPrefix(payload, []byte{0, 0, 0xfe, 0xff}):
			name = "utf-32"
		case bytes.HasPrefix(payload, []byte{0xff, 0xfe}), bytes.HasPrefix(payload, []byte{0xfe, 0xff}):
			name = "utf-16"
		case utf8.Valid(payload):
			name = "utf-8"
		default:
			return "", fmt.Errorf("%w: Requests statistical apparent_encoding for unmarked non-UTF8 content is not implemented", resource.ErrUnsupported)
		}
	}
	normalized := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(name), "_", "-"))
	var codec encoding.Encoding
	switch normalized {
	case "utf-8-sig", "utf8-sig":
		payload = bytes.TrimPrefix(payload, []byte{0xef, 0xbb, 0xbf})
		codec = textunicode.UTF8
	case "", "utf-8", "utf8", "u8", "utf", "cp65001":
		codec = textunicode.UTF8
	case "latin-1", "latin1", "l1", "iso-8859-1", "iso8859-1", "cp819", "iso-ir-100":
		codec = charmap.ISO8859_1
	case "ascii", "us-ascii", "us", "646", "cp367", "ansi-x3.4-1968":
		var out strings.Builder
		for _, b := range payload {
			if b > 127 {
				out.WriteRune('\ufffd')
			} else {
				out.WriteByte(b)
			}
		}
		return out.String(), nil
	case "utf-16", "utf16":
		codec = textunicode.UTF16(textunicode.LittleEndian, textunicode.UseBOM)
	case "utf-16-le", "utf-16le", "utf16le":
		codec = textunicode.UTF16(textunicode.LittleEndian, textunicode.IgnoreBOM)
	case "utf-16-be", "utf-16be", "utf16be":
		codec = textunicode.UTF16(textunicode.BigEndian, textunicode.IgnoreBOM)
	case "utf-32", "utf32":
		codec = utf32.UTF32(utf32.LittleEndian, utf32.UseBOM)
	case "utf-32-le", "utf-32le", "utf32le":
		codec = utf32.UTF32(utf32.LittleEndian, utf32.IgnoreBOM)
	case "utf-32-be", "utf-32be", "utf32be":
		codec = utf32.UTF32(utf32.BigEndian, utf32.IgnoreBOM)
	default:
		if strings.HasPrefix(normalized, "cp125") {
			normalized = "windows-" + strings.TrimPrefix(normalized, "cp")
		} else if strings.HasPrefix(normalized, "cp") {
			if candidate, err := ianaindex.IANA.Encoding("IBM" + strings.TrimPrefix(normalized, "cp")); err == nil && candidate != nil {
				codec = candidate
			}
		}
		var err error
		if codec == nil {
			codec, err = ianaindex.IANA.Encoding(normalized)
		}
		if err != nil {
			codec = textunicode.UTF8
		} else if codec == nil {
			return "", fmt.Errorf("%w: registered text charset %q is not supported", resource.ErrUnsupported, name)
		}
	}
	decoded, _, err := transform.Bytes(codec.NewDecoder(), payload)
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}
