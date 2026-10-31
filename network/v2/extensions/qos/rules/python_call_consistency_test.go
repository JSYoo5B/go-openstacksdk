package rules_test

import (
	"context"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/network/v2/extensions/qos/rules"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
)

// qosCallCandidates holds one benign value of every shared call kind. Each
// operation only receives the kinds within its own option category.
func qosCallCandidates(kind string) map[string]request.Call {
	return map[string]request.Call{
		"Header":        request.Header("X-Trace", "consistency"),
		"Headers":       request.Headers(map[string]string{"X-Trace": "consistency"}),
		"Microversion":  request.Microversion("2.1"),
		"Query":         request.Query("name", "consistency"),
		"IgnoreMissing": request.IgnoreMissing(true),
		"BasePath":      request.BasePath("qos/policies/parent/" + kind),
		"Limit":         request.Limit(5),
		"Marker":        request.Marker("marker"),
		"MaxItems":      request.MaxItems(1),
		"Paginated":     request.Paginated(false),
		"Field":         request.Field("max_kpps", 1),
	}
}

func qosAs[T request.Call](calls []request.Call) ([]T, bool) {
	out := make([]T, 0, len(calls))
	for _, call := range calls {
		typed, ok := call.(T)
		if !ok {
			return nil, false
		}
		out = append(out, typed)
	}
	return out, true
}

func qosListErr[T any](values func(func(T, error) bool)) error {
	for _, err := range values {
		if err != nil {
			return err
		}
	}
	return nil
}

type qosCallOperation struct {
	name  string
	kind  string
	calls []string
	// run returns ok=false when a call is outside the operation's category.
	run func(*rules.API, []request.Call) (error, bool)
}

func qosRun[T request.Call](calls []request.Call, run func([]T) error) (error, bool) {
	typed, ok := qosAs[T](calls)
	if !ok {
		return nil, false
	}
	return run(typed), true
}

var (
	qosFindCalls   = []string{"Header", "Headers", "Microversion", "Query", "IgnoreMissing"}
	qosGetCalls    = []string{"Header", "Headers", "Microversion"}
	qosListCalls   = []string{"Header", "Headers", "Microversion", "Query", "BasePath", "Limit", "Marker", "MaxItems", "Paginated"}
	qosRateLists   = []string{"Header", "Headers", "Microversion", "Query", "Limit", "Marker", "MaxItems", "Paginated"}
	qosDeleteCalls = []string{"Header", "Headers", "Microversion", "IgnoreMissing"}
	qosWriteCalls  = []string{"Header", "Headers", "Microversion", "Field"}
)

func qosCallOperations() []qosCallOperation {
	ctx := context.Background()
	return []qosCallOperation{
		{"FindBandwidthLimitRecord", "bandwidth_limit_rules", qosFindCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.BandwidthLimitFindOption) error {
				_, err := a.FindBandwidthLimitRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"GetBandwidthLimitRecord", "bandwidth_limit_rules", qosGetCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.BandwidthLimitGetOption) error {
				_, err := a.GetBandwidthLimitRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"ListBandwidthLimitRecords", "bandwidth_limit_rules", qosListCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.BandwidthLimitListOption) error {
				return qosListErr(a.ListBandwidthLimitRecords(ctx, "parent", o...))
			})
		}},
		{"AllBandwidthLimitRecords", "bandwidth_limit_rules", qosListCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.BandwidthLimitListOption) error {
				_, err := a.AllBandwidthLimitRecords(ctx, "parent", o...)
				return err
			})
		}},
		{"ProxyBandwidthLimitRecords", "bandwidth_limit_rules", qosListCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.BandwidthLimitListOption) error {
				_, err := a.ProxyBandwidthLimitRecords(ctx, "parent", o...)
				return err
			})
		}},
		{"DeleteBandwidthLimitRecord", "bandwidth_limit_rules", qosDeleteCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.QoSRuleDeleteOption) error {
				_, err := a.DeleteBandwidthLimitRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"FindDSCPMarkingRecord", "dscp_marking_rules", qosFindCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.DSCPMarkingFindOption) error {
				_, err := a.FindDSCPMarkingRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"GetDSCPMarkingRecord", "dscp_marking_rules", qosGetCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.DSCPMarkingGetOption) error {
				_, err := a.GetDSCPMarkingRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"ListDSCPMarkingRecords", "dscp_marking_rules", qosListCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.DSCPMarkingListOption) error {
				return qosListErr(a.ListDSCPMarkingRecords(ctx, "parent", o...))
			})
		}},
		{"AllDSCPMarkingRecords", "dscp_marking_rules", qosListCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.DSCPMarkingListOption) error {
				_, err := a.AllDSCPMarkingRecords(ctx, "parent", o...)
				return err
			})
		}},
		{"ProxyDSCPMarkingRecords", "dscp_marking_rules", qosListCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.DSCPMarkingListOption) error {
				_, err := a.ProxyDSCPMarkingRecords(ctx, "parent", o...)
				return err
			})
		}},
		{"DeleteDSCPMarkingRecord", "dscp_marking_rules", qosDeleteCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.QoSRuleDeleteOption) error {
				_, err := a.DeleteDSCPMarkingRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"FindMinimumBandwidthRecord", "minimum_bandwidth_rules", qosFindCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.MinimumBandwidthFindOption) error {
				_, err := a.FindMinimumBandwidthRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"GetMinimumBandwidthRecord", "minimum_bandwidth_rules", qosGetCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.MinimumBandwidthGetOption) error {
				_, err := a.GetMinimumBandwidthRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"ListMinimumBandwidthRecords", "minimum_bandwidth_rules", qosListCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.MinimumBandwidthListOption) error {
				return qosListErr(a.ListMinimumBandwidthRecords(ctx, "parent", o...))
			})
		}},
		{"AllMinimumBandwidthRecords", "minimum_bandwidth_rules", qosListCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.MinimumBandwidthListOption) error {
				_, err := a.AllMinimumBandwidthRecords(ctx, "parent", o...)
				return err
			})
		}},
		{"ProxyMinimumBandwidthRecords", "minimum_bandwidth_rules", qosListCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.MinimumBandwidthListOption) error {
				_, err := a.ProxyMinimumBandwidthRecords(ctx, "parent", o...)
				return err
			})
		}},
		{"DeleteMinimumBandwidthRecord", "minimum_bandwidth_rules", qosDeleteCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.QoSRuleDeleteOption) error {
				_, err := a.DeleteMinimumBandwidthRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"FindMinimumPacketRateRecord", "minimum_packet_rate_rules", qosFindCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.MinimumPacketRateFindOption) error {
				_, err := a.FindMinimumPacketRateRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"GetMinimumPacketRateRecord", "minimum_packet_rate_rules", qosGetCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.MinimumPacketRateGetOption) error {
				_, err := a.GetMinimumPacketRateRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"ListMinimumPacketRateRecords", "minimum_packet_rate_rules", qosRateLists, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.MinimumPacketRateListOption) error {
				return qosListErr(a.ListMinimumPacketRateRecords(ctx, "parent", o...))
			})
		}},
		{"AllMinimumPacketRateRecords", "minimum_packet_rate_rules", qosRateLists, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.MinimumPacketRateListOption) error {
				_, err := a.AllMinimumPacketRateRecords(ctx, "parent", o...)
				return err
			})
		}},
		{"CreateMinimumPacketRateRecord", "minimum_packet_rate_rules", qosWriteCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.PacketRateRuleMutationOption) error {
				_, err := a.CreateMinimumPacketRateRecord(ctx, "parent", o...)
				return err
			})
		}},
		{"UpdateMinimumPacketRateRecord", "minimum_packet_rate_rules", qosWriteCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.PacketRateRuleMutationOption) error {
				_, err := a.UpdateMinimumPacketRateRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"DeleteMinimumPacketRateRecord", "minimum_packet_rate_rules", qosDeleteCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.PacketRateRuleDeleteOption) error {
				_, err := a.DeleteMinimumPacketRateRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"FindPacketRateLimitRecord", "packet_rate_limit_rules", qosFindCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.PacketRateLimitFindOption) error {
				_, err := a.FindPacketRateLimitRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"GetPacketRateLimitRecord", "packet_rate_limit_rules", qosGetCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.PacketRateLimitGetOption) error {
				_, err := a.GetPacketRateLimitRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"ListPacketRateLimitRecords", "packet_rate_limit_rules", qosRateLists, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.PacketRateLimitListOption) error {
				return qosListErr(a.ListPacketRateLimitRecords(ctx, "parent", o...))
			})
		}},
		{"AllPacketRateLimitRecords", "packet_rate_limit_rules", qosRateLists, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.PacketRateLimitListOption) error {
				_, err := a.AllPacketRateLimitRecords(ctx, "parent", o...)
				return err
			})
		}},
		{"CreatePacketRateLimitRecord", "packet_rate_limit_rules", qosWriteCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.PacketRateRuleMutationOption) error {
				_, err := a.CreatePacketRateLimitRecord(ctx, "parent", o...)
				return err
			})
		}},
		{"UpdatePacketRateLimitRecord", "packet_rate_limit_rules", qosWriteCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.PacketRateRuleMutationOption) error {
				_, err := a.UpdatePacketRateLimitRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
		{"DeletePacketRateLimitRecord", "packet_rate_limit_rules", qosDeleteCalls, func(a *rules.API, c []request.Call) (error, bool) {
			return qosRun(c, func(o []rules.PacketRateRuleDeleteOption) error {
				_, err := a.DeletePacketRateLimitRecord(ctx, "parent", "rule", o...)
				return err
			})
		}},
	}
}

// qosCallReply answers every request with a well-formed record or page of the
// addressed rule kind, so accepted calls run to completion.
func qosCallReply(req *http.Request) *http.Response {
	parts := strings.Split(strings.Trim(req.URL.Path, "/"), "/")
	for i, part := range parts {
		if !strings.HasSuffix(part, "_rules") {
			continue
		}
		if req.Method == http.MethodDelete {
			return nativeRuleWire(http.StatusNoContent, "")
		}
		single := strings.TrimSuffix(part, "s")
		if i == len(parts)-1 && req.Method == http.MethodGet {
			if req.URL.Query().Get("marker") == "rule" {
				return nativeRuleWire(http.StatusOK, `{"`+part+`":[]}`)
			}
			return nativeRuleWire(http.StatusOK, `{"`+part+`":[{"id":"rule","name":"consistency"}]}`)
		}
		return nativeRuleWire(http.StatusOK, `{"`+single+`":{"id":"rule","name":"consistency","max_kpps":1}}`)
	}
	return nativeRuleWire(http.StatusNotFound, `{}`)
}

// TestQoSRuleSharedCallSupportIsConsistent pins, per operation, which shared
// calls it implements: each supported kind (and request.Custom over the
// operation's own options struct) completes without error, while each other kind of its
// category and request.Custom over a foreign struct fail with
// resource.ErrInvalidOption before HTTP. The table must match the "Calls:"
// list in the operation's doc comment.
func TestQoSRuleSharedCallSupportIsConsistent(t *testing.T) {
	docs := qosCallDocs(t)
	all := []string{"Header", "Headers", "Microversion", "Query", "IgnoreMissing", "BasePath", "Limit", "Marker", "MaxItems", "Paginated", "Field"}
	for _, op := range qosCallOperations() {
		if got := docs[op.name]; !slices.Equal(got, op.calls) {
			t.Fatalf("%s: doc comment Calls %v, table %v", op.name, got, op.calls)
		}
		inCategory := 0
		for _, kind := range all {
			var calls []nativeRuleCall
			api, _ := nativeRuleAPI(t, &calls, qosCallReply)
			err, ok := op.run(api, []request.Call{qosCallCandidates(op.kind)[kind]})
			if !ok {
				if slices.Contains(op.calls, kind) {
					t.Fatalf("%s: supported %s is outside the option category", op.name, kind)
				}
				continue
			}
			inCategory++
			if slices.Contains(op.calls, kind) {
				if err != nil {
					t.Fatalf("%s: supported %s failed: %v", op.name, kind, err)
				}
				continue
			}
			if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 || !strings.Contains(err.Error(), kind+" call option is unsupported by this operation") {
				t.Fatalf("%s: unsupported %s: %v %v", op.name, kind, err, calls)
			}
		}
		if inCategory < len(op.calls) {
			t.Fatalf("%s: only %d kinds in category", op.name, inCategory)
		}
		var calls []nativeRuleCall
		api, _ := nativeRuleAPI(t, &calls, qosCallReply)
		err, _ := op.run(api, []request.Call{request.Custom(func(*struct{ Foreign bool }) error { return nil })})
		if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 0 {
			t.Fatalf("%s: foreign Custom: %v %v", op.name, err, calls)
		}
	}
}

var qosCallsSentence = regexp.MustCompile(`Calls: ([^.]*(?:\.[A-Z][A-Za-z]*[^.]*)*)\.`)
var qosCallName = regexp.MustCompile(`request\.(\w+)`)

// qosCallDocs reads the first "Calls:" sentence of each API method doc.
func qosCallDocs(t *testing.T) map[string][]string {
	t.Helper()
	files, err := filepath.Glob("python_*.go")
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string][]string)
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Doc == nil {
				continue
			}
			text := strings.Join(strings.Fields(fn.Doc.Text()), " ")
			match := qosCallsSentence.FindStringSubmatch(text)
			if match == nil {
				continue
			}
			for _, call := range qosCallName.FindAllStringSubmatch(match[1], -1) {
				out[fn.Name.Name] = append(out[fn.Name.Name], call[1])
			}
		}
	}
	return out
}

// TestQoSRuleCustomCallsRunOverOwnOptions shows request.Custom over the
// exported options struct runs at its position among With* options and
// shared calls, with the same snapshot rules as the With* functions.
func TestQoSRuleCustomCallsRunOverOwnOptions(t *testing.T) {
	var calls []nativeRuleCall
	var seen http.Header
	api, _ := nativeRuleAPI(t, &calls, func(req *http.Request) *http.Response {
		seen = req.Header.Clone()
		return qosCallReply(req)
	})
	var retained *rules.DSCPMarkingListOpts
	_, err := api.AllDSCPMarkingRecords(context.Background(), "parent",
		request.Header("X-First", "shared"),
		rules.WithDSCPMarkingListHeaders(map[string]string{"X-Second": "legacy"}),
		request.Custom(func(o *rules.DSCPMarkingListOpts) error {
			retained = o
			if o.Headers["X-Second"] != "legacy" || o.Headers["X-First"] != "" {
				t.Fatal("custom ran out of order", o.Headers)
			}
			o.Headers["X-Third"] = "custom"
			return nil
		}),
		request.Header("X-Fourth", "shared"))
	if err != nil || len(calls) != 1 || seen.Get("X-Second") != "legacy" || seen.Get("X-Third") != "custom" || seen.Get("X-Fourth") != "shared" || seen.Get("X-First") != "" {
		t.Fatal(err, calls, seen)
	}
	retained.Headers["X-Third"] = "escaped"
	cause := errors.New("custom failed")
	_, err = api.AllDSCPMarkingRecords(context.Background(), "parent", request.Custom(func(*rules.DSCPMarkingListOpts) error { return cause }))
	if !errors.Is(err, cause) || len(calls) != 1 {
		t.Fatal(err, calls)
	}
	_, err = api.AllDSCPMarkingRecords(context.Background(), "parent", request.Custom[rules.DSCPMarkingListOpts](nil))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
		t.Fatal(err, calls)
	}
	// A Custom over another operation's options of the same family is foreign.
	_, err = api.AllDSCPMarkingRecords(context.Background(), "parent", request.Custom(func(*rules.DSCPMarkingFindOpts) error { return nil }))
	if !errors.Is(err, resource.ErrInvalidOption) || len(calls) != 1 {
		t.Fatal(err, calls)
	}
}

// TestQoSRuleSharedAndLegacyCallsSendSameRequests shows a deprecated With*
// option and its shared replacement produce the same HTTP request.
func TestQoSRuleSharedAndLegacyCallsSendSameRequests(t *testing.T) {
	capture := func(run func(*rules.API) error) (nativeRuleCall, http.Header) {
		var calls []nativeRuleCall
		var seen http.Header
		api, _ := nativeRuleAPI(t, &calls, func(req *http.Request) *http.Response {
			seen = req.Header.Clone()
			return qosCallReply(req)
		})
		if err := run(api); err != nil || len(calls) != 1 {
			t.Fatal(err, calls)
		}
		seen.Del("X-Auth-Token")
		return calls[0], seen
	}
	ctx := context.Background()
	for _, tc := range []struct {
		name           string
		legacy, shared func(*rules.API) error
	}{
		{"list", func(a *rules.API) error {
			_, err := a.AllBandwidthLimitRecords(ctx, "parent", rules.WithBandwidthLimitListLimit(2), rules.WithBandwidthLimitListMarker("m"), rules.WithBandwidthLimitListQuery("unknown", 1), rules.WithBandwidthLimitListMicroversion("2.1"), rules.WithBandwidthLimitListHeaders(map[string]string{"X-Trace": "t"}), rules.WithBandwidthLimitListPaginated(false))
			return err
		}, func(a *rules.API) error {
			_, err := a.AllBandwidthLimitRecords(ctx, "parent", request.Limit(2), request.Marker("m"), request.Query("unknown", 1), request.Microversion("2.1"), request.Headers(map[string]string{"X-Trace": "t"}), request.Paginated(false))
			return err
		}},
		{"get", func(a *rules.API) error {
			_, err := a.GetMinimumBandwidthRecord(ctx, "parent", "rule", rules.WithMinimumBandwidthGetMicroversion("2.1"), rules.WithMinimumBandwidthGetHeaders(map[string]string{"X-Trace": "t"}))
			return err
		}, func(a *rules.API) error {
			_, err := a.GetMinimumBandwidthRecord(ctx, "parent", "rule", request.Microversion("2.1"), request.Headers(map[string]string{"X-Trace": "t"}))
			return err
		}},
		{"delete", func(a *rules.API) error {
			_, err := a.DeletePacketRateLimitRecord(ctx, "parent", "rule", rules.WithPacketRateRuleDeleteIgnoreMissing(false), rules.WithPacketRateRuleDeleteHeaders(map[string]string{"X-Trace": "t"}))
			return err
		}, func(a *rules.API) error {
			_, err := a.DeletePacketRateLimitRecord(ctx, "parent", "rule", request.IgnoreMissing(false), request.Headers(map[string]string{"X-Trace": "t"}))
			return err
		}},
		{"create", func(a *rules.API) error {
			_, err := a.CreatePacketRateLimitRecord(ctx, "parent", rules.WithPacketRateRuleField("max_kpps", 5), rules.WithPacketRateRuleMicroversion("2.1"))
			return err
		}, func(a *rules.API) error {
			_, err := a.CreatePacketRateLimitRecord(ctx, "parent", request.Field("max_kpps", 5), request.Microversion("2.1"))
			return err
		}},
	} {
		legacyCall, legacyHeader := capture(tc.legacy)
		sharedCall, sharedHeader := capture(tc.shared)
		if legacyCall != sharedCall || !equalHeaders(legacyHeader, sharedHeader) {
			t.Fatalf("%s: legacy %+v %v, shared %+v %v", tc.name, legacyCall, legacyHeader, sharedCall, sharedHeader)
		}
	}
}

func equalHeaders(a, b http.Header) bool {
	if len(a) != len(b) {
		return false
	}
	for key, values := range a {
		if !slices.Equal(values, b[key]) {
			return false
		}
	}
	return true
}

// TestQoSRuleSharedQueryMarshalsWhenConstructed documents the one difference
// between request.Query and the deprecated list Query options: the shared
// call marshals its value when constructed, while WithXListQuery discards an
// unknown kwarg before evaluating it. Neither reports the discarded failure.
func TestQoSRuleSharedQueryMarshalsWhenConstructed(t *testing.T) {
	marshaled := 0
	value := qosCountingValue{count: &marshaled}
	legacy := rules.WithDSCPMarkingListQuery("unknown", value)
	if marshaled != 0 {
		t.Fatal("legacy list query marshaled a discarded kwarg", marshaled)
	}
	shared := request.Query("unknown", value)
	if marshaled != 1 {
		t.Fatal("shared query did not marshal when constructed", marshaled)
	}
	var calls []nativeRuleCall
	api, _ := nativeRuleAPI(t, &calls, qosCallReply)
	for _, option := range []rules.DSCPMarkingListOption{legacy, shared} {
		if _, err := api.AllDSCPMarkingRecords(context.Background(), "parent", option); err != nil {
			t.Fatal(err)
		}
	}
	if marshaled != 1 || len(calls) != 2 || calls[0] != calls[1] {
		t.Fatal(marshaled, calls)
	}
}

type qosCountingValue struct{ count *int }

func (v qosCountingValue) MarshalJSON() ([]byte, error) {
	*v.count++
	return json.Marshal("counted")
}
