package objects

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func staleCoreValue(value *string) string {
	if value == nil {
		return "<absent>"
	}
	return *value
}

func TestObjectStaleCoreChecksumProjection(t *testing.T) {
	for _, tc := range []struct {
		name    string
		headers http.Header
		md, sh  string
		invalid bool
	}{
		{"missing", http.Header{}, "<absent>", "<absent>", false},
		{"current", http.Header{"x-object-meta-x-sdk-md5": {"Literal"}, "X-Object-Meta-X-Sdk-Sha256": {"Unicode한글"}}, "Literal", "Unicode한글", false},
		{"legacy suffix", http.Header{"X-Object-Meta-X-Shade-Md5": {"legacy"}, "X-Object-Meta-X-Shade-Sha256": {"sha"}}, "legacy", "sha", false},
		{"empty current shadows legacy", http.Header{"X-Object-Meta-X-Sdk-Md5": {""}, "X-Object-Meta-X-Shade-Md5": {"legacy"}}, "", "<absent>", false},
		{"unrelated malformed ignored", http.Header{"Content-Length": {"bad"}, "Last-Modified": {"bad"}, "X-Object-Meta-Other": {"\xff\n"}}, "<absent>", "<absent>", false},
		{"aliases", http.Header{"X-Object-Meta-X-Sdk-Md5": {"one"}, "x-object-meta-x-sdk-md5": {"one"}}, "", "", true},
		{"multiple", http.Header{"X-Object-Meta-X-Sdk-Sha256": {"one", "two"}}, "", "", true},
		{"zero values", http.Header{"X-Object-Meta-X-Sdk-Md5": nil}, "", "", true},
		{"UTF8", http.Header{"X-Object-Meta-X-Sdk-Md5": {"\xff"}}, "", "", true},
		{"C1 control", http.Header{"X-Object-Meta-X-Sdk-Md5": {"value\u0085"}}, "", "", true},
		{"ASCII control", http.Header{"X-Object-Meta-X-Shade-Sha256": {"value\n"}}, "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			md, sh, err := objectStaleDigests(tc.headers)
			if tc.invalid {
				if !errors.Is(err, resource.ErrInvalidOption) || md != nil || sh != nil {
					t.Fatal("partial projection", md, sh, err)
				}
				return
			}
			if err != nil || staleCoreValue(md) != tc.md || staleCoreValue(sh) != tc.sh {
				t.Fatal(md, sh, err)
			}
		})
	}
}

func TestObjectStaleCoreLazyFilesAndComparison(t *testing.T) {
	md, sh := createCoreHashes("local bytes")
	missing := t.TempDir() + "/missing explicit file"
	for _, tc := range []struct {
		name             string
		code             int
		filename, md, sh string
		headers          http.Header
		stale            bool
		ioFailure        bool
	}{
		{"404 no read", 404, missing, "", "", nil, true, false},
		{"one known no read", 200, missing, md, "", http.Header{"X-Object-Meta-X-Sdk-Md5": {md}}, false, false},
		{"native204 known no read", 204, missing, "", sh, http.Header{"X-Object-Meta-X-Sdk-Sha256": {sh}}, false, false},
		{"both known one remote missing", 200, missing, md, sh, http.Header{"X-Object-Meta-X-Sdk-Sha256": {sh}}, true, false},
		{"case stays literal", 200, missing, strings.ToUpper(md), "", http.Header{"X-Object-Meta-X-Sdk-Md5": {md}}, true, false},
		{"file hashes match", 200, createCoreFile(t, "local bytes"), "", "", http.Header{"X-Object-Meta-X-Sdk-Md5": {md}, "X-Object-Meta-X-Sdk-Sha256": {sh}}, false, false},
		{"later file IO proof", 200, missing, "", "", http.Header{}, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := objectMetadataClient()
			calls := 0
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.Method != "HEAD" || r.Body != nil || r.URL.RawQuery != "" {
					t.Error("stale started an extra phase", r.Method, r.URL)
				}
				h := tc.headers.Clone()
				if h == nil {
					h = make(http.Header)
				}
				h.Set("X-Proof", "kept")
				return objectMetadataWire(r, tc.code, h, io.NopCloser(strings.NewReader("head raw"))), nil
			})
			result, err := New(c).IsObjectStale(context.Background(), "box", "key", tc.filename, WithIsObjectStaleMD5(tc.md), WithIsObjectStaleSHA256(tc.sh))
			if result == nil || result.Discovery == nil || result.Discovery.StatusCode != tc.code || calls != 1 {
				t.Fatal(result, err, calls)
			}
			if tc.ioFailure {
				proof := objectMetadataProof(t, err, tc.code, "head raw")
				if result.Stale != nil || result.RemoteMD5 != nil || result.RemoteSHA256 != nil || !errors.Is(err, os.ErrNotExist) {
					t.Fatal("later IO lost nullable decision", result, err)
				}
				result.Discovery.Body[0] = '!'
				result.Discovery.Header.Set("X-Proof", "changed")
				if string(proof.Body) != "head raw" || proof.Header.Get("X-Proof") != "kept" {
					t.Fatal("discovery aliases error")
				}
				return
			}
			if err != nil || result.Stale == nil || *result.Stale != tc.stale {
				t.Fatal(result, err)
			}
			if tc.name == "file hashes match" && (result.MD5 != md || result.SHA256 != sh) {
				t.Fatal("file comparison hashes missing", result)
			}
		})
	}
}

func TestObjectStaleCoreAcceptedFailuresAndStickyGuards(t *testing.T) {
	for _, mode := range []string{"close failure", "source restored by Close", "context cancellation", "projection"} {
		t.Run(mode, func(t *testing.T) {
			c := objectMetadataClient()
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("accepted boundary cause")
			calls := 0
			endpoint := c.Endpoint
			c.HTTPClient.Transport = objectMetadataTransport(func(r *http.Request) (*http.Response, error) {
				calls++
				code := 404
				h := http.Header{"X-Proof": {"kept"}}
				body := &objectMetadataBody{Reader: strings.NewReader("raw404")}
				switch mode {
				case "close failure":
					body.closeErr = cause
				case "source restored by Close":
					c.Endpoint = "http://changed.invalid/v1/a/"
					body.onClose = func() { c.Endpoint = endpoint }
				case "context cancellation":
					body.onClose = func() { cancel(cause) }
				case "projection":
					code = 200
					h.Set("X-Object-Meta-X-Sdk-Md5", "bad\u0085")
				}
				return objectMetadataWire(r, code, h, body), nil
			})
			result, err := New(c).IsObjectStale(ctx, "box", "key", "explicit missing file", WithIsObjectStaleMD5(strings.Repeat("a", 32)))
			code := 404
			if mode == "projection" {
				code = 200
			}
			objectMetadataProof(t, err, code, "raw404")
			if result == nil || result.Stale != nil || result.RemoteMD5 != nil || result.RemoteSHA256 != nil || calls != 1 {
				t.Fatal("dirty response committed stale", result, err, calls)
			}
			if mode == "close failure" && !errors.Is(err, cause) || mode == "context cancellation" && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) || (mode == "source restored by Close" || mode == "projection") && !errors.Is(err, resource.ErrInvalidOption) {
				t.Fatal("accepted cause lost", err)
			}
		})
	}
}
