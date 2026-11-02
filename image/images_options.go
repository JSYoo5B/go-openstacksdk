package image

import (
	"net/url"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/JSYoo5B/go-openstacksdk/internal/calls"
	"github.com/JSYoo5B/go-openstacksdk/request"
)

// GetImageOpts supplies ordinary request headers.
type GetImageOpts struct{ Headers map[string]string }

// GetImageOption is one option of GetImage: WithGetImageOpts,
// WithGetImageHeaders (merges), request.Header (merges one header) or
// request.Custom over *GetImageOpts. Other shared calls, including
// request.Headers, fail with resource.ErrInvalidOption before HTTP.
type GetImageOption = request.GetCall

// ListImagesOpts supplies literal server filters and local iteration controls.
// Pointer fields preserve explicit empty strings, false and zero. Repeated
// filters retain order and duplicates; MaxItems never changes the wire limit.
type ListImagesOpts struct {
	Headers                  map[string]string
	ID                       *string
	Name                     *string
	Visibility               *string
	MemberStatus             *string
	Owner                    *string
	Status                   *string
	Sort                     *string
	CreatedAt                *string
	UpdatedAt                *string
	ContainerFormat          *string
	DiskFormat               *string
	Limit                    *int
	SizeMin                  *int64
	SizeMax                  *int64
	Protected                *bool
	Hidden                   *bool
	Marker                   string
	Tags, SortKeys, SortDirs []string
	Filters                  url.Values
	MaxItems                 int
	SinglePage               bool
}

// ListImagesOption is one option of ListImages and AllImages:
// WithListImagesOpts, WithListImagesHeaders (merges), the typed filter and
// sort options, request.Header (merges one header), request.Limit,
// request.Marker, request.MaxItems or request.Custom over *ListImagesOpts.
// Other shared calls, including request.Headers, request.Query and
// request.Paginated, fail with resource.ErrInvalidOption before HTTP.
type ListImagesOption = request.ListCall

var (
	getImageCalls   = calls.Spec[GetImageOpts]{Header: calls.MergeHeader[GetImageOpts](WithGetImageHeaders)}
	listImagesCalls = calls.Spec[ListImagesOpts]{
		Header: calls.MergeHeader[ListImagesOpts](WithListImagesHeaders),
		Paging: calls.Paging[ListImagesOpts]{
			Limit:    calls.Pointer(func(c *ListImagesOpts) **int { return &c.Limit }),
			Marker:   calls.Value(func(c *ListImagesOpts) *string { return &c.Marker }),
			MaxItems: calls.Value(func(c *ListImagesOpts) *int { return &c.MaxItems }),
		},
	}
)

func WithGetImageOpts(value GetImageOpts) GetImageOption {
	snapshot := copyGetImageOpts(value)
	return request.Custom(func(config *GetImageOpts) error { *config = copyGetImageOpts(snapshot); return nil })
}
func WithGetImageHeaders(value map[string]string) GetImageOption {
	apply := mergeImageMutationHeaders(value)
	return request.Custom(func(config *GetImageOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}
func WithListImagesOpts(value ListImagesOpts) ListImagesOption {
	snapshot := copyListImagesOpts(value)
	return request.Custom(func(config *ListImagesOpts) error { *config = copyListImagesOpts(snapshot); return nil })
}
func WithListImagesHeaders(value map[string]string) ListImagesOption {
	apply := mergeImageMutationHeaders(value)
	return request.Custom(func(config *ListImagesOpts) error { return applyImageMemberHeader(&config.Headers, apply) })
}
func WithListImagesID(value string) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.ID = &snapshot; return nil })
}
func WithListImagesName(value string) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.Name = &snapshot; return nil })
}
func WithListImagesVisibility(value string) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.Visibility = &snapshot; return nil })
}
func WithListImagesMemberStatus(value string) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.MemberStatus = &snapshot; return nil })
}
func WithListImagesOwner(value string) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.Owner = &snapshot; return nil })
}
func WithListImagesStatus(value string) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.Status = &snapshot; return nil })
}
func WithListImagesSort(value string) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.Sort = &snapshot; return nil })
}
func WithListImagesCreatedAt(value string) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.CreatedAt = &snapshot; return nil })
}
func WithListImagesUpdatedAt(value string) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.UpdatedAt = &snapshot; return nil })
}
func WithListImagesContainerFormat(value string) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.ContainerFormat = &snapshot; return nil })
}
func WithListImagesDiskFormat(value string) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.DiskFormat = &snapshot; return nil })
}
func WithListImagesSizeMin(value int64) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.SizeMin = &snapshot; return nil })
}
func WithListImagesSizeMax(value int64) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.SizeMax = &snapshot; return nil })
}
func WithListImagesProtected(value bool) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.Protected = &snapshot; return nil })
}
func WithListImagesHidden(value bool) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { snapshot := value; config.Hidden = &snapshot; return nil })
}
func WithListImagesTags(values ...string) ListImagesOption {
	snapshot := append([]string(nil), values...)
	return request.Custom(func(config *ListImagesOpts) error { config.Tags = append([]string(nil), snapshot...); return nil })
}
func WithListImagesSortKeys(values ...string) ListImagesOption {
	snapshot := append([]string(nil), values...)
	return request.Custom(func(config *ListImagesOpts) error { config.SortKeys = append([]string(nil), snapshot...); return nil })
}
func WithListImagesSortDirs(values ...string) ListImagesOption {
	snapshot := append([]string(nil), values...)
	return request.Custom(func(config *ListImagesOpts) error { config.SortDirs = append([]string(nil), snapshot...); return nil })
}

// WithListImagesFilter replaces one extension key; no values deletes that key.
func WithListImagesFilter(key string, values ...string) ListImagesOption {
	snapshot := append([]string(nil), values...)
	return request.Custom(func(config *ListImagesOpts) error {
		if config.Filters == nil {
			config.Filters = make(url.Values)
		}
		if len(snapshot) == 0 {
			delete(config.Filters, key)
		} else {
			config.Filters[key] = append([]string(nil), snapshot...)
		}
		return nil
	})
}

// WithListImagesFilters replaces the complete extension filter map.
func WithListImagesFilters(values url.Values) ListImagesOption {
	snapshot := copyImageQuery(values)
	return request.Custom(func(config *ListImagesOpts) error { config.Filters = copyImageQuery(snapshot); return nil })
}
func WithListImagesSinglePage(value bool) ListImagesOption {
	return request.Custom(func(config *ListImagesOpts) error { config.SinglePage = value; return nil })
}

func copyGetImageOpts(value GetImageOpts) GetImageOpts {
	value.Headers = taskHeadersCopy(value.Headers)
	return value
}
func copyImageQuery(value url.Values) url.Values {
	owned := make(url.Values, len(value))
	for key, values := range value {
		owned[key] = append([]string(nil), values...)
	}
	return owned
}
func copyListImagesOpts(value ListImagesOpts) ListImagesOpts {
	value.Headers = taskHeadersCopy(value.Headers)
	value.Filters = copyImageQuery(value.Filters)
	value.Tags = append([]string(nil), value.Tags...)
	value.SortKeys = append([]string(nil), value.SortKeys...)
	value.SortDirs = append([]string(nil), value.SortDirs...)
	if value.ID != nil {
		snapshot := *value.ID
		value.ID = &snapshot
	}
	if value.Name != nil {
		snapshot := *value.Name
		value.Name = &snapshot
	}
	if value.Visibility != nil {
		snapshot := *value.Visibility
		value.Visibility = &snapshot
	}
	if value.MemberStatus != nil {
		snapshot := *value.MemberStatus
		value.MemberStatus = &snapshot
	}
	if value.Owner != nil {
		snapshot := *value.Owner
		value.Owner = &snapshot
	}
	if value.Status != nil {
		snapshot := *value.Status
		value.Status = &snapshot
	}
	if value.Sort != nil {
		snapshot := *value.Sort
		value.Sort = &snapshot
	}
	if value.CreatedAt != nil {
		snapshot := *value.CreatedAt
		value.CreatedAt = &snapshot
	}
	if value.UpdatedAt != nil {
		snapshot := *value.UpdatedAt
		value.UpdatedAt = &snapshot
	}
	if value.ContainerFormat != nil {
		snapshot := *value.ContainerFormat
		value.ContainerFormat = &snapshot
	}
	if value.DiskFormat != nil {
		snapshot := *value.DiskFormat
		value.DiskFormat = &snapshot
	}
	if value.Limit != nil {
		snapshot := *value.Limit
		value.Limit = &snapshot
	}
	if value.SizeMin != nil {
		snapshot := *value.SizeMin
		value.SizeMin = &snapshot
	}
	if value.SizeMax != nil {
		snapshot := *value.SizeMax
		value.SizeMax = &snapshot
	}
	if value.Protected != nil {
		snapshot := *value.Protected
		value.Protected = &snapshot
	}
	if value.Hidden != nil {
		snapshot := *value.Hidden
		value.Hidden = &snapshot
	}
	return value
}
func parseGetImageOptions(options []GetImageOption) (GetImageOpts, error) {
	value, err := applyTaskOptions(GetImageOpts{}, calls.Translate(options, getImageCalls), copyGetImageOpts)
	if err == nil {
		value.Headers, err = imageMutationHeaders(value.Headers, false, "")
	}
	return value, err
}
func parseListImagesOptions(options []ListImagesOption) (ListImagesOpts, url.Values, error) {
	value, err := applyTaskOptions(ListImagesOpts{}, calls.Translate(options, listImagesCalls), copyListImagesOpts)
	query := make(url.Values)
	if err != nil {
		return value, query, err
	}
	if value.MaxItems < 0 || value.Limit != nil && *value.Limit < 0 || value.SizeMin != nil && *value.SizeMin < 0 || value.SizeMax != nil && *value.SizeMax < 0 {
		return value, query, uploadInvalid("image limits and sizes must be non-negative")
	}
	if value.Sort != nil && (len(value.SortKeys) > 0 || len(value.SortDirs) > 0) {
		return value, query, uploadInvalid("image sort cannot be combined with classic sort fields")
	}
	if len(value.SortDirs) > 1 && len(value.SortDirs) != len(value.SortKeys) {
		return value, query, uploadInvalid("multiple image sort directions require equal sort key count")
	}
	for key, text := range map[string]*string{"id": value.ID, "name": value.Name, "visibility": value.Visibility, "member_status": value.MemberStatus, "owner": value.Owner, "status": value.Status, "sort": value.Sort, "created_at": value.CreatedAt, "updated_at": value.UpdatedAt, "container_format": value.ContainerFormat, "disk_format": value.DiskFormat} {
		if text != nil {
			query.Set(key, *text)
		}
	}
	if value.Marker != "" {
		query.Set("marker", value.Marker)
	}
	if value.Limit != nil {
		query.Set("limit", strconv.Itoa(*value.Limit))
	}
	if value.SizeMin != nil {
		query.Set("size_min", strconv.FormatInt(*value.SizeMin, 10))
	}
	if value.SizeMax != nil {
		query.Set("size_max", strconv.FormatInt(*value.SizeMax, 10))
	}
	if value.Protected != nil {
		query.Set("protected", strconv.FormatBool(*value.Protected))
	}
	if value.Hidden != nil {
		query.Set("os_hidden", strconv.FormatBool(*value.Hidden))
	}
	for key, values := range map[string][]string{"tag": value.Tags, "sort_key": value.SortKeys, "sort_dir": value.SortDirs} {
		for _, text := range values {
			query.Add(key, text)
		}
	}
	reserved := map[string]bool{"limit": true, "marker": true, "id": true, "name": true, "visibility": true, "member_status": true, "owner": true, "status": true, "size_min": true, "size_max": true, "protected": true, "os_hidden": true, "sort": true, "sort_key": true, "sort_dir": true, "tag": true, "created_at": true, "updated_at": true, "container_format": true, "disk_format": true, "deleted": true, "max_items": true, "single_page": true}
	for key, values := range value.Filters {
		if key == "" || !utf8.ValidString(key) || reserved[key] || len(values) == 0 {
			return value, query, uploadInvalid("invalid or owned image filter key %q", key)
		}
		for _, r := range key {
			if unicode.IsControl(r) {
				return value, query, uploadInvalid("image filter keys must not contain controls")
			}
		}
		query[key] = append([]string(nil), values...)
	}
	for _, values := range query {
		for _, text := range values {
			if !utf8.ValidString(text) {
				return value, query, uploadInvalid("image query values must be valid UTF-8")
			}
		}
	}
	value.Headers, err = imageMutationHeaders(value.Headers, false, "")
	return value, query, err
}
