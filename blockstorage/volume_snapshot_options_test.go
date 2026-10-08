package blockstorage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/resource"
)

func vsopLocation() resource.CloudLocation {
	cloud, region, name, domainID, domainName := "cloud-original", "region-original", "project-original", "domain-ID", "domain-original"
	return resource.CloudLocation{Cloud: &cloud, RegionName: &region, Zone: json.RawMessage(`{"zone":[1,true]}`), Project: resource.CloudProject{ID: json.RawMessage(`{"scope":900719925474099312345}`), Name: &name, DomainID: &domainID, DomainName: &domainName}}
}

func vsopDamageLocation(location *resource.CloudLocation) {
	*location.Cloud = "changed-cloud"
	*location.RegionName = "changed-region"
	location.Zone[0] = '!'
	location.Project.ID[0] = '!'
	*location.Project.Name = "changed-name"
	*location.Project.DomainID = "changed-domain-ID"
	*location.Project.DomainName = "changed-domain-name"
}

func vsopLocationJSON(t *testing.T, location *resource.CloudLocation) []byte {
	t.Helper()
	raw, err := json.Marshal(location)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func vsopListValue() VolumeSnapshotListOpts {
	detailed, paginated, allowUnknown := false, false, false
	maxItems, version, expression := 7, "3.60", "[].id"
	filters, conflicting := json.RawMessage(`{"name":"entry","metadata":{"large":900719925474099312345}}`), json.RawMessage(`{"status":"available"}`)
	location := vsopLocation()
	return VolumeSnapshotListOpts{Detailed: &detailed, Filters: &filters, Location: &location, Paginated: &paginated, MaxItems: &maxItems, Microversion: &version, Headers: map[string]string{"X-Probe": "original", "Accept": "application/json"}, Expression: &expression, AllowUnknownParams: &allowUnknown, ConflictingAttrs: &conflicting}
}

func vsopDamageList(value *VolumeSnapshotListOpts) {
	*value.Detailed = true
	*value.Paginated = true
	*value.MaxItems = 99
	*value.Microversion = "3.99"
	*value.Expression = "changed"
	*value.AllowUnknownParams = true
	(*value.Filters)[0] = '!'
	(*value.ConflictingAttrs)[0] = '!'
	value.Headers["X-Probe"] = "changed"
	vsopDamageLocation(value.Location)
}

func vsopPrepareCallbacks(ctx context.Context, kind string, nilIndex int, callbacks ...func() error) (any, error) {
	switch kind {
	case "read":
		options := make([]VolumeSnapshotReadOption, len(callbacks))
		for i, callback := range callbacks {
			callback := callback
			if i != nilIndex {
				options[i] = func(next *VolumeSnapshotReadOpts) error {
					next.Location = &resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`{"entered":true}`)}}
					return callback()
				}
			}
		}
		return PrepareVolumeSnapshotReadOptions(ctx, options...)
	case "search":
		options := make([]VolumeSnapshotSearchOption, len(callbacks))
		for i, callback := range callbacks {
			callback := callback
			if i != nilIndex {
				options[i] = func(next *VolumeSnapshotSearchOpts) error {
					next.Location = &resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`{"entered":true}`)}}
					return callback()
				}
			}
		}
		return PrepareVolumeSnapshotSearchOptions(ctx, options...)
	case "list":
		options := make([]VolumeSnapshotListOption, len(callbacks))
		for i, callback := range callbacks {
			callback := callback
			if i != nilIndex {
				options[i] = func(next *VolumeSnapshotListOpts) error {
					next.Location = &resource.CloudLocation{Project: resource.CloudProject{ID: json.RawMessage(`{"entered":true}`)}}
					return callback()
				}
			}
		}
		return PrepareVolumeSnapshotListOptions(ctx, options...)
	default:
		panic("unknown snapshot option family")
	}
}

func vsopZero(kind string) any {
	switch kind {
	case "read":
		return VolumeSnapshotReadOpts{}
	case "search":
		return VolumeSnapshotSearchOpts{}
	case "list":
		return VolumeSnapshotListOpts{}
	default:
		panic("unknown snapshot option family")
	}
}

func TestVolumeSnapshotPreparePreservesOmissionAndExplicitFalseZeroEmptyPolicies(t *testing.T) {
	read, err := PrepareVolumeSnapshotReadOptions(context.Background())
	if err != nil || !reflect.DeepEqual(read, VolumeSnapshotReadOpts{}) {
		t.Fatal(read, err)
	}
	search, err := PrepareVolumeSnapshotSearchOptions(context.Background())
	if err != nil || !reflect.DeepEqual(search, VolumeSnapshotSearchOpts{}) {
		t.Fatal(search, err)
	}
	list, err := PrepareVolumeSnapshotListOptions(context.Background())
	if err != nil || !reflect.DeepEqual(list, VolumeSnapshotListOpts{}) {
		t.Fatal(list, err)
	}
	list, err = PrepareVolumeSnapshotListOptions(context.Background(), WithVolumeSnapshotListDetailed(false), WithVolumeSnapshotListPagination(false), WithVolumeSnapshotListMaxItems(0), WithVolumeSnapshotListMicroversion(""), WithVolumeSnapshotListExpression(""), WithVolumeSnapshotListAllowUnknownParams(false), WithVolumeSnapshotListHeaders(map[string]string{}))
	if err != nil || list.Detailed == nil || *list.Detailed || list.Paginated == nil || *list.Paginated || list.MaxItems == nil || *list.MaxItems != 0 || list.Microversion == nil || *list.Microversion != "" || list.Expression == nil || *list.Expression != "" || list.AllowUnknownParams == nil || *list.AllowUnknownParams || list.Headers == nil || len(list.Headers) != 0 || list.Filters != nil || list.Location != nil || list.ConflictingAttrs != nil {
		t.Fatal(list, err)
	}
	explicit, err := PrepareVolumeSnapshotListOptions(context.Background(), WithVolumeSnapshotListHeaders(nil))
	if err != nil || explicit.Headers != nil {
		t.Fatal(explicit, err)
	}
}

func TestVolumeSnapshotLocationFactoriesOwnEveryNullableComponentAndAreReusable(t *testing.T) {
	for _, family := range []string{"read", "search", "list"} {
		for _, factory := range []string{"options", "location"} {
			t.Run(family+"/"+factory, func(t *testing.T) {
				location := vsopLocation()
				expected := vsopLocationJSON(t, &location)
				var prepare func() (*resource.CloudLocation, error)
				switch family {
				case "read":
					option := WithVolumeSnapshotReadLocation(location)
					if factory == "options" {
						option = WithVolumeSnapshotReadOptions(VolumeSnapshotReadOpts{Location: &location})
					}
					prepare = func() (*resource.CloudLocation, error) {
						value, err := PrepareVolumeSnapshotReadOptions(context.Background(), option)
						return value.Location, err
					}
				case "search":
					option := WithVolumeSnapshotSearchLocation(location)
					if factory == "options" {
						option = WithVolumeSnapshotSearchOptions(VolumeSnapshotSearchOpts{Location: &location})
					}
					prepare = func() (*resource.CloudLocation, error) {
						value, err := PrepareVolumeSnapshotSearchOptions(context.Background(), option)
						return value.Location, err
					}
				case "list":
					option := WithVolumeSnapshotListLocation(location)
					if factory == "options" {
						option = WithVolumeSnapshotListOptions(VolumeSnapshotListOpts{Location: &location})
					}
					prepare = func() (*resource.CloudLocation, error) {
						value, err := PrepareVolumeSnapshotListOptions(context.Background(), option)
						return value.Location, err
					}
				}
				vsopDamageLocation(&location)
				first, err := prepare()
				if err != nil || first == nil || !bytes.Equal(vsopLocationJSON(t, first), expected) {
					t.Fatal(first, err)
				}
				vsopDamageLocation(first)
				second, err := prepare()
				if err != nil || second == nil || second == first || !bytes.Equal(vsopLocationJSON(t, second), expected) {
					t.Fatal("factory reused caller or prior result memory", second, err)
				}
			})
		}
	}
}

func TestVolumeSnapshotCompleteListFactoryOwnsRawHeadersPointersAndReturnedPolicies(t *testing.T) {
	input := vsopListValue()
	expected := vsopListValue()
	factory := WithVolumeSnapshotListOptions(input)
	vsopDamageList(&input)
	first, err := PrepareVolumeSnapshotListOptions(context.Background(), factory)
	if err != nil || !reflect.DeepEqual(first, expected) {
		t.Fatal("caller changed factory policy", first, err)
	}
	vsopDamageList(&first)
	second, err := PrepareVolumeSnapshotListOptions(context.Background(), factory)
	if err != nil || !reflect.DeepEqual(second, expected) {
		t.Fatal("prior result changed reusable factory", second, err)
	}
	second.Headers["new"] = "second-only"
	third, err := PrepareVolumeSnapshotListOptions(context.Background(), factory)
	if err != nil || !reflect.DeepEqual(third, expected) {
		t.Fatal("new map entry leaked between prepares", third, err)
	}
}

func TestVolumeSnapshotRawFactoriesPreserveExactJSONAndNilVersusEmptyBytes(t *testing.T) {
	for _, family := range []string{"search", "list filters", "list conflicting attrs"} {
		for _, raw := range []json.RawMessage{nil, json.RawMessage{}, json.RawMessage(`null`), json.RawMessage(`false`), json.RawMessage(`0`), json.RawMessage(`[]`), json.RawMessage(`""`), json.RawMessage(` {"decimal":123456789012345678901234567890.001,"escaped":"\u0061","ordered": [true, null]} `)} {
			t.Run(family+"/"+string(raw), func(t *testing.T) {
				expected := bytes.Clone(raw)
				var prepare func() (*json.RawMessage, error)
				switch family {
				case "search":
					option := WithVolumeSnapshotSearchFilters(raw)
					prepare = func() (*json.RawMessage, error) {
						value, err := PrepareVolumeSnapshotSearchOptions(context.Background(), option)
						return value.Filters, err
					}
				case "list filters":
					option := WithVolumeSnapshotListFilters(raw)
					prepare = func() (*json.RawMessage, error) {
						value, err := PrepareVolumeSnapshotListOptions(context.Background(), option)
						return value.Filters, err
					}
				case "list conflicting attrs":
					option := WithVolumeSnapshotListConflictingAttrs(raw)
					prepare = func() (*json.RawMessage, error) {
						value, err := PrepareVolumeSnapshotListOptions(context.Background(), option)
						return value.ConflictingAttrs, err
					}
				}
				if len(raw) > 0 {
					raw[0] = '!'
				}
				first, err := prepare()
				if err != nil || first == nil || !bytes.Equal(*first, expected) || ((*first == nil) != (expected == nil)) {
					t.Fatal(first, err)
				}
				if len(*first) > 0 {
					(*first)[0] = '?'
				}
				second, err := prepare()
				if err != nil || second == nil || second == first || !bytes.Equal(*second, expected) || ((*second == nil) != (expected == nil)) {
					t.Fatal("raw factory returned shared storage", second, err)
				}
			})
		}
	}
}

func TestVolumeSnapshotSearchExpressionFactoryEncodesLiteralAndSearchOptionsOwnItsInputs(t *testing.T) {
	expression := "[].{id:id, label:'quoted\\path\n\"value\"'}"
	expected, _ := json.Marshal(expression)
	value, err := PrepareVolumeSnapshotSearchOptions(context.Background(), WithVolumeSnapshotSearchExpression(expression))
	if err != nil || value.Filters == nil || !bytes.Equal(*value.Filters, expected) {
		t.Fatal(value, err)
	}
	raw := json.RawMessage(`{"metadata":{"x":[true,900719925474099312345]}}`)
	ownedRaw := bytes.Clone(raw)
	location := vsopLocation()
	ownedLocation := vsopLocationJSON(t, &location)
	factory := WithVolumeSnapshotSearchOptions(VolumeSnapshotSearchOpts{Filters: &raw, Location: &location})
	raw[0] = '!'
	vsopDamageLocation(&location)
	first, err := PrepareVolumeSnapshotSearchOptions(context.Background(), factory)
	if err != nil || first.Filters == nil || !bytes.Equal(*first.Filters, ownedRaw) || !bytes.Equal(vsopLocationJSON(t, first.Location), ownedLocation) {
		t.Fatal(first, err)
	}
	(*first.Filters)[0] = '!'
	vsopDamageLocation(first.Location)
	second, err := PrepareVolumeSnapshotSearchOptions(context.Background(), factory)
	if err != nil || second.Filters == first.Filters || !bytes.Equal(*second.Filters, ownedRaw) || !bytes.Equal(vsopLocationJSON(t, second.Location), ownedLocation) {
		t.Fatal(second, err)
	}
}

func TestVolumeSnapshotIndividualListFactoriesRemainIndependentAndLaterOptionsOverride(t *testing.T) {
	filters, conflicting := json.RawMessage(`{"name":"chosen"}`), json.RawMessage(`{"status":"available"}`)
	headers := map[string]string{"X-Probe": "entry"}
	filtersFactory, conflictingFactory, headersFactory := WithVolumeSnapshotListFilters(filters), WithVolumeSnapshotListConflictingAttrs(conflicting), WithVolumeSnapshotListHeaders(headers)
	filters[0] = '!'
	conflicting[0] = '!'
	headers["X-Probe"] = "caller changed"
	options := []VolumeSnapshotListOption{WithVolumeSnapshotListOptions(vsopListValue()), filtersFactory, conflictingFactory, headersFactory, WithVolumeSnapshotListDetailed(true), WithVolumeSnapshotListPagination(true), WithVolumeSnapshotListMaxItems(0), WithVolumeSnapshotListMicroversion(""), WithVolumeSnapshotListExpression(""), WithVolumeSnapshotListAllowUnknownParams(true)}
	first, err := PrepareVolumeSnapshotListOptions(context.Background(), options...)
	if err != nil || string(*first.Filters) != `{"name":"chosen"}` || string(*first.ConflictingAttrs) != `{"status":"available"}` || first.Headers["X-Probe"] != "entry" || !*first.Detailed || !*first.Paginated || *first.MaxItems != 0 || *first.Microversion != "" || *first.Expression != "" || !*first.AllowUnknownParams || first.Location == nil {
		t.Fatal(first, err)
	}
	vsopDamageList(&first)
	second, err := PrepareVolumeSnapshotListOptions(context.Background(), options...)
	if err != nil || string(*second.Filters) != `{"name":"chosen"}` || string(*second.ConflictingAttrs) != `{"status":"available"}` || second.Headers["X-Probe"] != "entry" || !*second.Detailed || !*second.Paginated || *second.MaxItems != 0 || *second.Microversion != "" || *second.Expression != "" || !*second.AllowUnknownParams {
		t.Fatal(second, err)
	}
}

func TestVolumeSnapshotWholeOptionsReplaceEarlierPolicyRatherThanMerging(t *testing.T) {
	location := vsopLocation()
	read, err := PrepareVolumeSnapshotReadOptions(context.Background(), WithVolumeSnapshotReadLocation(location), WithVolumeSnapshotReadOptions(VolumeSnapshotReadOpts{}))
	if err != nil || read.Location != nil {
		t.Fatal(read, err)
	}
	search, err := PrepareVolumeSnapshotSearchOptions(context.Background(), WithVolumeSnapshotSearchLocation(location), WithVolumeSnapshotSearchFilters(json.RawMessage(`false`)), WithVolumeSnapshotSearchOptions(VolumeSnapshotSearchOpts{}))
	if err != nil || search.Location != nil || search.Filters != nil {
		t.Fatal(search, err)
	}
	list, err := PrepareVolumeSnapshotListOptions(context.Background(), WithVolumeSnapshotListOptions(vsopListValue()), WithVolumeSnapshotListOptions(VolumeSnapshotListOpts{}), WithVolumeSnapshotListDetailed(false))
	if err != nil || list.Detailed == nil || *list.Detailed {
		t.Fatal(list, err)
	}
	list.Detailed = nil
	if !reflect.DeepEqual(list, VolumeSnapshotListOpts{}) {
		t.Fatal("whole options retained earlier fields", list)
	}
}

func TestVolumeSnapshotOriginalsRunOnceInCapturedOrderAndRetainedConfigsCannotAlterLaterPolicy(t *testing.T) {
	t.Run("read", func(t *testing.T) {
		location := vsopLocation()
		expected := vsopLocationJSON(t, &location)
		var retained, late *VolumeSnapshotReadOpts
		var order []int
		options := make([]VolumeSnapshotReadOption, 2)
		options[0] = func(next *VolumeSnapshotReadOpts) error {
			order = append(order, 1)
			next.Location = &location
			retained = next
			options[1] = func(*VolumeSnapshotReadOpts) error { return errors.New("replaced callback ran") }
			return nil
		}
		options[1] = func(next *VolumeSnapshotReadOpts) error {
			order = append(order, 2)
			vsopDamageLocation(retained.Location)
			if !bytes.Equal(vsopLocationJSON(t, next.Location), expected) {
				t.Fatal("retained config leaked into next callback")
			}
			late = next
			return nil
		}
		got, err := PrepareVolumeSnapshotReadOptions(context.Background(), options...)
		if err != nil || !reflect.DeepEqual(order, []int{1, 2}) {
			t.Fatal(got, err, order)
		}
		vsopDamageLocation(late.Location)
		if !bytes.Equal(vsopLocationJSON(t, got.Location), expected) {
			t.Fatal("last retained config changed final prepare")
		}
	})
	t.Run("search", func(t *testing.T) {
		raw, location := json.RawMessage(`{"x":true}`), vsopLocation()
		expectedLocation := vsopLocationJSON(t, &location)
		var retained, late *VolumeSnapshotSearchOpts
		var order []int
		options := make([]VolumeSnapshotSearchOption, 2)
		options[0] = func(next *VolumeSnapshotSearchOpts) error {
			order = append(order, 1)
			next.Filters = &raw
			next.Location = &location
			retained = next
			options[1] = func(*VolumeSnapshotSearchOpts) error { return errors.New("replaced callback ran") }
			return nil
		}
		options[1] = func(next *VolumeSnapshotSearchOpts) error {
			order = append(order, 2)
			(*retained.Filters)[0] = '!'
			vsopDamageLocation(retained.Location)
			if string(*next.Filters) != `{"x":true}` || !bytes.Equal(vsopLocationJSON(t, next.Location), expectedLocation) {
				t.Fatal("retained search config leaked")
			}
			late = next
			return nil
		}
		got, err := PrepareVolumeSnapshotSearchOptions(context.Background(), options...)
		if err != nil || !reflect.DeepEqual(order, []int{1, 2}) {
			t.Fatal(got, err, order)
		}
		(*late.Filters)[0] = '!'
		vsopDamageLocation(late.Location)
		if string(*got.Filters) != `{"x":true}` || !bytes.Equal(vsopLocationJSON(t, got.Location), expectedLocation) {
			t.Fatal("last search config changed final prepare")
		}
	})
	t.Run("list", func(t *testing.T) {
		original, expected := vsopListValue(), vsopListValue()
		var retained, late *VolumeSnapshotListOpts
		var order []int
		options := make([]VolumeSnapshotListOption, 2)
		options[0] = func(next *VolumeSnapshotListOpts) error {
			order = append(order, 1)
			*next = original
			retained = next
			options[1] = func(*VolumeSnapshotListOpts) error { return errors.New("replaced callback ran") }
			return nil
		}
		options[1] = func(next *VolumeSnapshotListOpts) error {
			order = append(order, 2)
			vsopDamageList(retained)
			if !reflect.DeepEqual(*next, expected) {
				t.Fatal("retained list config leaked")
			}
			late = next
			return nil
		}
		got, err := PrepareVolumeSnapshotListOptions(context.Background(), options...)
		if err != nil || !reflect.DeepEqual(order, []int{1, 2}) {
			t.Fatal(got, err, order)
		}
		vsopDamageList(late)
		if !reflect.DeepEqual(got, expected) {
			t.Fatal("last list config changed final prepare", got)
		}
	})
}

func TestVolumeSnapshotPrepareRejectsNilAndCanceledContextBeforeOriginals(t *testing.T) {
	for _, family := range []string{"read", "search", "list"} {
		for _, kind := range []string{"nil", "canceled", "deadline"} {
			t.Run(family+"/"+kind, func(t *testing.T) {
				sentinel := errors.New("snapshot caller cause")
				var ctx context.Context
				if kind == "canceled" {
					selected, cancel := context.WithCancelCause(context.Background())
					cancel(sentinel)
					ctx = selected
				}
				if kind == "deadline" {
					selected, cancel := context.WithTimeout(context.Background(), 0)
					defer cancel()
					ctx = selected
				}
				calls := 0
				_, err := vsopPrepareCallbacks(ctx, family, -1, func() error { calls++; return nil })
				if err == nil || calls != 0 {
					t.Fatal("preflight ran original", err, calls)
				}
				if kind == "nil" && !errors.Is(err, resource.ErrInvalidOption) {
					t.Fatal(err)
				}
				if kind == "canceled" && (!errors.Is(err, context.Canceled) || !errors.Is(err, sentinel)) {
					t.Fatal("lost cancel cause", err)
				}
				if kind == "deadline" && !errors.Is(err, context.DeadlineExceeded) {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestVolumeSnapshotPrepareNilOptionAndCallbackFailuresStopWithoutPartialPolicy(t *testing.T) {
	for _, family := range []string{"read", "search", "list"} {
		t.Run(family+"/nil", func(t *testing.T) {
			calls := 0
			got, err := vsopPrepareCallbacks(context.Background(), family, 1, func() error { calls++; return nil }, func() error { t.Fatal("nil slot invoked"); return nil }, func() error { t.Fatal("callback after nil invoked"); return nil })
			if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 || !reflect.DeepEqual(got, vsopZero(family)) {
				t.Fatal(got, err, calls)
			}
		})
		for _, cancelInside := range []bool{false, true} {
			t.Run(family+"/callback", func(t *testing.T) {
				sentinel, cause := errors.New("original callback failed"), errors.New("callback canceled parent")
				ctx, cancel := context.WithCancelCause(context.Background())
				defer cancel(nil)
				calls := 0
				got, err := vsopPrepareCallbacks(ctx, family, -1, func() error {
					calls++
					if cancelInside {
						cancel(cause)
					}
					return sentinel
				}, func() error { t.Fatal("callback after failure invoked"); return nil })
				if !errors.Is(err, sentinel) || calls != 1 {
					t.Fatal(got, err, calls)
				}
				if cancelInside && (!errors.Is(err, cause) || !errors.Is(err, context.Canceled)) {
					t.Fatal("callback/context causes not joined", err)
				}
				if !reflect.DeepEqual(got, vsopZero(family)) {
					t.Fatal("failed prepare exposed partial policy", got)
				}
			})
		}
		t.Run(family+"/cancel-only", func(t *testing.T) {
			cause := errors.New("option cancel-only cause")
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			got, err := vsopPrepareCallbacks(ctx, family, -1, func() error { cancel(cause); return nil }, func() error { t.Fatal("callback after cancellation invoked"); return nil })
			if !errors.Is(err, cause) || !errors.Is(err, context.Canceled) || !reflect.DeepEqual(got, vsopZero(family)) {
				t.Fatal(got, err)
			}
		})
	}
}
