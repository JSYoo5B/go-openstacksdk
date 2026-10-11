package containers

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

func listRecordPageKey(value *url.URL) string {
	copy := *value
	copy.Scheme, copy.Host = strings.ToLower(copy.Scheme), strings.ToLower(copy.Host)
	copy.RawQuery = copy.Query().Encode()
	return copy.String()
}
func listRecordSplitLinks(header string) ([]string, error) {
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
		if !quote {
			if char == '<' {
				if angle {
					return nil, metadataInvalid("malformed HTTP Link")
				}
				angle = true
			}
			if char == '>' {
				if !angle {
					return nil, metadataInvalid("malformed HTTP Link")
				}
				angle = false
			}
			if char == ',' && !angle {
				parts = append(parts, strings.TrimSpace(header[start:index]))
				start = index + 1
			}
		}
	}
	if angle || quote || escape {
		return nil, metadataInvalid("malformed HTTP Link")
	}
	parts = append(parts, strings.TrimSpace(header[start:]))
	return parts, nil
}
func listRecordHTTPNext(headers http.Header) (string, error) {
	var next string
	for _, header := range headers.Values("Link") {
		parts, err := listRecordSplitLinks(header)
		if err != nil {
			return "", err
		}
		for _, part := range parts {
			if part == "" {
				continue
			}
			if part[0] != '<' {
				return "", metadataInvalid("HTTP Link requires angle-bracket URL")
			}
			end := strings.IndexByte(part, '>')
			if end < 1 {
				return "", metadataInvalid("malformed HTTP Link URL")
			}
			_, params, err := mime.ParseMediaType("application/octet-stream" + strings.TrimSpace(part[end+1:]))
			if err != nil {
				return "", err
			}
			if params["rel"] != "next" {
				continue
			}
			target := part[1:end]
			if next != "" && next != target {
				return "", metadataInvalid("conflicting HTTP next links")
			}
			next = target
		}
	}
	return next, nil
}
func listRecordBodyNext(page *rest.Response) (string, error) {
	raw := bytes.TrimSpace(page.Body)
	if len(raw) == 0 || raw[0] != '{' {
		return "", nil
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return "", err
	}
	if links, exists := fields["X-Account-Container-Count"]; exists {
		var rows []json.RawMessage
		// Source treats a dictionary as a generator of one-key dictionaries, which
		// do not provide rel/href. Only list entries can advertise a next link.
		if bytes.TrimSpace(links)[0] != '{' {
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
				if rel == "next" {
					if href, exists := values["href"]; exists {
						truthy, err := cloudfilter.PythonTruthy(href)
						if err != nil {
							return "", err
						}
						if truthy {
							var text string
							if err := json.Unmarshal(href, &text); err != nil {
								return "", err
							}
							return text, nil
						}
						break
					}
				}
			}
		}
	}
	if raw, exists := fields["next"]; exists {
		truthy, err := cloudfilter.PythonTruthy(raw)
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
func listRecordLockNext(collection, current, next *url.URL, initial url.Values) error {
	if next.User != nil || next.Opaque != "" || next.Fragment != "" || !strings.EqualFold(next.Scheme, collection.Scheme) || !strings.EqualFold(next.Host, collection.Host) || next.EscapedPath() != collection.EscapedPath() {
		return metadataInvalid("container next link changes account origin or path")
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
			return metadataInvalid("container next link changes filter %q", key)
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
func listRecordNext(page *rest.Response, collection, current *url.URL, initial url.Values, limitTruthy bool, last json.RawMessage, total int) (*url.URL, error) {
	link, err := listRecordBodyNext(page)
	if err != nil {
		return nil, err
	}
	if link == "" {
		link, err = listRecordHTTPNext(page.Header)
		if err != nil {
			return nil, err
		}
	}
	if link != "" {
		parsed, err := url.Parse(link)
		if err != nil {
			return nil, err
		}
		next := current.ResolveReference(parsed)
		if err := listRecordLockNext(collection, current, next, initial); err != nil {
			return nil, err
		}
		return next, nil
	}
	countNext := false
	if values := page.Header.Values("X-Account-Container-Count"); len(values) > 0 {
		if len(values) != 1 {
			return nil, metadataInvalid("ambiguous account count header")
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
	marker, err := listRecordMarker(last)
	if err != nil {
		return nil, err
	}
	next := *collection
	query := current.Query()
	delete(query, "marker")
	if len(marker) > 0 {
		query["marker"] = marker
	}
	next.RawQuery = query.Encode()
	return &next, nil
}

func listRecordMarkerKey(query url.Values) string {
	return url.Values{"marker": append([]string(nil), query["marker"]...)}.Encode()
}
