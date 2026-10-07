package blockstorage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/JSYoo5B/gophercloudsdk/internal/cloudfilter"
	"github.com/JSYoo5B/gophercloudsdk/internal/rest"
	"github.com/gophercloud/gophercloud/v2"
)

// volumeTypesNext consumes the source body's link policy, a usable HTTP Link
// repair, then the native singular-key compatibility fallback. It merges the
// next query into the current nonpagination query, as Resource.list does.
// The caller must stop on an empty canonical resource array before invoking
// this helper, then pass its result through reader.continuation and pageKey
// against the frozen initial collection URL before issuing another exchange.
func volumeTypesNext(response *rest.Response, current *url.URL, fields map[string]json.RawMessage) (*url.URL, error) {
	if response == nil || current == nil {
		return nil, attachInvalid("volume type pagination requires its response and current URL")
	}
	var href string
	var err error
	if links, present := fields["links"]; present {
		href, err = volumeTypesSourceLinks(links)
	} else if links, present := fields["volume_types_links"]; present {
		href, err = volumeTypesSourceLinks(links)
	}
	if err != nil {
		return nil, err
	}
	if href == "" {
		href, err = volumeTypesOptionalHref(fields["next"])
		if err != nil {
			return nil, err
		}
	}
	if href == "" {
		// Requests stores one entry per relation, replacing an earlier entry.
		// The existing quoted Link parser also accepts case-insensitive rel
		// tokens and multi-relation values; this usable HTTP path is a repair
		// of the pinned source's response.links[next][uri] lookup.
		for _, header := range response.Header.Values("Link") {
			if strings.TrimSpace(header) == "" {
				continue
			}
			links, parseErr := rest.HeaderNextLinks(header)
			if parseErr != nil {
				return nil, parseErr
			}
			if len(links) != 0 {
				href = links[len(links)-1]
			}
		}
	}
	if href == "" {
		// Native v2.15.0 uses volume_type_links, whereas the Python resource
		// derives volume_types_links. Keep this lower-priority extension and
		// the native last-next behavior without extracting typed resources.
		if raw, present := fields["volume_type_links"]; present {
			if !utf8.Valid(raw) {
				return nil, attachInvalid("native volume type pagination links must be valid UTF-8")
			}
			var links []gophercloud.Link
			if err := json.Unmarshal(raw, &links); err != nil {
				return nil, fmt.Errorf("native volume type pagination links: %w", err)
			}
			href, err = gophercloud.ExtractNextURL(links)
			if err != nil {
				return nil, err
			}
		}
	}
	if href == "" {
		return nil, nil
	}
	// Check the original spelling before URL.String can escape invalid UTF-8
	// or discard an empty fragment marker. Origin/path checks remain the
	// guarded reader's responsibility after the query merge.
	if !attachText(href) || strings.Contains(href, "#") {
		return nil, attachInvalid("volume type pagination URL must be valid UTF-8 without controls or fragment")
	}
	reference, err := url.Parse(href)
	if err != nil {
		return nil, attachInvalid("volume type pagination URL: %v", err)
	}
	if reference.User != nil || reference.Opaque != "" {
		return nil, attachInvalid("volume type pagination URL changes authority or URL form")
	}
	query, err := url.ParseQuery(current.RawQuery)
	if err != nil {
		return nil, attachInvalid("current volume type pagination query: %v", err)
	}
	delete(query, "marker")
	delete(query, "limit")
	incoming, err := url.ParseQuery(reference.RawQuery)
	if err != nil {
		return nil, attachInvalid("next volume type pagination query: %v", err)
	}
	for key, values := range incoming {
		var retained []string
		for _, value := range values {
			// urllib.parse.parse_qs drops blank values by default. A wholly
			// blank advertised key therefore does not replace a current key.
			if value != "" {
				retained = append(retained, value)
			}
		}
		if len(retained) != 0 {
			query[key] = retained
		}
	}
	next := current.ResolveReference(reference)
	next.RawQuery = query.Encode()
	next.ForceQuery = false
	return next, nil
}

func volumeTypesSourceLinks(raw json.RawMessage) (string, error) {
	value := bytes.TrimSpace(raw)
	if !utf8.Valid(raw) || !json.Valid(value) {
		return "", attachInvalid("volume type pagination links must be valid JSON and UTF-8")
	}
	switch value[0] {
	case '{':
		// Python iterates a dictionary as one-key dictionaries. No such row
		// can simultaneously have rel==next and a present href.
		return "", nil
	case '"':
		var text string
		if err := json.Unmarshal(value, &text); err != nil {
			return "", err
		}
		if text == "" {
			return "", nil
		}
		return "", attachInvalid("volume type pagination links contains a nonobject row")
	case '[':
		var rows []json.RawMessage
		if err := json.Unmarshal(value, &rows); err != nil {
			return "", err
		}
		for _, rawRow := range rows {
			row, err := attachmentObject(rawRow)
			if err != nil {
				return "", fmt.Errorf("volume type pagination link row: %w", err)
			}
			relRaw := bytes.TrimSpace(row["rel"])
			if len(relRaw) == 0 || relRaw[0] != '"' {
				continue
			}
			var rel string
			if err := json.Unmarshal(relRaw, &rel); err != nil {
				return "", err
			}
			if rel == "next" {
				if href, present := row["href"]; present {
					// First matching row is consumed even if its href is falsey.
					return volumeTypesOptionalHref(href)
				}
			}
		}
		return "", nil
	default:
		return "", attachInvalid("volume type pagination links must be iterable rows")
	}
}

func volumeTypesOptionalHref(raw json.RawMessage) (string, error) {
	truthy, err := cloudfilter.PythonTruthy(raw)
	if err != nil {
		return "", err
	}
	if !truthy {
		return "", nil
	}
	var href string
	if err := json.Unmarshal(raw, &href); err != nil {
		return "", attachInvalid("a truthy volume type pagination href must be a string")
	}
	return href, nil
}
