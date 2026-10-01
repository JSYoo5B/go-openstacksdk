// Package cyborg implements transport for the SDK-owned Cyborg resource models.
package cyborg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/pagination"
	"gophercloudsdk/accelerator/v2/common"
	"gophercloudsdk/resource"
)

func Collection[T any](client *gophercloud.ServiceClient, path, single, plural string,
	id, name, status func(*T) string, metadata func(*T) *common.Metadata,
	deleteURL func(string) string) *resource.Collection[T] {
	adapter := resource.Adapter[T]{Kind: plural, ID: id, Name: name, Status: status,
		Get: func(ctx context.Context, id string) (*T, error) {
			return Fetch[T](ctx, client, "GET", client.ServiceURL(path, url.PathEscape(id)), nil, single, metadata, 200)
		},
		List: func(query url.Values) pagination.Pager {
			endpoint := client.ServiceURL(path)
			if len(query) > 0 {
				endpoint += "?" + query.Encode()
			}
			return pagination.NewPager(client, endpoint, func(result pagination.PageResult) pagination.Page {
				return page[T]{LinkedPageBase: pagination.LinkedPageBase{PageResult: result}, plural: plural, metadata: metadata}
			})
		},
		Extract: func(p pagination.Page) ([]T, error) { return p.(page[T]).extract() },
	}
	if deleteURL != nil {
		adapter.Delete = func(ctx context.Context, id string) error {
			_, err := client.Delete(ctx, deleteURL(id), &gophercloud.RequestOpts{OkCodes: []int{204}})
			return err
		}
	}
	if status != nil {
		adapter.Failed = func(value string) bool {
			return strings.EqualFold(value, "error") || strings.EqualFold(value, "bindfailed")
		}
	}
	return resource.NewCollection(adapter)
}

func Fetch[T any](ctx context.Context, client *gophercloud.ServiceClient, method, endpoint string, body any, single string, metadata func(*T) *common.Metadata, codes ...int) (*T, error) {
	return fetch[T](ctx, client, method, endpoint, body, nil, single, metadata, codes...)
}

func fetch[T any](ctx context.Context, client *gophercloud.ServiceClient, method, endpoint string, body any, headers map[string]string, single string, metadata func(*T) *common.Metadata, codes ...int) (*T, error) {
	var raw json.RawMessage
	response, err := client.Request(ctx, method, endpoint, &gophercloud.RequestOpts{JSONBody: body, JSONResponse: &raw, MoreHeaders: headers, OkCodes: codes})
	if err != nil {
		return nil, err
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if nested, ok := envelope[single]; ok {
		raw = nested
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	meta := metadata(&value)
	meta.Header, meta.StatusCode = response.Header.Clone(), response.StatusCode
	return &value, nil
}

type page[T any] struct {
	pagination.LinkedPageBase
	plural   string
	metadata func(*T) *common.Metadata
}

func (p page[T]) extract() ([]T, error) {
	if p.Err != nil {
		return nil, p.Err
	}
	encoded, err := json.Marshal(p.Body)
	if err != nil {
		return nil, err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	raw, ok := fields[p.plural]
	if !ok || len(raw) == 0 || raw[0] != '[' {
		return nil, fmt.Errorf("Cyborg response must contain %s array", p.plural)
	}
	var values []T
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, err
	}
	for i := range values {
		meta := p.metadata(&values[i])
		meta.Header = p.Header.Clone()
		meta.StatusCode = p.StatusCode
	}
	return values, nil
}

func (p page[T]) IsEmpty() (bool, error) { values, err := p.extract(); return len(values) == 0, err }

func (p page[T]) NextPageURL() (string, error) {
	encoded, err := json.Marshal(p.Body)
	if err != nil {
		return "", err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return "", err
	}
	for _, key := range []string{"links", p.plural + "_links"} {
		if raw, ok := fields[key]; ok {
			var links []common.Link
			if err := json.Unmarshal(raw, &links); err != nil {
				return "", err
			}
			for _, link := range links {
				if link.Rel == "next" {
					return p.resolve(link.Href)
				}
			}
		}
	}
	if raw, ok := fields["next"]; ok {
		var next string
		if err := json.Unmarshal(raw, &next); err != nil {
			return "", err
		}
		return p.resolve(next)
	}
	return "", nil
}

func (p page[T]) resolve(next string) (string, error) {
	if next == "" {
		return "", nil
	}
	u, err := url.Parse(next)
	if err != nil {
		return "", err
	}
	u = p.URL.ResolveReference(u)
	if u.User != nil || u.Scheme != p.URL.Scheme || !strings.EqualFold(u.Host, p.URL.Host) {
		return "", fmt.Errorf("Cyborg pagination link changes service origin")
	}
	u.Fragment = ""
	return u.String(), nil
}

// ResponseMetadata snapshots mutation responses without inventing resource data.
func ResponseMetadata(response *http.Response) common.Metadata {
	return common.Metadata{Header: response.Header.Clone(), StatusCode: response.StatusCode}
}
