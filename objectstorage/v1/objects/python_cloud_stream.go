package objects

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"errors"
	"io"
	"iter"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/klauspost/compress/zstd"
)

// CloudStreamOpts configures the closed cloud stream_object call. Nil ChunkSize
// selects 1024 bytes. QueryString is intentionally unused by the pinned Source.
// Chunk sizes are positive and at most 16 MiB to bound each owned chunk.
type CloudStreamOpts struct {
	ChunkSize   *int
	QueryString string
}
type CloudStreamOption func(*CloudStreamOpts) error

func cloneCloudStreamOpts(value CloudStreamOpts) CloudStreamOpts {
	if value.ChunkSize != nil {
		size := *value.ChunkSize
		value.ChunkSize = &size
	}
	return value
}
func WithCloudStreamOpts(value CloudStreamOpts) CloudStreamOption {
	value = cloneCloudStreamOpts(value)
	return func(cfg *CloudStreamOpts) error { *cfg = cloneCloudStreamOpts(value); return nil }
}
func WithCloudStreamChunkSize(value int) CloudStreamOption {
	return func(cfg *CloudStreamOpts) error { size := value; cfg.ChunkSize = &size; return nil }
}
func WithCloudStreamQueryString(value string) CloudStreamOption {
	return func(cfg *CloudStreamOpts) error { cfg.QueryString = value; return nil }
}

// CloudStreamReceipt contains actual response facts without copying or retaining
// the object payload. Receipt returns a fresh header map on every call.
type CloudStreamReceipt struct {
	StatusCode int
	Header     http.Header
}

// CloudObjectStream is a library-owned, lazy, single-use binary iterator. Chunks
// closes the response on EOF, failure, or consumer break. Check Err after a break
// to observe a cleanup failure; Close also returns the terminal error. A handle
// closed before iteration never makes a request. Concurrent iteration is not
// supported, as with a Python generator. Close may stop a pending GET; when a
// physical Read is already active Close waits for it. Use the request context
// to cancel a blocked network Read.
type CloudObjectStream interface {
	Chunks() iter.Seq2[[]byte, error]
	Err() error
	Close() error
	Receipt() *CloudStreamReceipt
}

type cloudObjectStream struct {
	api               *API
	ctx               context.Context
	container, object string
	options           []CloudStreamOption
	mu                sync.Mutex
	readMu            sync.Mutex
	started, closed   bool
	source            *objectListSource
	body              io.ReadCloser
	decoders          []io.Closer
	receipt           *CloudStreamReceipt
	terminal          error
}

// StreamCloudObject implements the fixed cloud string/container branch rather
// than Proxy.stream_object's mutable Object and arbitrary attributes. Creation
// is lazy: input checks, options and GET occur on the first Chunks iteration.
func (a *API) StreamCloudObject(ctx context.Context, container, object string, options ...CloudStreamOption) CloudObjectStream {
	return &cloudObjectStream{api: a, ctx: ctx, container: container, object: object, options: slices.Clone(options)}
}

func (s *cloudObjectStream) Receipt() *CloudStreamReceipt {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.receipt == nil {
		return nil
	}
	return &CloudStreamReceipt{StatusCode: s.receipt.StatusCode, Header: s.receipt.Header.Clone()}
}
func (s *cloudObjectStream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return cloneCloudStreamError(s.terminal)
}
func (s *cloudObjectStream) Close() error { s.finish(nil); return s.Err() }

func (s *cloudObjectStream) isClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

func (s *cloudObjectStream) record(cause error) {
	if cause == nil {
		return
	}
	cause = metadataContextError(s.ctx, cause)
	s.mu.Lock()
	defer s.mu.Unlock()
	var proof *resource.ResponseError
	if s.receipt != nil && !errors.As(cause, &proof) {
		cause = &resource.ResponseError{StatusCode: s.receipt.StatusCode, Header: s.receipt.Header.Clone(), Cause: cause}
	}
	s.terminal = errors.Join(s.terminal, request.Wrap("StreamCloudObject", "objects", cloneCloudStreamError(cause)))
}

func (s *cloudObjectStream) finish(cause error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		s.record(cause)
		return
	}
	s.closed = true
	body, decoders, source := s.body, s.decoders, s.source
	s.body = nil
	s.decoders = nil
	s.mu.Unlock()
	// Never hold state locks while closing a caller-supplied physical body.
	// Codec Read/Close is serialized independently from lazy GET adoption.
	s.readMu.Lock()
	var closes []error
	for index := len(decoders) - 1; index >= 0; index-- {
		closes = append(closes, decoders[index].Close())
	}
	if body != nil {
		closes = append(closes, body.Close())
	}
	s.readMu.Unlock()
	if source != nil {
		closes = append(closes, source.check(s.ctx))
	}
	s.record(metadataContextError(s.ctx, errors.Join(cause, errors.Join(closes...))))
}

func (s *cloudObjectStream) open() (io.Reader, int, error) {
	if s.isClosed() {
		return nil, 0, nil
	}
	p, err := s.api.captureObjectList(s.ctx, s.container)
	if err != nil {
		return nil, 0, err
	}
	s.mu.Lock()
	s.source = p
	s.mu.Unlock()
	if err := validateMetadataObject(s.object); err != nil {
		return nil, 0, metadataContextError(s.ctx, err)
	}
	cfg := CloudStreamOpts{}
	for _, option := range s.options {
		if s.isClosed() {
			return nil, 0, nil
		}
		if err := p.check(s.ctx); err != nil {
			return nil, 0, err
		}
		if option == nil {
			return nil, 0, metadataInvalid("nil cloud stream option")
		}
		next := cloneCloudStreamOpts(cfg)
		if err := errors.Join(option(&next), p.check(s.ctx)); err != nil {
			return nil, 0, metadataContextError(s.ctx, err)
		}
		if s.isClosed() {
			return nil, 0, nil
		}
		cfg = cloneCloudStreamOpts(next)
	}
	size := 1024
	if cfg.ChunkSize != nil {
		size = *cfg.ChunkSize
	}
	if size <= 0 || size > 16*1024*1024 {
		return nil, 0, metadataInvalid("cloud stream chunk size must be between 1 and 16777216")
	}
	// query_string is ignored even if it contains malformed percent escapes,
	// controls or invalid UTF-8. It must never reach the request target.
	target := p.target + "/" + url.PathEscape(s.object)
	if _, ok := p.client.MoreHeaders["Accept"]; !ok {
		p.client.MoreHeaders["Accept"] = "*/*"
	}
	if _, ok := p.client.MoreHeaders["Accept-Encoding"]; !ok {
		p.client.MoreHeaders["Accept-Encoding"] = "gzip, deflate, zstd"
	}
	policy := rawObjectHeaderPolicy(p)
	policy.Values["Accept-Encoding"] = p.client.MoreHeaders["Accept-Encoding"]
	accepted := make([]int, 300)
	for i := range accepted {
		accepted[i] = 100 + i
	}
	rejected := make([]int, 600)
	for i := range rejected {
		rejected[i] = 400 + i
	}
	if s.isClosed() {
		return nil, 0, nil
	}
	wire, err := rest.DoUnreadGuardedRejectionsHeaderPolicy(s.ctx, &p.client, p.check, http.MethodGet, target, policy, rest.RejectionPolicy{Codes: rejected, PreserveCleanRetry: true, StopFaultRetryCallbacks: true}, accepted...)
	if wire != nil {
		s.mu.Lock()
		s.receipt = &CloudStreamReceipt{StatusCode: wire.StatusCode, Header: wire.Header.Clone()}
		stopped := s.closed
		if !stopped {
			s.body = wire.Body
		}
		s.mu.Unlock()
		if stopped {
			// Close may happen inside an option or while GET is in flight. A late
			// accepted body must still be closed exactly once and retain its receipt.
			s.record(errors.Join(err, wire.Body.Close(), p.check(s.ctx)))
			return nil, size, nil
		}
	} else if native, ok := rawObjectNativeReceipt(err); ok {
		s.mu.Lock()
		s.receipt = &CloudStreamReceipt{StatusCode: native.Actual, Header: native.ResponseHeader.Clone()}
		s.mu.Unlock()
	}
	if err != nil {
		native, ok := rawObjectNativeReceipt(err)
		if ok && rawObjectCleanStatus(err) {
			// raise_from_response reads response.content before choosing NotFound.
			// Native rejection already owns/closed its physical bytes: validate its
			// encoded body without reading or closing the physical response again.
			reader, decoders, decodeErr := cloudStreamDecoder(bytes.NewReader(native.Body), native.ResponseHeader.Get("Content-Encoding"))
			if decodeErr == nil {
				_, decodeErr = io.Copy(io.Discard, reader)
			}
			for index := len(decoders) - 1; index >= 0; index-- {
				decodeErr = errors.Join(decodeErr, decoders[index].Close())
			}
			decodeErr = errors.Join(decodeErr, p.check(s.ctx))
			if decodeErr != nil {
				return nil, 0, &resource.ResponseError{StatusCode: native.Actual, Header: native.ResponseHeader.Clone(), Body: bytes.Clone(native.Body), Cause: errors.Join(err, decodeErr)}
			}
			if native.Actual == http.StatusNotFound {
				return nil, size, nil
			}
		}
		return nil, 0, err
	}
	s.readMu.Lock()
	if s.isClosed() {
		s.readMu.Unlock()
		return nil, size, nil
	}
	reader, closers, decodeErr := cloudStreamDecoder(wire.Body, wire.Header.Get("Content-Encoding"))
	s.mu.Lock()
	stopped := s.closed
	if !stopped {
		s.decoders = closers
	}
	s.mu.Unlock()
	if stopped {
		for index := len(closers) - 1; index >= 0; index-- {
			decodeErr = errors.Join(decodeErr, closers[index].Close())
		}
		s.readMu.Unlock()
		s.record(decodeErr)
		return nil, size, nil
	}
	s.readMu.Unlock()
	return reader, size, errors.Join(decodeErr, p.check(s.ctx))
}

func (s *cloudObjectStream) Chunks() iter.Seq2[[]byte, error] {
	return func(yield func([]byte, error) bool) {
		s.mu.Lock()
		if s.started || s.closed {
			s.mu.Unlock()
			return
		}
		s.started = true
		s.mu.Unlock()
		reader, size, err := s.open()
		if err != nil {
			s.finish(err)
			yield(nil, s.Err())
			return
		}
		if reader == nil {
			s.finish(nil)
			if err := s.Err(); err != nil {
				yield(nil, err)
			}
			return
		}
		defer s.finish(nil)
		for {
			if s.isClosed() {
				return
			}
			buffer := make([]byte, size)
			filled, stalls := 0, 0
			var terminal error
			eof := false
			for filled < len(buffer) {
				if s.isClosed() {
					return
				}
				if err := s.source.check(s.ctx); err != nil {
					terminal = err
					break
				}
				s.readMu.Lock()
				if s.isClosed() {
					s.readMu.Unlock()
					return
				}
				n, readErr := reader.Read(buffer[filled:])
				s.readMu.Unlock()
				if s.isClosed() {
					return
				}
				if n < 0 || n > len(buffer)-filled {
					terminal = metadataInvalid("invalid decoded stream reader count")
					break
				}
				filled += n
				guardErr := s.source.check(s.ctx)
				if readErr != nil || guardErr != nil {
					if readErr == io.EOF && guardErr == nil {
						eof = true
					} else {
						terminal = errors.Join(readErr, guardErr)
					}
					break
				}
				if n == 0 {
					stalls++
					if stalls >= 100 {
						terminal = io.ErrNoProgress
						break
					}
				} else {
					stalls = 0
				}
			}
			if terminal != nil {
				s.finish(terminal)
				yield(bytes.Clone(buffer[:filled]), s.Err())
				return
			}
			if eof {
				s.finish(nil)
				if filled > 0 || s.Err() != nil {
					yield(bytes.Clone(buffer[:filled]), s.Err())
				}
				return
			}
			if !yield(bytes.Clone(buffer[:filled]), nil) {
				return
			}
		}
	}
}

// Decode the installed CPython 3.14 Requests/urllib3 formats incrementally.
// A single unknown encoding (including optional, uninstalled brotli) remains
// opaque. In a known multi-encoding chain urllib3 uses deflate for unknown links.
func cloudStreamDecoder(body io.Reader, encoding string) (io.Reader, []io.Closer, error) {
	encoding = strings.ToLower(encoding)
	known := func(value string) bool {
		return value == "gzip" || value == "x-gzip" || value == "deflate" || value == "zstd"
	}
	parts := []string{encoding}
	if !known(encoding) {
		if !strings.Contains(encoding, ",") {
			return body, nil, nil
		}
		parts = strings.Split(encoding, ",")
		selected := false
		for _, part := range parts {
			if known(strings.TrimSpace(part)) {
				selected = true
			}
		}
		if !selected {
			return body, nil, nil
		}
		if len(parts) > 5 {
			return nil, nil, metadataInvalid("more than five content encodings")
		}
	}
	reader := body
	var closers []io.Closer
	for index := len(parts) - 1; index >= 0; index-- {
		part := strings.TrimSpace(parts[index])
		switch part {
		case "gzip", "x-gzip":
			decoded := &cloudStreamGzip{source: reader}
			closers = append(closers, decoded)
			reader = decoded
		case "zstd":
			decoded, err := zstd.NewReader(reader, zstd.WithDecoderConcurrency(1), zstd.WithDecoderLowmem(true))
			if err != nil {
				return nil, closers, err
			}
			closer := decoded.IOReadCloser()
			closers = append(closers, closer)
			reader = closer
		default:
			decoded := &cloudStreamDeflate{source: reader}
			closers = append(closers, decoded)
			reader = decoded

		}
	}
	return reader, closers, nil
}

// Private error snapshots prevent callers from changing a later Err receipt.
func cloneCloudStreamError(err error) error {
	switch value := err.(type) {
	case nil:
		return nil
	case *resource.OperationError:
		copy := *value
		copy.Cause = cloneCloudStreamError(value.Cause)
		return &copy
	case *resource.ResponseError:
		copy := *value
		copy.Body = bytes.Clone(value.Body)
		copy.Header = value.Header.Clone()
		copy.Cause = cloneCloudStreamError(value.Cause)
		return &copy
	case gophercloud.ErrUnexpectedResponseCode:
		value.Body = bytes.Clone(value.Body)
		value.ResponseHeader = value.ResponseHeader.Clone()
		value.Expected = slices.Clone(value.Expected)
		return value
	case *gophercloud.ErrUnableToReauthenticate:
		copy := *value
		copy.ErrOriginal = cloneCloudStreamError(value.ErrOriginal)
		copy.ErrReauth = cloneCloudStreamError(value.ErrReauth)
		return &copy
	case *gophercloud.ErrErrorAfterReauthentication:
		copy := *value
		copy.ErrOriginal = cloneCloudStreamError(value.ErrOriginal)
		return &copy
	case interface{ Unwrap() error }:
		snapshot := cloudStreamUnarySnapshot{message: err.Error(), original: err, cause: cloneCloudStreamError(value.Unwrap())}
		if terminal, ok := err.(interface{ TerminalSDKFailure() bool }); ok && terminal.TerminalSDKFailure() {
			return &cloudStreamTerminalSnapshot{snapshot}
		}
		return &snapshot
	case interface{ Unwrap() []error }:
		causes := value.Unwrap()
		copies := make([]error, len(causes))
		for i, cause := range causes {
			copies[i] = cloneCloudStreamError(cause)
		}
		return errors.Join(copies...)
	default:
		return err
	}
}

// Preserve private unary policy wrappers without exposing their mutable receipt.
type cloudStreamUnarySnapshot struct {
	message         string
	original, cause error
}

func (s *cloudStreamUnarySnapshot) Error() string        { return s.message }
func (s *cloudStreamUnarySnapshot) Unwrap() error        { return s.cause }
func (s *cloudStreamUnarySnapshot) Is(target error) bool { return errors.Is(s.original, target) }

type cloudStreamTerminalSnapshot struct{ cloudStreamUnarySnapshot }

func (s *cloudStreamTerminalSnapshot) TerminalSDKFailure() bool { return true }

// urllib3 first attempts a zlib stream and falls back to raw DEFLATE if that
// attempt fails before emitting data. Preserve only that undecided prefix.
type cloudStreamPrefixReader struct {
	source io.Reader
	active bool
	prefix bytes.Buffer
}

func (r *cloudStreamPrefixReader) Read(data []byte) (int, error) {
	n, err := r.source.Read(data)
	if n > 0 && n <= len(data) && r.active {
		_, _ = r.prefix.Write(data[:n])
	}
	return n, err
}

type cloudStreamDeflate struct {
	source  io.Reader
	prefix  *cloudStreamPrefixReader
	decoder io.ReadCloser
	first   bool
}

func cloudStreamDeflateFormatError(err error) bool {
	if err == zlib.ErrHeader || err == zlib.ErrDictionary || err == zlib.ErrChecksum {
		return true
	}
	_, ok := err.(flate.CorruptInputError)
	return ok
}
func (d *cloudStreamDeflate) fallback() {
	if d.decoder != nil {
		_ = d.decoder.Close()
	}
	owned := bytes.Clone(d.prefix.prefix.Bytes())
	d.prefix.active = false
	d.prefix.prefix.Reset()
	d.decoder = flate.NewReader(io.MultiReader(bytes.NewReader(owned), d.source))
	d.first = false
}
func (d *cloudStreamDeflate) Read(data []byte) (int, error) {
	if d.decoder == nil {
		d.prefix = &cloudStreamPrefixReader{source: d.source, active: true}
		d.first = true
		reader, err := zlib.NewReader(d.prefix)
		if err != nil {
			if cloudStreamDeflateFormatError(err) {
				d.fallback()
			} else if err == io.EOF || err == io.ErrUnexpectedEOF {
				return 0, io.EOF
			} else {
				return 0, err
			}
		} else {
			d.decoder = reader
		}
	}
	n, err := d.decoder.Read(data)
	if d.first && cloudStreamDeflateFormatError(err) {
		d.fallback()
		n, err = d.decoder.Read(data)
	}
	if n > 0 && (err == nil || err == io.EOF || err == io.ErrUnexpectedEOF) {
		d.first = false
		d.prefix.active = false
		d.prefix.prefix.Reset()
	}
	// Python zlib.flush accepts missing final checksum/end markers. A physical
	// ResponseError containing EOF is never collapsed into successful EOF.
	if err == io.ErrUnexpectedEOF {
		err = io.EOF
	}
	return n, err
}
func (d *cloudStreamDeflate) Close() error {
	if d.decoder == nil {
		return nil
	}
	return d.decoder.Close()
}

// Python's gzip decoder accepts cleanly ended truncated streams and trailing
// garbage after a complete member. Observe physical EOF separately so a real
// Read/Close/context/source failure can never establish that successful ending.
type cloudStreamEOFReader struct {
	source   io.Reader
	cleanEOF bool
	fault    error
}

func (r *cloudStreamEOFReader) Read(data []byte) (int, error) {
	n, err := r.source.Read(data)
	if err == io.EOF {
		r.cleanEOF = true
	} else if err != nil {
		r.fault = errors.Join(r.fault, err)
	}
	return n, err
}

type cloudStreamGzip struct {
	source              io.Reader
	observed            *cloudStreamEOFReader
	buffered            *bufio.Reader
	decoder             *gzip.Reader
	completed, finished bool
}

func (g *cloudStreamGzip) drain() error {
	_, err := io.Copy(io.Discard, g.buffered)
	if err != nil {
		return err
	}
	if g.observed.fault != nil {
		return g.observed.fault
	}
	if !g.observed.cleanEOF {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (g *cloudStreamGzip) Read(data []byte) (int, error) {
	if g.finished {
		return 0, io.EOF
	}
	if g.observed == nil {
		g.observed = &cloudStreamEOFReader{source: g.source}
		g.buffered = bufio.NewReader(g.observed)
	}
	for {
		if g.decoder == nil {
			prefix, _ := g.buffered.Peek(2)
			if g.observed.fault != nil {
				return 0, g.observed.fault
			}
			if len(prefix) == 2 && (prefix[0] != 0x1f || prefix[1] != 0x8b) {
				if g.completed {
					if err := g.drain(); err != nil {
						return 0, err
					}
					g.finished = true
					return 0, io.EOF
				}
				return 0, gzip.ErrHeader
			}
			reader, err := gzip.NewReader(g.buffered)
			if g.observed.fault != nil {
				return 0, errors.Join(err, g.observed.fault)
			}
			if err != nil {
				if g.completed && (err == gzip.ErrHeader || err == gzip.ErrChecksum || err == io.ErrUnexpectedEOF || err == io.EOF) {
					if err := g.drain(); err != nil {
						return 0, err
					}
					g.finished = true
					return 0, io.EOF
				}
				if (err == io.EOF || err == io.ErrUnexpectedEOF) && g.observed.cleanEOF {
					g.finished = true
					return 0, io.EOF
				}
				return 0, err
			}
			reader.Multistream(false)
			g.decoder = reader
		}
		n, err := g.decoder.Read(data)
		if g.observed.fault != nil {
			return n, errors.Join(err, g.observed.fault)
		}
		if err == io.EOF {
			_ = g.decoder.Close()
			g.decoder = nil
			g.completed = true
			if n > 0 {
				return n, nil
			}
			continue
		}
		if err == io.ErrUnexpectedEOF && g.observed.cleanEOF {
			g.finished = true
			return n, io.EOF
		}
		return n, err
	}
}
func (g *cloudStreamGzip) Close() error {
	g.finished = true
	if g.decoder == nil {
		return nil
	}
	return g.decoder.Close()
}
