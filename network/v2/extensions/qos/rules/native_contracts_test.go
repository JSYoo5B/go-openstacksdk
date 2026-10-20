package rules_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

type nativeRuleTransport func(*http.Request) (*http.Response, error)

func (transport nativeRuleTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	response, err := transport(req)
	if response != nil && response.Request == nil {
		response.Request = req
	}
	return response, err
}

func nativeRuleWire(code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}}
}

type nativeRuleCall struct{ method, path, query, body string }

func nativeRuleAPI(t *testing.T, calls *[]nativeRuleCall, reply func(*http.Request) *http.Response) (*rules.API, *testcloud.Cloud) {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = nativeRuleTransport(func(req *http.Request) (*http.Response, error) {
		raw := ""
		if req.Body != nil {
			data, _ := io.ReadAll(req.Body)
			raw = string(data)
		}
		*calls = append(*calls, nativeRuleCall{req.Method, req.URL.Path, req.URL.RawQuery, raw})
		return reply(req), nil
	})
	client := cloud.Client("network", "/catalog/unused/")
	client.ResourceBase = cloud.Server.URL + "/neutron/v2.0/"
	return rules.New(client), cloud
}

func nativeRuleOperation(t *testing.T, err error, operation string) {
	t.Helper()
	var wrapped *resource.OperationError
	if !errors.As(err, &wrapped) || wrapped.Operation != operation || wrapped.Resource != "rules" {
		t.Fatal("generated rules context", err, wrapped)
	}
}

// nativeRuleKind exercises one rule family through closures that hide its concrete types.
type nativeRuleKind struct {
	name, collection, envelope, row string
	create, createZero              func(context.Context, *rules.API) (string, error)
	get                             func(context.Context, *rules.API) (string, error)
	update, updateEmpty             func(context.Context, *rules.API) (string, error)
	del                             func(context.Context, *rules.API) error
	list                            func(context.Context, *rules.API, bool) ([]string, []error)
	createBody, createZeroBody      string
	updateBody, query               string
	collision                       func(context.Context, *rules.API) error
}

func collect[T any](seq func(func(*T, error) bool), id func(*T) string) ([]string, []error) {
	var ids []string
	var errs []error
	for value, err := range seq {
		if err != nil {
			errs = append(errs, err)
			continue
		}
		ids = append(ids, id(value))
	}
	return ids, errs
}

func nativeRuleKinds() []nativeRuleKind {
	one, zero := 1, 0
	return []nativeRuleKind{
		{
			name: "BandwidthLimitRule", collection: "bandwidth_limit_rules", envelope: "bandwidth_limit_rule",
			row: `{"id":"r-1","max_kbps":1000,"max_burst_kbps":0,"direction":"egress","tags":null}`,
			create: func(ctx context.Context, api *rules.API) (string, error) {
				v, err := api.CreateBandwidthLimitRule(ctx, "qp", rules.CreateBandwidthLimitRuleOpts{MaxKBps: 1000, MaxBurstKBps: 100, Direction: "egress"}, rules.WithCreateBandwidthLimitRuleField("x_extension", 1))
				return idOf(v, err, func() string { return fmt.Sprint(v.ID, v.MaxKBps, v.Direction) })
			},
			createZero: func(ctx context.Context, api *rules.API) (string, error) {
				_, err := api.CreateBandwidthLimitRule(ctx, "qp", rules.CreateBandwidthLimitRuleOpts{})
				return "", err
			},
			get: func(ctx context.Context, api *rules.API) (string, error) {
				v, err := api.GetBandwidthLimitRule(ctx, "qp", "r-1")
				return idOf(v, err, func() string { return fmt.Sprint(v.ID, v.MaxKBps, v.Direction) })
			},
			update: func(ctx context.Context, api *rules.API) (string, error) {
				_, err := api.UpdateBandwidthLimitRule(ctx, "qp", "r-1", rules.UpdateBandwidthLimitRuleOpts{MaxKBps: &one, MaxBurstKBps: &zero}, rules.WithUpdateBandwidthLimitRuleField("x_extension", 1))
				return "", err
			},
			updateEmpty: func(ctx context.Context, api *rules.API) (string, error) {
				_, err := api.UpdateBandwidthLimitRule(ctx, "qp", "r-1", rules.UpdateBandwidthLimitRuleOpts{})
				return "", err
			},
			del: func(ctx context.Context, api *rules.API) error { return api.DeleteBandwidthLimitRule(ctx, "qp", "r-1") },
			list: func(ctx context.Context, api *rules.API, filtered bool) ([]string, []error) {
				var options []rules.ListBandwidthLimitRulesOption
				if filtered {
					options = append(options, rules.WithListBandwidthLimitRulesOptions(rules.BandwidthLimitRulesListOpts{MaxKBps: 1000, Direction: "egress", Limit: 1}), rules.WithListBandwidthLimitRulesQuery("extra", "1"))
				}
				return collect(api.ListBandwidthLimitRules(ctx, "qp", options...), func(v *rules.BandwidthLimitRule) string { return v.ID })
			},
			createBody:     `{"bandwidth_limit_rule":{"direction":"egress","max_burst_kbps":100,"max_kbps":1000,"x_extension":1}}`,
			createZeroBody: `{"bandwidth_limit_rule":{"max_kbps":0}}`,
			updateBody:     `{"bandwidth_limit_rule":{"max_burst_kbps":0,"max_kbps":1,"x_extension":1}}`,
			query:          "direction=egress&extra=1&limit=1&max_kbps=1000",
			collision: func(ctx context.Context, api *rules.API) error {
				_, err := api.CreateBandwidthLimitRule(ctx, "qp", rules.CreateBandwidthLimitRuleOpts{}, rules.WithCreateBandwidthLimitRuleField("max_kbps", 1))
				return err
			},
		},
		{
			name: "DSCPMarkingRule", collection: "dscp_marking_rules", envelope: "dscp_marking_rule",
			row: `{"id":"r-1","dscp_mark":26}`,
			create: func(ctx context.Context, api *rules.API) (string, error) {
				v, err := api.CreateDSCPMarkingRule(ctx, "qp", rules.CreateDSCPMarkingRuleOpts{DSCPMark: 26}, rules.WithCreateDSCPMarkingRuleField("x_extension", 1))
				return idOf(v, err, func() string { return fmt.Sprint(v.ID, v.DSCPMark) })
			},
			createZero: func(ctx context.Context, api *rules.API) (string, error) {
				_, err := api.CreateDSCPMarkingRule(ctx, "qp", rules.CreateDSCPMarkingRuleOpts{})
				return "", err
			},
			get: func(ctx context.Context, api *rules.API) (string, error) {
				v, err := api.GetDSCPMarkingRule(ctx, "qp", "r-1")
				return idOf(v, err, func() string { return fmt.Sprint(v.ID, v.DSCPMark) })
			},
			update: func(ctx context.Context, api *rules.API) (string, error) {
				_, err := api.UpdateDSCPMarkingRule(ctx, "qp", "r-1", rules.UpdateDSCPMarkingRuleOpts{DSCPMark: &zero}, rules.WithUpdateDSCPMarkingRuleField("x_extension", 1))
				return "", err
			},
			updateEmpty: func(ctx context.Context, api *rules.API) (string, error) {
				_, err := api.UpdateDSCPMarkingRule(ctx, "qp", "r-1", rules.UpdateDSCPMarkingRuleOpts{})
				return "", err
			},
			del: func(ctx context.Context, api *rules.API) error { return api.DeleteDSCPMarkingRule(ctx, "qp", "r-1") },
			list: func(ctx context.Context, api *rules.API, filtered bool) ([]string, []error) {
				var options []rules.ListDSCPMarkingRulesOption
				if filtered {
					options = append(options, rules.WithListDSCPMarkingRulesOptions(rules.DSCPMarkingRulesListOpts{DSCPMark: 26, Limit: 1}), rules.WithListDSCPMarkingRulesQuery("extra", "1"))
				}
				return collect(api.ListDSCPMarkingRules(ctx, "qp", options...), func(v *rules.DSCPMarkingRule) string { return v.ID })
			},
			createBody:     `{"dscp_marking_rule":{"dscp_mark":26,"x_extension":1}}`,
			createZeroBody: `{"dscp_marking_rule":{"dscp_mark":0}}`,
			updateBody:     `{"dscp_marking_rule":{"dscp_mark":0,"x_extension":1}}`,
			query:          "dscp_mark=26&extra=1&limit=1",
			collision: func(ctx context.Context, api *rules.API) error {
				_, err := api.CreateDSCPMarkingRule(ctx, "qp", rules.CreateDSCPMarkingRuleOpts{}, rules.WithCreateDSCPMarkingRuleField("dscp_mark", 1))
				return err
			},
		},
		{
			name: "MinimumBandwidthRule", collection: "minimum_bandwidth_rules", envelope: "minimum_bandwidth_rule",
			row: `{"id":"r-1","min_kbps":500,"direction":"ingress"}`,
			create: func(ctx context.Context, api *rules.API) (string, error) {
				v, err := api.CreateMinimumBandwidthRule(ctx, "qp", rules.CreateMinimumBandwidthRuleOpts{MinKBps: 500, Direction: "ingress"}, rules.WithCreateMinimumBandwidthRuleField("x_extension", 1))
				return idOf(v, err, func() string { return fmt.Sprint(v.ID, v.MinKBps, v.Direction) })
			},
			createZero: func(ctx context.Context, api *rules.API) (string, error) {
				_, err := api.CreateMinimumBandwidthRule(ctx, "qp", rules.CreateMinimumBandwidthRuleOpts{})
				return "", err
			},
			get: func(ctx context.Context, api *rules.API) (string, error) {
				v, err := api.GetMinimumBandwidthRule(ctx, "qp", "r-1")
				return idOf(v, err, func() string { return fmt.Sprint(v.ID, v.MinKBps, v.Direction) })
			},
			update: func(ctx context.Context, api *rules.API) (string, error) {
				_, err := api.UpdateMinimumBandwidthRule(ctx, "qp", "r-1", rules.UpdateMinimumBandwidthRuleOpts{MinKBps: &zero, Direction: "egress"}, rules.WithUpdateMinimumBandwidthRuleField("x_extension", 1))
				return "", err
			},
			updateEmpty: func(ctx context.Context, api *rules.API) (string, error) {
				_, err := api.UpdateMinimumBandwidthRule(ctx, "qp", "r-1", rules.UpdateMinimumBandwidthRuleOpts{})
				return "", err
			},
			del: func(ctx context.Context, api *rules.API) error {
				return api.DeleteMinimumBandwidthRule(ctx, "qp", "r-1")
			},
			list: func(ctx context.Context, api *rules.API, filtered bool) ([]string, []error) {
				var options []rules.ListMinimumBandwidthRulesOption
				if filtered {
					options = append(options, rules.WithListMinimumBandwidthRulesOptions(rules.MinimumBandwidthRulesListOpts{MinKBps: 500, Direction: "ingress", Limit: 1}), rules.WithListMinimumBandwidthRulesQuery("extra", "1"))
				}
				return collect(api.ListMinimumBandwidthRules(ctx, "qp", options...), func(v *rules.MinimumBandwidthRule) string { return v.ID })
			},
			createBody:     `{"minimum_bandwidth_rule":{"direction":"ingress","min_kbps":500,"x_extension":1}}`,
			createZeroBody: `{"minimum_bandwidth_rule":{"min_kbps":0}}`,
			updateBody:     `{"minimum_bandwidth_rule":{"direction":"egress","min_kbps":0,"x_extension":1}}`,
			query:          "direction=ingress&extra=1&limit=1&min_kbps=500",
			collision: func(ctx context.Context, api *rules.API) error {
				_, err := api.CreateMinimumBandwidthRule(ctx, "qp", rules.CreateMinimumBandwidthRuleOpts{}, rules.WithCreateMinimumBandwidthRuleField("min_kbps", 1))
				return err
			},
		},
	}
}

func idOf[T any](value *T, err error, describe func() string) (string, error) {
	if err != nil || value == nil {
		return "", err
	}
	return describe(), nil
}

func TestNativeQoSRuleRoutesBodiesAndDecode(t *testing.T) {
	ctx := context.Background()
	for _, kind := range nativeRuleKinds() {
		t.Run(kind.name, func(t *testing.T) {
			var calls []nativeRuleCall
			var cloud *testcloud.Cloud
			api, cloud := nativeRuleAPI(t, &calls, func(req *http.Request) *http.Response {
				switch {
				case req.Method == http.MethodDelete:
					return nativeRuleWire(204, "")
				case req.Method == http.MethodPost:
					return nativeRuleWire(201, `{"`+kind.envelope+`":`+kind.row+`}`)
				case strings.HasSuffix(req.URL.Path, "/"+kind.collection):
					// Neutron *_links arrays are ignored; only a links.next string is followed.
					return nativeRuleWire(200, `{"`+kind.collection+`":[`+kind.row+`],"`+kind.collection+`_links":[{"rel":"next","href":"`+cloud.Server.URL+`/never"}],"links":{"next":"`+cloud.Server.URL+`/other/rules"}}`)
				case req.URL.Path == "/other/rules":
					return nativeRuleWire(200, `{"`+kind.collection+`":[{"id":"r-2"}]}`)
				}
				return nativeRuleWire(200, `{"`+kind.envelope+`":`+kind.row+`}`)
			})
			created, err := kind.create(ctx, api)
			if err != nil || !strings.HasPrefix(created, "r-1") {
				t.Fatal(created, err)
			}
			if _, err := kind.createZero(ctx, api); err != nil {
				t.Fatal(err)
			}
			got, err := kind.get(ctx, api)
			if err != nil || got != created {
				t.Fatal(got, err)
			}
			if _, err := kind.update(ctx, api); err != nil {
				t.Fatal(err)
			}
			if _, err := kind.updateEmpty(ctx, api); err != nil {
				t.Fatal(err)
			}
			ids, errs := kind.list(ctx, api, true)
			if !reflect.DeepEqual(ids, []string{"r-1", "r-2"}) || errs != nil {
				t.Fatal(ids, errs)
			}
			if err := kind.del(ctx, api); err != nil {
				t.Fatal(err)
			}
			base := "/neutron/v2.0/qos/policies/qp/" + kind.collection
			query, _ := url.ParseQuery(calls[5].query)
			calls[5].query = query.Encode()
			want := []nativeRuleCall{
				{http.MethodPost, base, "", kind.createBody},
				// The required-looking rate field has no omitempty and is sent as zero.
				{http.MethodPost, base, "", kind.createZeroBody},
				{http.MethodGet, base + "/r-1", "", ""},
				// Pointer rate fields allow an explicit zero.
				{http.MethodPut, base + "/r-1", "", kind.updateBody},
				{http.MethodPut, base + "/r-1", "", `{"` + kind.envelope + `":{}}`},
				{http.MethodGet, base, kind.query, ""},
				{http.MethodGet, "/other/rules", "", ""},
				{http.MethodDelete, base + "/r-1", "", ""},
			}
			if !reflect.DeepEqual(calls, want) {
				t.Fatalf("%+v", calls)
			}
		})
	}
}

func TestNativeQoSRuleStrictStatusesAndPreflight(t *testing.T) {
	ctx := context.Background()
	for _, kind := range nativeRuleKinds() {
		for _, call := range []struct {
			operation string
			accepted  []int
			call      func(*rules.API) error
		}{
			// Create narrows the default POST codes to 201.
			{"Create" + kind.name, []int{201}, func(api *rules.API) error { _, err := kind.createZero(ctx, api); return err }},
			{"Get" + kind.name, []int{200}, func(api *rules.API) error { _, err := kind.get(ctx, api); return err }},
			{"Update" + kind.name, []int{200}, func(api *rules.API) error { _, err := kind.updateEmpty(ctx, api); return err }},
			{"Delete" + kind.name, []int{202, 204}, func(api *rules.API) error { return kind.del(ctx, api) }},
		} {
			for _, code := range []int{200, 201, 202, 204, 404} {
				if slices.Contains(call.accepted, code) {
					continue
				}
				t.Run(fmt.Sprintf("%s/%d", call.operation, code), func(t *testing.T) {
					var calls []nativeRuleCall
					api, _ := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response {
						return nativeRuleWire(code, `{"`+kind.envelope+`":{}}`)
					})
					err := call.call(api)
					nativeRuleOperation(t, err, call.operation)
					var native gophercloud.ErrUnexpectedResponseCode
					if !errors.As(err, &native) || native.Actual != code || !reflect.DeepEqual(native.Expected, call.accepted) || len(calls) != 1 {
						t.Fatal(err, native)
					}
				})
			}
		}
		t.Run(kind.name+"/list statuses", func(t *testing.T) {
			for _, tc := range []struct {
				code int
				body string
			}{{404, `{}`}, {200, `{"` + kind.collection + `":[]}`}, {204, ""}} {
				var calls []nativeRuleCall
				api, _ := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(tc.code, tc.body) })
				ids, errs := kind.list(ctx, api, false)
				var native gophercloud.ErrUnexpectedResponseCode
				switch {
				case tc.code == 404 && len(errs) == 1 && errors.As(errs[0], &native) && reflect.DeepEqual(native.Expected, []int{200, 204, 300}):
				case tc.code == 200 && len(errs) == 0 && len(ids) == 0:
				case tc.code == 204 && len(errs) == 1 && errors.Is(errs[0], io.EOF):
				default:
					t.Fatal(tc.code, ids, errs)
				}
			}
		})
		t.Run(kind.name+"/preflight", func(t *testing.T) {
			var calls []nativeRuleCall
			api, _ := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(201, `{}`) })
			err := kind.collision(ctx, api)
			if err == nil {
				t.Fatal("core extension accepted")
			}
			nativeRuleOperation(t, err, "Create"+kind.name)
			if len(calls) != 0 {
				t.Fatal(calls)
			}
		})
	}
	t.Run("nil options", func(t *testing.T) {
		var calls []nativeRuleCall
		api, _ := nativeRuleAPI(t, &calls, func(*http.Request) *http.Response { return nativeRuleWire(201, `{}`) })
		_, err := api.UpdateDSCPMarkingRule(ctx, "qp", "r-1", rules.UpdateDSCPMarkingRuleOpts{}, nil)
		nativeRuleOperation(t, err, "UpdateDSCPMarkingRule")
		for _, err := range api.ListMinimumBandwidthRules(ctx, "qp", nil) {
			nativeRuleOperation(t, err, "ListMinimumBandwidthRules")
		}
		if len(calls) != 0 {
			t.Fatal(calls)
		}
	})
}
