package templateloader

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"
)

var schemePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*:`)

// normaliseFilePath turns a bare path into an absolute file URL. Anything with
// a URL scheme is returned unchanged.
func normaliseFilePath(p string) (string, error) {
	if schemePattern.MatchString(p) {
		return p, nil
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(abs)}).String(), nil
}

// baseURL is the URL of the directory that contains rawURL.
func baseURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	dir := path.Dir(u.Path)
	if u.Path == "" {
		dir = ""
	}
	return u.ResolveReference(&url.URL{Path: dir}).String(), nil
}

// join resolves value against base like Python urljoin for the supported schemes.
func join(base, value string) (string, error) {
	if base == "" {
		return value, nil
	}
	b, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	ref, err := url.Parse(value)
	if err != nil {
		return "", err
	}
	return b.ResolveReference(ref).String(), nil
}

// fetch reads a file or HTTP(S) URL. Any non-2xx status is an error.
func (l *Loader) fetch(ctx context.Context, rawURL string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid URL %q: %v", ErrTemplate, rawURL, err)
	}
	var reader io.ReadCloser
	switch strings.ToLower(u.Scheme) {
	case "file":
		if u.Host != "" && !strings.EqualFold(u.Host, "localhost") {
			return nil, fmt.Errorf("%w: file URL %q names a remote host", ErrTemplate, rawURL)
		}
		file, err := os.Open(filepath.FromSlash(u.Path))
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTemplate, err)
		}
		reader = file
	case "http", "https":
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrTemplate, err)
		}
		resp, err := l.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrTemplate, err)
		}
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("%w: GET %s returned HTTP %d", ErrTemplate, rawURL, resp.StatusCode)
		}
		reader = resp.Body
	default:
		return nil, fmt.Errorf("%w: unsupported URL scheme %q in %q", ErrTemplate, u.Scheme, rawURL)
	}
	defer reader.Close()
	return l.readAll(ctx, reader, rawURL)
}

func (l *Loader) readAll(ctx context.Context, reader io.Reader, name string) ([]byte, error) {
	if l.maxBytes > 0 {
		reader = io.LimitReader(reader, l.maxBytes+1)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("%w: reading %s: %v", ErrTemplate, name, err)
	}
	if l.maxBytes > 0 && int64(len(data)) > l.maxBytes {
		return nil, fmt.Errorf("%w: %s exceeds the %d byte limit", ErrTemplate, name, l.maxBytes)
	}
	return data, nil
}

// object reads through the configured ObjectReader. A missing reader behaves
// like Python's absent object_request: nothing can be fetched.
func (l *Loader) object(ctx context.Context, rawURL string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if l.objects == nil {
		return nil, nil
	}
	data, err := l.objects(ctx, rawURL)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("%w: reading object %s: %v", ErrTemplate, rawURL, err)
	}
	return bytes.Clone(data), nil
}

// fileContent mirrors Python read_url_content: UTF-8 text is kept as is and
// anything else becomes base64.encodebytes text (76 columns, trailing newline).
func fileContent(data []byte) string {
	if len(data) == 0 {
		return ""
	}
	if utf8.Valid(data) {
		return string(data)
	}
	return encodeBytes(data)
}

func encodeBytes(data []byte) string {
	encoded := base64.StdEncoding.EncodeToString(data)
	var out strings.Builder
	for len(encoded) > 76 {
		out.WriteString(encoded[:76])
		out.WriteByte('\n')
		encoded = encoded[76:]
	}
	if encoded != "" {
		out.WriteString(encoded)
		out.WriteByte('\n')
	}
	return out.String()
}
