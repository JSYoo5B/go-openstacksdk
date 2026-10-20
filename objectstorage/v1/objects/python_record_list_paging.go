package objects

import (
	"bytes"
	"encoding/json"
	"math/big"
	"mime"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/cloudfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/jsonfilter"
	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
)

// objectListPaginationKey is Object.pagination_key. It names both the body
// links member of a singleton page and the Swift count response header.
const objectListPaginationKey = "X-Container-Object-Count"

func objectListPageKey(value *url.URL) string {
	copy := *value
	copy.Scheme, copy.Host = strings.ToLower(copy.Scheme), strings.ToLower(copy.Host)
	copy.RawQuery = copy.Query().Encode()
	return copy.String()
}
func objectListMarkerKey(query url.Values) string {
	return url.Values{"marker": append([]string(nil), query["marker"]...)}.Encode()
}
func objectListSplitLinks(header string) ([]string, error) {
	var parts []string
	start := 0
	angle, quote, escape := false, false, false
	for index, char := range header {
		if escape {
			escape = false
			continue
		}
		if quote && char == '\\' {
			escape = true
			continue
		}
		if char == '"' {
			quote = !quote
			continue
		}
		if quote {
			continue
		}
		switch {
		case char == '<' && angle, char == '>' && !angle:
			return nil, metadataInvalid("malformed HTTP Link")
		case char == '<':
			angle = true
		case char == '>':
			angle = false
		case char == ',' && !angle:
			parts = append(parts, strings.TrimSpace(header[start:index]))
			start = index + 1
		}
	}
	if angle || quote || escape {
		return nil, metadataInvalid("malformed HTTP Link")
	}
	return append(parts, strings.TrimSpace(header[start:])), nil
}
func objectListHTTPNext(headers http.Header) (string, error) {
	var next string
	for _, header := range headers.Values("Link") {
		parts, err := objectListSplitLinks(header)
		if err != nil {
			return "", err
		}
		for _, part := range parts {
			if part == "" {
				continue
			}
			end := strings.IndexByte(part, '>')
			if part[0] != '<' || end < 1 {
				return "", metadataInvalid("malformed HTTP Link URL")
			}
			_, params, err := mime.ParseMediaType("application/octet-stream" + strings.TrimSpace(part[end+1:]))
			if err != nil {
				return "", err
			}
			if params["rel"] != "next" {
				continue
			}
			if target := part[1:end]; next != "" && next != target {
				return "", metadataInvalid("conflicting HTTP next links")
			} else {
				next = target
			}
		}
	}
	return next, nil
}

// objectListBodyNext reads a singleton dictionary page: the pagination key
// list of rel/href links first, then a top-level next member.
func objectListBodyNext(page *rest.Response) (string, error) {
	raw := bytes.TrimSpace(page.Body)
	if len(raw) == 0 || raw[0] != '{' {
		return "", nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return "", err
	}
	if links, exists := fields[objectListPaginationKey]; exists && bytes.TrimSpace(links)[0] != '{' {
		// A dictionary iterates as one-key dictionaries without rel/href.
		var rows []json.RawMessage
		if err := json.Unmarshal(links, &rows); err != nil {
			return "", err
		}
		if rows == nil {
			return "", metadataInvalid("Source pagination links are not iterable")
		}
		for _, row := range rows {
			members, err := cloudfilter.ObjectMembers(row)
			if err != nil {
				return "", err
			}
			values := make(map[string]json.RawMessage)
			for _, member := range members {
				values[member.Key] = member.Value
			}
			var rel string
			_ = json.Unmarshal(values["rel"], &rel)
			href, hasHref := values["href"]
			if rel != "next" || !hasHref {
				continue
			}
			truthy, err := cloudfilter.PythonJSONTruthy(href)
			if err != nil {
				return "", err
			}
			if !truthy {
				break
			}
			var text string
			if err := json.Unmarshal(href, &text); err != nil {
				return "", err
			}
			return text, nil
		}
	}
	if raw, exists := fields["next"]; exists {
		truthy, err := cloudfilter.PythonJSONTruthy(raw)
		if err != nil || !truthy {
			return "", err
		}
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return "", err
		}
		return text, nil
	}
	return "", nil
}

// objectListLockNext keeps an advertised link on the same container listing
// and the initial non-paging filters; omitted query members are inherited.
func objectListLockNext(collection, current, next *url.URL, initial url.Values) error {
	if next.User != nil || next.Opaque != "" || next.Fragment != "" || !strings.EqualFold(next.Scheme, collection.Scheme) || !strings.EqualFold(next.Host, collection.Host) || next.EscapedPath() != collection.EscapedPath() {
		return metadataInvalid("object next link changes container origin or path")
	}
	query, err := url.ParseQuery(next.RawQuery)
	if err != nil {
		return err
	}
	previous := current.Query()
	for key, values := range query {
		if key == "marker" {
			continue
		}
		if key == "limit" {
			if old, exists := previous[key]; exists && !reflect.DeepEqual(old, values) {
				return metadataInvalid("next link changes limit")
			}
			continue
		}
		if old, exists := initial[key]; !exists || !reflect.DeepEqual(old, values) {
			return metadataInvalid("object next link changes filter %q", key)
		}
	}
	for key, values := range previous {
		if !query.Has(key) {
			query[key] = append([]string(nil), values...)
		}
	}
	next.RawQuery = query.Encode()
	return nil
}

// objectListNext follows Source _get_next_link for Object: body links, RFC
// Link, then the X-Container-Object-Count total against all consumed rows, and
// finally the truthy-limit fallback. Count and limit continuation reuse the
// container URI with marker set to the last row id (its name).
func objectListNext(page *rest.Response, collection, current *url.URL, initial url.Values, limitTruthy bool, last json.RawMessage, total int) (*url.URL, error) {
	link, err := objectListBodyNext(page)
	if err == nil && link == "" {
		link, err = objectListHTTPNext(page.Header)
	}
	if err != nil {
		return nil, err
	}
	if link != "" {
		parsed, err := url.Parse(link)
		if err != nil {
			return nil, err
		}
		next := current.ResolveReference(parsed)
		if err := objectListLockNext(collection, current, next, initial); err != nil {
			return nil, err
		}
		return next, nil
	}
	countNext := false
	if values := page.Header.Values(objectListPaginationKey); len(values) > 0 {
		if len(values) != 1 {
			return nil, metadataInvalid("ambiguous container object count header")
		}
		if values[0] != "" {
			count, err := jsonfilter.PythonIntegerString(values[0])
			if err != nil {
				return nil, err
			}
			countNext = count.Cmp(big.NewInt(int64(total))) > 0
		}
	}
	if !countNext && !limitTruthy {
		return nil, nil
	}
	marker, err := cloudfilter.RequestQueryValues(last)
	if err != nil {
		return nil, err
	}
	next := *collection
	query := current.Query()
	delete(query, "marker")
	if !limitTruthy {
		// Source pops limit after each page and restores only a truthy one.
		delete(query, "limit")
	}
	if len(marker) > 0 {
		query["marker"] = marker
	}
	next.RawQuery = query.Encode()
	return &next, nil
}
