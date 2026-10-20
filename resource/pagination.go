package resource

import (
	"context"
	"net/url"
	"reflect"
	"strings"

	"github.com/gophercloud/gophercloud/v2/pagination"
)

// eachPage retains the native page type for its extractor. Native page bases
// embed PageResult.URL, which lets us reject a repeated link before fetching it.
func eachPage(ctx context.Context, pager pagination.Pager, handler func(context.Context, pagination.Page) (bool, error)) error {
	visited := make(map[string]struct{})
	return pager.EachPage(ctx, func(ctx context.Context, page pagination.Page) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if current := pageURL(page); current != "" {
			visited[paginationURLKey(current)] = struct{}{}
		}
		ok, err := handler(ctx, page)
		if err != nil || !ok {
			return ok, err
		}
		next, err := page.NextPageURL()
		if err != nil || next == "" {
			return err == nil, err
		}
		key := paginationURLKey(next)
		if _, repeated := visited[key]; repeated {
			return false, &PaginationCycleError{URL: next}
		}
		// Also guard custom pages without PageResult.URL. For native pages the
		// current response URL above includes the first page in this set.
		visited[key] = struct{}{}
		return true, nil
	})
}

func pageURL(page pagination.Page) string {
	v := reflect.ValueOf(page)
	for v.IsValid() && v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	if !v.IsValid() || v.Kind() != reflect.Struct {
		return ""
	}
	field := v.FieldByName("URL")
	if !field.IsValid() || !field.CanInterface() {
		return ""
	}
	u, ok := field.Interface().(url.URL)
	if !ok {
		return ""
	}
	return u.String()
}

func paginationURLKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	if query, err := url.ParseQuery(u.RawQuery); err == nil {
		u.RawQuery = query.Encode()
	}
	return u.String()
}
