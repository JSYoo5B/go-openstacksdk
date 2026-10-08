package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	"github.com/JSYoo5B/go-openstacksdk/internal/rest"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	volumes "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type getVolumesEntry struct {
	volume *resource.RawResource
	origin *rest.Response
	index  int
	raw    json.RawMessage
}

// materialize finishes the list before any attachment or server field is read.
func (p *preparedGetVolumes) materialize(ctx context.Context, result *GetVolumesResult) ([]getVolumesEntry, error) {
	entries := make([]getVolumesEntry, 0)
	err := p.readVolumes(ctx, result, func(entry getVolumesEntry) (bool, error) {
		entries = append(entries, entry)
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// readVolumes owns transport and raw rows. SDK bindings choose whether to
// consume the complete list or stop at the shared identity iterator's yield.
func (p *preparedGetVolumes) readVolumes(ctx context.Context, result *GetVolumesResult, consume func(getVolumesEntry) (bool, error)) error {
	current := p.initialURL
	seen := make(map[string]bool)
	for {
		if err := p.guard(ctx); err != nil {
			return err
		}
		key, err := p.pageKey(current)
		if err != nil {
			return err
		}
		if seen[key] {
			return &resource.PaginationCycleError{URL: current.String()}
		}
		seen[key] = true
		response, err := p.exchange(ctx, current)
		if response != nil {
			result.Pages = append(result.Pages, &GetVolumesPage{Body: bytes.Clone(response.Body), Header: response.Header.Clone(), StatusCode: response.StatusCode})
		}
		if err != nil {
			return err
		}
		fields, rows, err := getVolumesPageRows(response)
		if err != nil {
			return response.Fail(err)
		}
		for index, row := range rows {
			if err := p.guard(ctx); err != nil {
				return err
			}
			var volume resource.RawResource
			if err := json.Unmarshal(row, &volume); err != nil {
				return response.Fail(fmt.Errorf("volumes[%d]: %w", index, err))
			}
			volume.Header = response.Header.Clone()
			volume.StatusCode = response.StatusCode
			more, err := consume(getVolumesEntry{volume: &volume, origin: response, index: index, raw: bytes.Clone(row)})
			if err != nil {
				return err
			}
			if !more {
				return p.guard(ctx)
			}
		}
		if err := p.guard(ctx); err != nil {
			return err
		}
		// Both pinned Resource.list and the native volume pager terminate on an
		// empty resource array, without consuming an unused continuation.
		if len(rows) == 0 {
			return nil
		}
		next, err := getVolumesNativeNext(response, current, fields)
		if err != nil {
			return response.Fail(err)
		}
		if next == "" {
			return p.guard(ctx)
		}
		continuation, err := p.continuation(current, next)
		if err != nil {
			return response.Fail(err)
		}
		key, err = p.pageKey(continuation)
		if err != nil {
			return response.Fail(err)
		}
		if seen[key] {
			return response.Fail(&resource.PaginationCycleError{URL: continuation.String()})
		}
		current = continuation
	}
}

func getVolumesPageRows(response *rest.Response) (map[string]json.RawMessage, []json.RawMessage, error) {
	fields, err := attachmentObject(response.Body)
	if err != nil {
		return nil, nil, err
	}
	raw, present := fields["volumes"]
	value := bytes.TrimSpace(raw)
	if !present || len(value) == 0 || value[0] != '[' {
		return nil, nil, attachInvalid("volume list response must contain a nonnull canonical volumes array")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, nil, err
	}
	return fields, rows, nil
}

func getVolumesNativeNext(response *rest.Response, current *url.URL, fields map[string]json.RawMessage) (string, error) {
	// Native link extraction remains authoritative. Supply only the exact
	// canonical envelope so Go struct case matching cannot promote an alias.
	links := make(map[string]json.RawMessage)
	if value, present := fields["volumes_links"]; present {
		links["volumes_links"] = value
	}
	page := volumes.VolumePage{LinkedPageBase: pagination.LinkedPageBase{PageResult: pagination.PageResult{
		Result: gophercloud.Result{Body: links, Header: response.Header.Clone(), StatusCode: response.StatusCode},
		URL:    *current,
	}}}
	return page.NextPageURL()
}

func (p *preparedGetVolumes) continuation(current *url.URL, href string) (*url.URL, error) {
	if !attachText(href) || strings.Contains(href, "#") {
		return nil, attachInvalid("volume pagination URL must be valid UTF-8 without controls or fragment")
	}
	reference, err := url.Parse(href)
	if err != nil {
		return nil, fmt.Errorf("%w: volume pagination URL: %w", resource.ErrInvalidOption, err)
	}
	if reference.User != nil || reference.Opaque != "" {
		return nil, attachInvalid("volume pagination URL changes authority or URL form")
	}
	next := current.ResolveReference(reference)
	if _, err := p.pageKey(next); err != nil {
		return nil, err
	}
	return next, nil
}

// The key canonicalizes advertised query order solely for cycle detection;
// physical requests retain the server's advertised query spelling and values.
func (p *preparedGetVolumes) pageKey(target *url.URL) (string, error) {
	if target == nil || !attachText(target.String()) || target.User != nil || target.Opaque != "" || target.Fragment != "" || !strings.EqualFold(target.Scheme, p.initialURL.Scheme) || !strings.EqualFold(target.Host, p.initialURL.Host) || target.EscapedPath() != p.initialURL.EscapedPath() {
		return "", attachInvalid("volume pagination changes collection origin or escaped path")
	}
	if err := rest.ValidateTarget(&p.cinder.client, target.String()); err != nil {
		return "", err
	}
	query, err := url.ParseQuery(target.RawQuery)
	if err != nil {
		return "", fmt.Errorf("%w: volume pagination query: %w", resource.ErrInvalidOption, err)
	}
	for key, values := range query {
		owned := append([]string(nil), values...)
		slices.Sort(owned)
		query[key] = owned
	}
	return strings.ToLower(target.Scheme) + "://" + strings.ToLower(target.Host) + target.EscapedPath() + "?" + query.Encode(), nil
}
