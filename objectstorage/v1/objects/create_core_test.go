package objects

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func createCoreHashes(data string) (string, string) {
	return fmt.Sprintf("%x", md5.Sum([]byte(data))), fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
}
func createCoreFile(t *testing.T, data string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "explicit input")
	if err := os.WriteFile(name, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	return name
}

type createCoreBorrowed struct {
	reader                        io.Reader
	reads, closes, seeks, largest int
}

func (r *createCoreBorrowed) Read(data []byte) (int, error) {
	r.reads++
	if len(data) > r.largest {
		r.largest = len(data)
	}
	return r.reader.Read(data)
}
func (r *createCoreBorrowed) Close() error { r.closes++; return errors.New("caller Close forbidden") }
func (r *createCoreBorrowed) Seek(int64, int) (int64, error) {
	r.seeks++
	return 0, errors.New("caller Seek forbidden")
}

func TestObjectCreateCoreOwnedSources(t *testing.T) {
	c := objectMetadataClient()
	p, err := New(c).captureCreateObject(context.Background(), "box", "key")
	if err != nil {
		t.Fatal(err)
	}
	filename := createCoreFile(t, "stable file")
	owned, err := p.ownInput(context.Background(), CreateObjectInput{Filename: filename}, "file")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	first, second := owned.section(0, owned.size), owned.section(0, owned.size)
	data, err := io.ReadAll(first)
	if err != nil || string(data) != "stable file" {
		t.Fatal(string(data), err)
	}
	data, err = io.ReadAll(second)
	if err != nil || string(data) != "stable file" {
		t.Fatal("sections share a cursor", string(data), err)
	}
	spool := owned.path
	if err := owned.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(spool); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("spool retained", err)
	}
	reader := &createCoreBorrowed{reader: strings.NewReader(strings.Repeat("x", 70000))}
	owned, err = p.ownInput(context.Background(), CreateObjectInput{Reader: reader}, "reader")
	if err != nil || owned.size != 70000 || reader.largest > 32768 || reader.closes != 0 || reader.seeks != 0 {
		t.Fatal(owned, err, reader)
	}
	if err := owned.close(); err != nil {
		t.Fatal(err)
	}
	spoolDir := t.TempDir()
	t.Setenv("TMPDIR", spoolDir)
	fault := errors.New("borrowed read cause")
	reader = &createCoreBorrowed{reader: objectMetadataReader(func(data []byte) (int, error) { copy(data, "partial"); return 7, fault })}
	owned, err = p.ownInput(context.Background(), CreateObjectInput{Reader: reader}, "reader")
	files, readErr := os.ReadDir(spoolDir)
	if owned != nil || !errors.Is(err, fault) || readErr != nil || len(files) != 0 || reader.closes != 0 || reader.seeks != 0 {
		t.Fatal("failed input retained ownership", owned, err, files, readErr)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("input canceled")
	reader = &createCoreBorrowed{reader: objectMetadataReader(func(data []byte) (int, error) { cancel(cause); return copy(data, "x"), nil })}
	owned, err = p.ownInput(ctx, CreateObjectInput{Reader: reader}, "reader")
	files, readErr = os.ReadDir(spoolDir)
	if owned != nil || !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || readErr != nil || len(files) != 0 {
		t.Fatal(owned, err, files, readErr)
	}
}

func TestObjectCreateCoreChecksumDecisions(t *testing.T) {
	md, sh := createCoreHashes("owned contents")
	upperMD, upperSH := strings.ToUpper(md), strings.ToUpper(sh)
	for _, tc := range []struct {
		name           string
		opts           []CreateObjectOption
		head           int
		remote         http.Header
		wantMD, wantSH string
		meta           bool
		skip           bool
	}{
		{"default recomputes both", []CreateObjectOption{WithCreateObjectMD5(strings.Repeat("0", 32))}, 404, nil, md, sh, true, false},
		{"known both trusted case", []CreateObjectOption{WithCreateObjectMD5(upperMD), WithCreateObjectSHA256(upperSH)}, 200, http.Header{"X-Object-Meta-X-Sdk-Md5": {upperMD}, "X-Object-Meta-X-Sdk-Sha256": {upperSH}}, upperMD, upperSH, true, true},
		{"disabled missing skips comparison hashing", []CreateObjectOption{WithCreateObjectGenerateChecksums(false)}, 404, nil, "", "", false, false},
		{"disabled existing comparison only", []CreateObjectOption{WithCreateObjectGenerateChecksums(false)}, 200, http.Header{}, md, sh, false, false},
		{"disabled known one skips file hash", []CreateObjectOption{WithCreateObjectGenerateChecksums(false), WithCreateObjectMD5(upperMD)}, 204, http.Header{"X-Object-Meta-X-Sdk-Md5": {upperMD}}, upperMD, "", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			c.MoreHeaders = map[string]string{"X-Source": "captured"}
			calls, puts := 0, 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "PUT" && (r.Header.Get("Content-Type") != "" || r.Header.Get("X-Object-Meta-Tag") != "") {
					t.Error("upload headers leaked into discovery", r.Header)
				}
				switch r.Method {
				case "GET":
					return objectMetadataWire(r, 200, http.Header{}, io.NopCloser(strings.NewReader(`{"swift":{"max_file_size":100},"slo":{"min_segment_size":1}}`))), nil
				case "HEAD":
					return objectMetadataOK(r, tc.head, tc.remote.Clone()), nil
				case "PUT":
					puts++
					data, err := io.ReadAll(r.Body)
					if err != nil || string(data) != "owned contents" || r.Header.Get("Content-Type") != "image/test" || r.Header.Get("X-Object-Meta-Tag") != "literal" {
						t.Error("owned upload changed", string(data), r.Header, err)
					}
					if tc.meta && (r.Header.Get("X-Object-Meta-X-Sdk-Md5") != tc.wantMD || r.Header.Get("X-Object-Meta-X-Sdk-Sha256") != tc.wantSH) {
						t.Error("metadata digest changed", r.Header)
					}
					if !tc.meta && (r.Header.Get("X-Object-Meta-X-Sdk-Md5") != "caller" || r.Header.Get("X-Object-Meta-X-Sdk-Sha256") != "") {
						t.Error("comparison-only hashes became metadata", r.Header)
					}
					return objectMetadataOK(r, 201, http.Header{}), nil
				}
				return nil, errors.New("unexpected method")
			})
			opts := append([]CreateObjectOption{WithCreateObjectHeader("Content-Type", "image/test"), WithCreateObjectMetadata(map[string]string{"Tag": "literal", "x-sdk-md5": "caller"})}, tc.opts...)
			result, err := New(c).CreateObject(context.Background(), "box", "key", CreateObjectInput{Filename: createCoreFile(t, "owned contents")}, opts...)
			wantCalls := 3
			if tc.skip {
				wantCalls = 2
			}
			if err != nil || result == nil || result.MD5 != tc.wantMD || result.SHA256 != tc.wantSH || result.Skipped != tc.skip || calls != wantCalls || (puts == 0) != tc.skip || result.Capabilities == nil || result.Discovery == nil {
				t.Fatal(result, err, calls, puts)
			}
			if tc.skip && (result.Mode != "" || result.Ordinary != nil || result.Manifest != nil) {
				t.Fatal("matching metadata did not skip", result)
			}
		})
	}
	c := objectMetadataClient()
	c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
		if r.Method != "PUT" || r.Header.Get("X-Object-Meta-X-Sdk-Md5") != "literal" {
			t.Error("byte mode calculated a digest", r.Method, r.Header)
		}
		return objectMetadataOK(r, 202, http.Header{}), nil
	})
	result, err := New(c).CreateObject(context.Background(), "box", "key", CreateObjectInput{Data: []byte("bytes")}, WithCreateObjectMD5(md), WithCreateObjectMetadataValue("x-sdk-md5", "literal"))
	if err != nil || result == nil || result.MD5 != "" || result.SHA256 != "" || result.Capabilities != nil || result.Discovery != nil || result.Ordinary.Acknowledgement.StatusCode != 202 {
		t.Fatal(result, err)
	}
}

func TestObjectCreateCoreCapabilityProjection(t *testing.T) {
	for _, tc := range []struct {
		name, body        string
		code              int
		requested, want   int64
		invalid, fallback bool
	}{
		{"normal", `{"swift":{"max_file_size":10},"slo":{"min_segment_size":3}}`, 200, 8, 8, false, false},
		{"max before min", `{"swift":{"max_file_size":2},"slo":{"min_segment_size":8}}`, 200, 9, 2, false, false},
		{"explicit zero", `{"swift":{"max_file_size":10},"slo":{"min_segment_size":3}}`, 200, 0, 3, false, false},
		{"404 fallback", "missing", 404, 9, 9, false, true}, {"412 fallback", "missing", 412, 9, 9, false, true},
		{"bad first section", `{"swift":false}`, 200, 8, 0, true, false}, {"later atomic bound", `{"swift":{"max_file_size":10},"slo":{"min_segment_size":1.5}}`, 200, 8, 0, true, false},
		{"overflow", `{"swift":{"max_file_size":9223372036854775808}}`, 200, 8, 0, true, false}, {"null top", "null", 200, 8, 0, true, false}, {"invalid UTF8", "{\"x\":\"\xff\"}", 200, 8, 0, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			calls := 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "GET" || r.URL.Path != "/info" {
					t.Error("capability route", r.Method, r.URL)
				}
				return objectMetadataWire(r, tc.code, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader(tc.body))), nil
			})
			p, err := New(c).captureCreateObject(context.Background(), "box", "key")
			if err != nil {
				t.Fatal(err)
			}
			result, err := p.capabilities(context.Background(), tc.requested)
			if result == nil || result.Response.StatusCode != tc.code || result.Size != tc.want || result.UsedFallback != tc.fallback || calls != 1 {
				t.Fatal(result, err, calls)
			}
			if tc.invalid {
				objectMetadataProof(t, err, tc.code, tc.body)
				if result.MaxFileSize != 0 || result.MinSegmentSize != 0 {
					t.Fatal("partial capability commit", result)
				}
			} else if err != nil {
				t.Fatal(err)
			}
		})
	}
	for _, code := range []int{404, 412, 500} {
		c := objectMetadataClient()
		fault := errors.New("capability close")
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			return objectMetadataWire(r, code, http.Header{"X-Proof": {"kept"}}, &objectMetadataBody{Reader: strings.NewReader("raw"), closeErr: fault}), nil
		})
		p, err := New(c).captureCreateObject(context.Background(), "box", "key")
		if err != nil {
			t.Fatal(err)
		}
		result, err := p.capabilities(context.Background(), 8)
		if !errors.Is(err, fault) {
			t.Fatal("dirty rejection lost Close cause", err)
		}
		if code != 500 {
			if result == nil || result.Size != 0 || result.UsedFallback {
				t.Fatal(result, err)
			}
			objectMetadataProof(t, err, code, "raw")
		} else {
			var native gophercloud.ErrUnexpectedResponseCode
			if result != nil || !errors.As(err, &native) || native.Actual != 500 {
				t.Fatal(result, err)
			}
		}
	}
}

func TestObjectCreateCoreEmptyAndZeroBounds(t *testing.T) {
	for _, data := range []string{"", "nonempty"} {
		c := objectMetadataClient()
		puts := 0
		c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
			switch r.Method {
			case "GET":
				return objectMetadataWire(r, 200, http.Header{"X-Proof": {"kept"}}, io.NopCloser(strings.NewReader(`{"swift":{"max_file_size":0},"slo":{"min_segment_size":0}}`))), nil
			case "HEAD":
				return objectMetadataOK(r, 404, http.Header{}), nil
			case "PUT":
				puts++
				if r.Body != http.NoBody || r.ContentLength != 0 || len(r.TransferEncoding) != 0 {
					t.Error("empty file was chunked", r)
				}
				return objectMetadataOK(r, 201, http.Header{}), nil
			}
			return nil, errors.New("unexpected")
		})
		result, err := New(c).CreateObject(context.Background(), "box", "key", CreateObjectInput{Reader: bytes.NewBufferString(data)}, WithCreateObjectSegmentSize(0))
		if result == nil || result.Capabilities == nil || result.Discovery == nil || result.Capabilities.Size != 0 {
			t.Fatal(result, err)
		}
		if data == "" {
			if err != nil || puts != 1 || result.Mode != "ordinary" {
				t.Fatal(result, err, puts)
			}
		} else if !errors.Is(err, resource.ErrInvalidOption) || puts != 0 || result.Mode != "" {
			t.Fatal(result, err, puts)
		}
	}
}
