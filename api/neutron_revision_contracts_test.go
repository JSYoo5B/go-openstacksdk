package api_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/JSYoo5B/gophercloudsdk/internal/testcloud"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/layer3/floatingips"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/layer3/routers"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/qos/policies"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/security/groups"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/subnetpools"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/extensions/trunks"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/networks"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/ports"
	"github.com/JSYoo5B/gophercloudsdk/network/v2/subnets"
	"github.com/JSYoo5B/gophercloudsdk/request"
	"github.com/JSYoo5B/gophercloudsdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

const neutronRevisionBase = "/reverse/v2.0/"

type neutronRevisionInput struct {
	revision    *int
	description string
	replacement *neutronRevisionInput
	invalid     string
}

type neutronRevisionFixture struct {
	name, path, envelope string
	accepts201           bool
	update               func(context.Context, *gophercloud.ServiceClient, neutronRevisionInput) (string, error)
}

// The shared adapter calls each actual public API.Update with concrete native
// opts. Generic request options also exercise the existing capability boundary.
func neutronRevisionCall[O, R any](ctx context.Context, opts O, replacement *O, invalid string,
	call func(context.Context, string, O, ...request.Option[O]) (*R, error)) (*R, error) {
	options := []request.Option[O]{request.WithField[O]("vendor_mode", map[string]any{"enabled": false, "label": "space: %2F?# 한글"})}
	if replacement != nil {
		options = append(options, request.WithOptions(*replacement))
	}
	switch invalid {
	case "query":
		options = append(options, request.WithQuery[O]("vendor", "query"))
	case "header":
		options = append(options, request.WithHeader[O]("X-Vendor", "header"))
	case "nil":
		options = append(options, nil)
	}
	return call(ctx, "target", opts, options...)
}

func neutronRevisionFixtures() []neutronRevisionFixture {
	return []neutronRevisionFixture{
		{name: "networks", path: "networks", envelope: "network", accepts201: true, update: func(ctx context.Context, c *gophercloud.ServiceClient, in neutronRevisionInput) (string, error) {
			opts := networks.UpdateOpts{Description: &in.description, RevisionNumber: in.revision}
			var replacement *networks.UpdateOpts
			if in.replacement != nil {
				replacement = &networks.UpdateOpts{Description: &in.replacement.description, RevisionNumber: in.replacement.revision}
			}
			value, err := neutronRevisionCall(ctx, opts, replacement, in.invalid, networks.New(c).Update)
			if value == nil {
				return "", err
			}
			return value.ID, err
		}},
		{name: "ports", path: "ports", envelope: "port", accepts201: true, update: func(ctx context.Context, c *gophercloud.ServiceClient, in neutronRevisionInput) (string, error) {
			opts := ports.UpdateOpts{Description: &in.description, RevisionNumber: in.revision}
			var replacement *ports.UpdateOpts
			if in.replacement != nil {
				replacement = &ports.UpdateOpts{Description: &in.replacement.description, RevisionNumber: in.replacement.revision}
			}
			value, err := neutronRevisionCall(ctx, opts, replacement, in.invalid, ports.New(c).Update)
			if value == nil {
				return "", err
			}
			return value.ID, err
		}},
		{name: "subnets", path: "subnets", envelope: "subnet", accepts201: true, update: func(ctx context.Context, c *gophercloud.ServiceClient, in neutronRevisionInput) (string, error) {
			opts := subnets.UpdateOpts{Description: &in.description, RevisionNumber: in.revision}
			var replacement *subnets.UpdateOpts
			if in.replacement != nil {
				replacement = &subnets.UpdateOpts{Description: &in.replacement.description, RevisionNumber: in.replacement.revision}
			}
			value, err := neutronRevisionCall(ctx, opts, replacement, in.invalid, subnets.New(c).Update)
			if value == nil {
				return "", err
			}
			return value.ID, err
		}},
		{name: "trunks", path: "trunks", envelope: "trunk", accepts201: false, update: func(ctx context.Context, c *gophercloud.ServiceClient, in neutronRevisionInput) (string, error) {
			opts := trunks.UpdateOpts{Description: &in.description, RevisionNumber: in.revision}
			var replacement *trunks.UpdateOpts
			if in.replacement != nil {
				replacement = &trunks.UpdateOpts{Description: &in.replacement.description, RevisionNumber: in.replacement.revision}
			}
			value, err := neutronRevisionCall(ctx, opts, replacement, in.invalid, trunks.New(c).Update)
			if value == nil {
				return "", err
			}
			return value.ID, err
		}},
		{name: "subnetpools", path: "subnetpools", envelope: "subnetpool", accepts201: false, update: func(ctx context.Context, c *gophercloud.ServiceClient, in neutronRevisionInput) (string, error) {
			opts := subnetpools.UpdateOpts{Description: &in.description, RevisionNumber: in.revision}
			var replacement *subnetpools.UpdateOpts
			if in.replacement != nil {
				replacement = &subnetpools.UpdateOpts{Description: &in.replacement.description, RevisionNumber: in.replacement.revision}
			}
			value, err := neutronRevisionCall(ctx, opts, replacement, in.invalid, subnetpools.New(c).Update)
			if value == nil {
				return "", err
			}
			return value.ID, err
		}},
		{name: "routers", path: "routers", envelope: "router", accepts201: false, update: func(ctx context.Context, c *gophercloud.ServiceClient, in neutronRevisionInput) (string, error) {
			opts := routers.UpdateOpts{Description: &in.description, RevisionNumber: in.revision}
			var replacement *routers.UpdateOpts
			if in.replacement != nil {
				replacement = &routers.UpdateOpts{Description: &in.replacement.description, RevisionNumber: in.replacement.revision}
			}
			value, err := neutronRevisionCall(ctx, opts, replacement, in.invalid, routers.New(c).Update)
			if value == nil {
				return "", err
			}
			return value.ID, err
		}},
		{name: "floatingips", path: "floatingips", envelope: "floatingip", accepts201: false, update: func(ctx context.Context, c *gophercloud.ServiceClient, in neutronRevisionInput) (string, error) {
			opts := floatingips.UpdateOpts{Description: &in.description, RevisionNumber: in.revision}
			var replacement *floatingips.UpdateOpts
			if in.replacement != nil {
				replacement = &floatingips.UpdateOpts{Description: &in.replacement.description, RevisionNumber: in.replacement.revision}
			}
			value, err := neutronRevisionCall(ctx, opts, replacement, in.invalid, floatingips.New(c).Update)
			if value == nil {
				return "", err
			}
			return value.ID, err
		}},
		{name: "policies", path: "qos/policies", envelope: "policy", accepts201: false, update: func(ctx context.Context, c *gophercloud.ServiceClient, in neutronRevisionInput) (string, error) {
			opts := policies.UpdateOpts{Description: &in.description, RevisionNumber: in.revision}
			var replacement *policies.UpdateOpts
			if in.replacement != nil {
				replacement = &policies.UpdateOpts{Description: &in.replacement.description, RevisionNumber: in.replacement.revision}
			}
			value, err := neutronRevisionCall(ctx, opts, replacement, in.invalid, policies.New(c).Update)
			if value == nil {
				return "", err
			}
			return value.ID, err
		}},
		{name: "groups", path: "security-groups", envelope: "security_group", accepts201: false, update: func(ctx context.Context, c *gophercloud.ServiceClient, in neutronRevisionInput) (string, error) {
			opts := groups.UpdateOpts{Description: &in.description, RevisionNumber: in.revision}
			var replacement *groups.UpdateOpts
			if in.replacement != nil {
				replacement = &groups.UpdateOpts{Description: &in.replacement.description, RevisionNumber: in.replacement.revision}
			}
			value, err := neutronRevisionCall(ctx, opts, replacement, in.invalid, groups.New(c).Update)
			if value == nil {
				return "", err
			}
			return value.ID, err
		}},
	}
}

func neutronRevisionClient(cloud *testcloud.Cloud) *gophercloud.ServiceClient {
	client := cloud.Client("network", "/unused-catalog")
	client.ResourceBase = cloud.Server.URL + neutronRevisionBase
	client.MoreHeaders = map[string]string{"X-Source": "ordinary-source"}
	return client
}

func neutronRevisionWire(t *testing.T, r *http.Request, f neutronRevisionFixture, condition, description, token string) {
	t.Helper()
	if r.Method != http.MethodPut || r.URL.Path != neutronRevisionBase+f.path+"/target" || r.URL.RawQuery != "" {
		t.Errorf("native route: %s %s", r.Method, r.URL)
	}
	if got := r.Header.Values("If-Match"); (condition == "" && len(got) != 0) || (condition != "" && (len(got) != 1 || got[0] != condition)) {
		t.Errorf("If-Match=%q want %q", got, condition)
	}
	if r.Header.Get("X-Auth-Token") != token || r.Header.Get("X-Source") != "ordinary-source" || r.Header.Get("Accept") != "application/json" || r.Header.Get("Content-Type") != "application/json" {
		t.Errorf("headers=%v", r.Header)
	}
	var root map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&root); err != nil {
		t.Errorf("body: %v", err)
		return
	}
	if len(root) != 1 {
		t.Errorf("root envelope=%v", root)
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(root[f.envelope], &body); err != nil {
		t.Errorf("envelope %q: %v", f.envelope, err)
		return
	}
	var got string
	if err := json.Unmarshal(body["description"], &got); err != nil || got != description {
		t.Errorf("description=%q err=%v", got, err)
	}
	if _, exists := body["revision_number"]; exists {
		t.Error("revision condition leaked into JSON body")
	}
	var extension map[string]any
	if err := json.Unmarshal(body["vendor_mode"], &extension); err != nil || extension["enabled"] != false || extension["label"] != "space: %2F?# 한글" {
		t.Errorf("extension=%v err=%v", extension, err)
	}
}

func neutronRevisionResponse(w http.ResponseWriter, f neutronRevisionFixture, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{f.envelope: map[string]any{"id": "actual-" + f.name, "description": "server-value", "revision_number": 9, "ip_version": 4, "default_prefixlen": 24, "min_prefixlen": 8, "max_prefixlen": 32}})
}

func TestNeutronRevisionContractsAllNativeUpdates(t *testing.T) {
	for _, f := range neutronRevisionFixtures() {
		t.Run(f.name, func(t *testing.T) {
			for _, condition := range []struct {
				name     string
				revision *int
				want     string
			}{
				{name: "omitted"},
				{name: "zero", revision: func() *int { v := 0; return &v }(), want: "revision_number=0"},
				{name: "positive", revision: func() *int { v := 19; return &v }(), want: "revision_number=19"},
				{name: "signed-native-value", revision: func() *int { v := -3; return &v }(), want: "revision_number=-3"},
			} {
				t.Run(condition.name, func(t *testing.T) {
					cloud := testcloud.New(t)
					client := neutronRevisionClient(cloud)
					var calls atomic.Int32
					cloud.Mux.HandleFunc(neutronRevisionBase+f.path+"/target", func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						neutronRevisionWire(t, r, f, condition.want, "literal update", "test-token")
						neutronRevisionResponse(w, f, 200)
					})
					id, err := f.update(context.Background(), client, neutronRevisionInput{revision: condition.revision, description: "literal update"})
					if err != nil || id != "actual-"+f.name || calls.Load() != 1 {
						t.Fatalf("id=%q calls=%d err=%v", id, calls.Load(), err)
					}
					if client.Endpoint != cloud.Server.URL+"/unused-catalog/" || client.ResourceBase != cloud.Server.URL+neutronRevisionBase || client.MoreHeaders["X-Source"] != "ordinary-source" || len(client.MoreHeaders) != 1 {
						t.Fatal("shared source client changed")
					}
				})
			}
		})
	}
}

func TestNeutronRevisionContractsReplacementChoosesBodyAndCondition(t *testing.T) {
	for _, f := range neutronRevisionFixtures() {
		t.Run(f.name, func(t *testing.T) {
			zero, positive := 0, 27
			for _, replacement := range []struct {
				name     string
				revision *int
				want     string
			}{
				{name: "omitted"}, {name: "zero", revision: &zero, want: "revision_number=0"}, {name: "positive", revision: &positive, want: "revision_number=27"},
			} {
				t.Run(replacement.name, func(t *testing.T) {
					cloud := testcloud.New(t)
					var calls atomic.Int32
					cloud.Mux.HandleFunc(neutronRevisionBase+f.path+"/target", func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						neutronRevisionWire(t, r, f, replacement.want, "replacement", "test-token")
						neutronRevisionResponse(w, f, 200)
					})
					first := 11
					id, err := f.update(context.Background(), neutronRevisionClient(cloud), neutronRevisionInput{revision: &first, description: "initial", replacement: &neutronRevisionInput{revision: replacement.revision, description: "replacement"}})
					if err != nil || id != "actual-"+f.name || calls.Load() != 1 || first != 11 || zero != 0 || positive != 27 {
						t.Fatalf("id=%q calls=%d values=%d/%d/%d err=%v", id, calls.Load(), first, zero, positive, err)
					}
				})
			}
		})
	}
}

func TestNeutronRevisionContractsPreflightRetainsExtensionCapabilities(t *testing.T) {
	for _, f := range neutronRevisionFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			var calls atomic.Int32
			cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				t.Error("preflight failure sent HTTP")
				neutronRevisionResponse(w, f, 200)
			})
			for _, invalid := range []string{"query", "header", "nil"} {
				t.Run(invalid, func(t *testing.T) {
					revision := 4
					id, err := f.update(context.Background(), neutronRevisionClient(cloud), neutronRevisionInput{revision: &revision, description: "valid", invalid: invalid})
					var operation *resource.OperationError
					if id != "" || !errors.Is(err, resource.ErrInvalidOption) || !errors.As(err, &operation) || operation.Operation != "Update" || calls.Load() != 0 {
						t.Fatalf("id=%q calls=%d err=%v", id, calls.Load(), err)
					}
				})
			}
		})
	}
}

func TestNeutronRevisionContractsNativeStatusesAndConflict(t *testing.T) {
	for _, f := range neutronRevisionFixtures() {
		t.Run(f.name, func(t *testing.T) {
			for _, status := range []int{201, 412} {
				t.Run(fmt.Sprint(status), func(t *testing.T) {
					cloud := testcloud.New(t)
					var calls atomic.Int32
					cloud.Mux.HandleFunc(neutronRevisionBase+f.path+"/target", func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						neutronRevisionWire(t, r, f, "revision_number=8", "conditional change", "test-token")
						if status == 412 {
							testcloud.JSON(w, status, `{"NeutronError":{"type":"PreconditionFailed","message":"revision changed"}}`)
							return
						}
						neutronRevisionResponse(w, f, status)
					})
					revision := 8
					id, err := f.update(context.Background(), neutronRevisionClient(cloud), neutronRevisionInput{revision: &revision, description: "conditional change"})
					if status == 201 && f.accepts201 {
						if err != nil || id != "actual-"+f.name {
							t.Fatalf("accepted native201 id=%q err=%v", id, err)
						}
					} else {
						var operation *resource.OperationError
						var native gophercloud.ErrUnexpectedResponseCode
						if id != "" || !errors.As(err, &operation) || operation.Operation != "Update" || !errors.As(err, &native) || native.Actual != status || !gophercloud.ResponseCodeIs(err, status) {
							t.Fatalf("native status=%d id=%q err=%v", status, id, err)
						}
						if status == 412 && len(native.Body) == 0 {
							t.Error("conflict lost native response body")
						}
					}
					if calls.Load() != 1 {
						t.Fatalf("native status triggered SDK fallback: calls=%d", calls.Load())
					}
				})
			}
		})
	}
}

func TestNeutronRevisionContractsNativeReplayKeepsSerializedCondition(t *testing.T) {
	for _, f := range neutronRevisionFixtures() {
		t.Run(f.name, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := neutronRevisionClient(cloud)
			var attempts, reauths, backoffs, retries atomic.Int32
			revision := 6
			cloud.Provider.ReauthFunc = func(context.Context) error {
				reauths.Add(1)
				revision = 999
				cloud.Provider.SetToken("refreshed-token")
				return nil
			}
			cloud.Provider.RetryBackoffFunc = func(_ context.Context, code *gophercloud.ErrUnexpectedResponseCode, _ error, count uint) error {
				backoffs.Add(1)
				if code.Actual != 429 || count != 1 {
					t.Errorf("backoff status=%d count=%d", code.Actual, count)
				}
				return nil
			}
			cloud.Provider.RetryFunc = func(_ context.Context, method, target string, options *gophercloud.RequestOpts, err error, count uint) error {
				retries.Add(1)
				if !gophercloud.ResponseCodeIs(err, 503) {
					return err
				}
				if method != http.MethodPut || target != cloud.Server.URL+neutronRevisionBase+f.path+"/target" || count != 2 || options.MoreHeaders["If-Match"] != "revision_number=6" {
					t.Errorf("retry method=%s target=%s count=%d headers=%v", method, target, count, options.MoreHeaders)
				}
				options.MoreHeaders["X-Native-Retry"] = "preserved"
				return nil
			}
			cloud.Mux.HandleFunc(neutronRevisionBase+f.path+"/target", func(w http.ResponseWriter, r *http.Request) {
				n := attempts.Add(1)
				token := "refreshed-token"
				if n == 1 {
					token = "test-token"
				}
				neutronRevisionWire(t, r, f, "revision_number=6", "same prepared body", token)
				switch n {
				case 1:
					testcloud.JSON(w, 401, `{"error":"expired"}`)
				case 2:
					testcloud.JSON(w, 429, `{"error":"rate-limited"}`)
				case 3:
					testcloud.JSON(w, 503, `{"error":"retry"}`)
				default:
					if r.Header.Get("X-Native-Retry") != "preserved" {
						t.Error("native retry options lost")
					}
					neutronRevisionResponse(w, f, 200)
				}
			})
			id, err := f.update(context.Background(), client, neutronRevisionInput{revision: &revision, description: "same prepared body"})
			if err != nil || id != "actual-"+f.name || attempts.Load() != 4 || reauths.Load() != 1 || backoffs.Load() != 1 || retries.Load() != 1 || revision != 999 {
				t.Fatalf("id=%q attempts=%d callbacks=%d/%d/%d value=%d err=%v", id, attempts.Load(), reauths.Load(), backoffs.Load(), retries.Load(), revision, err)
			}
			if len(client.MoreHeaders) != 1 || client.ProviderClient != cloud.Provider || cloud.Provider.Token() != "refreshed-token" {
				t.Fatal("source or shared authentication changed")
			}
		})
	}
}

func TestNeutronRevisionContractsRawNativeHeaderEscapePolicy(t *testing.T) {
	f := neutronRevisionFixtures()[0]
	for _, mode := range []string{"source-override", "callback-override", "callback-omit"} {
		t.Run(mode, func(t *testing.T) {
			cloud := testcloud.New(t)
			client := neutronRevisionClient(cloud)
			var attempts, retries atomic.Int32
			if mode == "source-override" {
				client.MoreHeaders["If-Match"] = "raw-source-condition"
			} else {
				cloud.Provider.RetryFunc = func(_ context.Context, _, _ string, options *gophercloud.RequestOpts, err error, _ uint) error {
					retries.Add(1)
					if !gophercloud.ResponseCodeIs(err, 503) {
						return err
					}
					if mode == "callback-override" {
						options.MoreHeaders["If-Match"] = "raw-callback-condition"
					} else {
						options.OmitHeaders = append(options.OmitHeaders, "If-Match")
					}
					return nil
				}
			}
			cloud.Mux.HandleFunc(neutronRevisionBase+f.path+"/target", func(w http.ResponseWriter, r *http.Request) {
				n := attempts.Add(1)
				want := "revision_number=2"
				if mode == "source-override" {
					want = "raw-source-condition"
				} else if n > 1 {
					if mode == "callback-override" {
						want = "raw-callback-condition"
					} else {
						want = ""
					}
				}
				neutronRevisionWire(t, r, f, want, "native raw policy", "test-token")
				if mode != "source-override" && n == 1 {
					testcloud.JSON(w, 503, `{"error":"retry"}`)
					return
				}
				neutronRevisionResponse(w, f, 200)
			})
			revision := 2
			id, err := f.update(context.Background(), client, neutronRevisionInput{revision: &revision, description: "native raw policy"})
			wantCalls, wantRetries := int32(2), int32(1)
			if mode == "source-override" {
				wantCalls, wantRetries = 1, 0
			}
			if err != nil || id != "actual-"+f.name || attempts.Load() != wantCalls || retries.Load() != wantRetries {
				t.Fatalf("id=%q attempts=%d retries=%d err=%v", id, attempts.Load(), retries.Load(), err)
			}
			if mode == "source-override" && client.MoreHeaders["If-Match"] != "raw-source-condition" {
				t.Fatal("raw source override mutated")
			}
		})
	}
}

func TestNeutronRevisionContractsNativeRedirectPolicy(t *testing.T) {
	f := neutronRevisionFixtures()[0]
	for _, follow := range []bool{false, true} {
		t.Run(fmt.Sprint(follow), func(t *testing.T) {
			cloud := testcloud.New(t)
			var attempts, redirects atomic.Int32
			cloud.Provider.HTTPClient.CheckRedirect = func(next *http.Request, _ []*http.Request) error {
				redirects.Add(1)
				if !follow {
					return http.ErrUseLastResponse
				}
				next.Header.Set("If-Match", "raw-redirect-condition")
				return nil
			}
			cloud.Mux.HandleFunc(neutronRevisionBase+f.path+"/target", func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				neutronRevisionWire(t, r, f, "revision_number=3", "redirect body", "test-token")
				w.Header().Set("Location", neutronRevisionBase+"redirected")
				w.WriteHeader(307)
			})
			cloud.Mux.HandleFunc(neutronRevisionBase+"redirected", func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				if !follow {
					t.Error("redirect refusal ignored")
				}
				if r.Method != http.MethodPut || r.Header.Get("If-Match") != "raw-redirect-condition" || r.Header.Get("X-Auth-Token") != "test-token" {
					t.Errorf("redirect native request=%s headers=%v", r.Method, r.Header)
				}
				var root map[string]json.RawMessage
				if err := json.NewDecoder(r.Body).Decode(&root); err != nil || len(root[f.envelope]) == 0 {
					t.Errorf("redirect body=%v err=%v", root, err)
				}
				neutronRevisionResponse(w, f, 200)
			})
			revision := 3
			id, err := f.update(context.Background(), neutronRevisionClient(cloud), neutronRevisionInput{revision: &revision, description: "redirect body"})
			if follow {
				if err != nil || id != "actual-"+f.name || attempts.Load() != 2 {
					t.Fatalf("follow id=%q attempts=%d err=%v", id, attempts.Load(), err)
				}
			} else if id != "" || !gophercloud.ResponseCodeIs(err, 307) || attempts.Load() != 1 {
				t.Fatalf("refusal id=%q attempts=%d err=%v", id, attempts.Load(), err)
			}
			if redirects.Load() != 1 {
				t.Fatalf("redirect callback=%d", redirects.Load())
			}
		})
	}
}
