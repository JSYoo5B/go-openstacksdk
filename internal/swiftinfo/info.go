// Package swiftinfo shares Swift capability projection and segment policy.
// Service packages retain their own source guards and wire observations.
package swiftinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

const DefaultSegmentSize int64 = 1073741824
const FallbackMaxFileSize int64 = 2684354561

type Sections struct {
	Body                                       map[string]json.RawMessage
	Swift, SLO, BulkDelete, StaticWeb, TempURL map[string]json.RawMessage
}

func Decode(data []byte) (Sections, error) {
	if !utf8.Valid(data) {
		return Sections{}, fmt.Errorf("Swift info must be UTF-8 JSON")
	}
	data = bytes.TrimSpace(data)
	if len(data) == 0 || data[0] != '{' {
		return Sections{}, fmt.Errorf("Swift info must be a JSON object")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return Sections{}, err
	}
	value := Sections{Body: fields}
	for _, section := range []struct {
		name   string
		target *map[string]json.RawMessage
	}{
		{"swift", &value.Swift}, {"slo", &value.SLO},
		{"bulk_delete", &value.BulkDelete}, {"staticweb", &value.StaticWeb},
		{"tempurl", &value.TempURL},
	} {
		raw, present := fields[section.name]
		if !present || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			continue
		}
		if err := json.Unmarshal(raw, section.target); err != nil {
			return Sections{}, fmt.Errorf("Swift info section %q must be an object: %w", section.name, err)
		}
	}
	return value, nil
}

func Bound(section map[string]json.RawMessage, name string) (int64, error) {
	raw, present := section[name]
	if !present || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return 0, nil
	}
	value, err := strconv.ParseInt(string(bytes.TrimSpace(raw)), 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("Swift info bound %q must be a nonnegative int64", name)
	}
	return value, nil
}

func Select(requested, maximum, minimum int64) int64 {
	if requested > maximum {
		return maximum
	}
	if requested < minimum {
		return minimum
	}
	return requested
}

var versionPath = regexp.MustCompile(`/v[0-9]+\.?[0-9]*(/.*)?`)

func Target(endpoint string) (string, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.User != nil || parsed.Opaque != "" || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("%w: invalid Swift info catalog endpoint", resource.ErrInvalidOption)
	}
	path := parsed.EscapedPath()
	if versionPath.MatchString(path) {
		path = versionPath.ReplaceAllString(path, "/info")
	} else {
		path = strings.TrimRight(path, "/") + "/info"
	}
	decoded, err := url.PathUnescape(path)
	if err != nil {
		return "", fmt.Errorf("%w: invalid Swift info escaped path", resource.ErrInvalidOption)
	}
	parsed.Path, parsed.RawPath = decoded, path
	return parsed.String(), nil
}
