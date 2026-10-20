package quotafetch_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/quotafetch"
	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
)

func quotaProfile() quotafetch.Profile {
	return quotafetch.Profile{Role: "volume", IntegerFields: []string{"volumes", "gigabytes"}}
}

type quotaTransport func(*http.Request) (*http.Response, error)

func (f quotaTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
func quotaClient(t *testing.T, reply quotaTransport) *gophercloud.ServiceClient {
	t.Helper()
	cloud := testcloud.New(t)
	cloud.Provider.HTTPClient.Transport = reply
	return cloud.Client("volumev3", "/prefix/v3/")
}
func quotaReply(code int, raw string) *http.Response {
	return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(raw)), Header: http.Header{"X-Request-Id": {"actual-quota"}}}
}
func assertQuotaProof(t *testing.T, err error, code int, raw string) {
	t.Helper()
	var proof *resource.ResponseError
	if !errors.As(err, &proof) || proof.StatusCode != code || string(proof.Body) != raw || proof.Header.Get("X-Request-Id") != "actual-quota" || proof.Cause == nil || !errors.Is(err, proof.Cause) {
		t.Fatal(err, proof)
	}
}

func TestQuotaRecordNormalizesUsageReservationsAndKnownDescriptors(t *testing.T) {
	raw := `{"quota_set":{"id":"foreign","volumes":{"limit":"12","in_use":2,"reserved":null,"allocated":9},"gigabytes":false,"extension":{"limit":9007199254740993,"reserved":4},"ignored":{"allocated":8},"self":"gone","location":{"spoof":true},"microversion":"foreign"}}`
	client := quotaClient(t, func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" || req.URL.EscapedPath() != "/prefix/v3/os-quota-sets/folder%2Fid%25" || req.URL.RawQuery != "usage=False" || req.Body != nil {
			t.Fatal(req.Method, req.URL, req.Body)
		}
		return quotaReply(203, raw), nil
	})
	result, err := quotafetch.Read(context.Background(), client, "folder/id%", "", quotaProfile())
	if err != nil || result.StatusCode != 203 || result.ProjectID != "folder/id%" || string(result.Envelope) != raw {
		t.Fatal(result, err)
	}
	for key, want := range map[string]string{"id": `"foreign"`, "volumes": "12", "gigabytes": "false", "extension": "9007199254740993", "usage": `{"volumes":2}`, "reservation": `{"extension":4,"volumes":null}`, "location": "null", "microversion": "null", "name": "null"} {
		if string(result.Resource.Body[key]) != want {
			t.Fatal(key, want, result.Resource.Body)
		}
	}
	if result.Resource.Body["ignored"] != nil || result.Resource.Body["self"] != nil || string(result.Wire.Body["volumes"]) != `{"limit":"12","in_use":2,"reserved":null,"allocated":9}` || string(result.Wire.Body["self"]) != `"gone"` {
		t.Fatal(result)
	}
	result.Resource.Body["id"][1] = 'X'
	result.Header.Set("X-Request-Id", "changed")
	result.Envelope[0] = 'X'
	if string(result.Wire.Body["id"]) != `"foreign"` || result.Resource.Header.Get("X-Request-Id") != "actual-quota" || result.Wire.Header.Get("X-Request-Id") != "actual-quota" {
		t.Fatal(result)
	}
}
func TestQuotaIntegerDescriptorUsesSourceConversionAndExactNumbers(t *testing.T) {
	for _, tc := range []struct {
		raw, want string
		fail      bool
	}{{`"١٢"`, "12", false}, {`"-1"`, "0", false}, {`" 12 "`, "0", false}, {`""`, "0", false}, {`true`, "true", false}, {`12.99`, "12", false}, {`1e-400`, "0", false}, {`9007199254740993`, "9007199254740993", false}, {`[]`, "0", false}, {`null`, "null", false}, {`"²"`, "", true}, {`1e400`, "", true}} {
		t.Run(tc.raw, func(t *testing.T) {
			raw := `{"quota_set":{"volumes":` + tc.raw + `}}`
			client := quotaClient(t, func(*http.Request) (*http.Response, error) { return quotaReply(200, raw), nil })
			result, err := quotafetch.Read(context.Background(), client, "project", "", quotaProfile())
			if tc.fail {
				if err == nil || result == nil || result.Resource != nil {
					t.Fatal(result, err)
				}
				assertQuotaProof(t, err, 200, raw)
			} else if err != nil || string(result.Resource.Body["volumes"]) != tc.want {
				t.Fatal(result, err)
			}
		})
	}
}
func TestQuotaFreshEmptyNonJSONAndBareObjectResponses(t *testing.T) {
	for _, raw := range []string{"", "not-json", "{broken", `{}`, `{"volumes":7}`, `{"quota_set":{}}`} {
		t.Run(raw, func(t *testing.T) {
			client := quotaClient(t, func(*http.Request) (*http.Response, error) { return quotaReply(204, raw), nil })
			result, err := quotafetch.Read(context.Background(), client, "project", "", quotaProfile())
			if err != nil || result == nil || result.StatusCode != 204 || string(result.Envelope) != raw {
				t.Fatal(result, err)
			}
			parsed := json.Valid([]byte(raw))
			if (result.Wire != nil) != parsed {
				t.Fatal("invented Wire", result)
			}
			want := "null"
			if parsed {
				want = "{}"
			}
			if string(result.Resource.Body["usage"]) != want || string(result.Resource.Body["reservation"]) != want {
				t.Fatal(result.Resource.Body)
			}
		})
	}
}
func TestQuotaWrongShapeAndDictionaryOrderFailuresRetainActualResponse(t *testing.T) {
	for _, raw := range []string{`[]`, `null`, `1`, `{"quota_set":false}`, `{"quota_set":null}`, `{"quota_set":[]}`, `{"quota_set":{"usage":null,"volumes":{"in_use":3}}}`, `{"quota_set":{"reservation":1,"volumes":{"reserved":3}}}`, "{\"quota_set\":{\"volumes\":\"\xff\"}}"} {
		t.Run(raw, func(t *testing.T) {
			client := quotaClient(t, func(*http.Request) (*http.Response, error) { return quotaReply(202, raw), nil })
			result, err := quotafetch.Read(context.Background(), client, "project", "", quotaProfile())
			if err == nil || result == nil || result.Resource != nil || result.StatusCode != 202 {
				t.Fatal(result, err)
			}
			assertQuotaProof(t, err, 202, raw)
		})
	}
	// Overwriting usage after nested extraction is legal, then dict descriptor
	// coercion makes non-object values {}; reversing the order above is an error.
	raw := `{"quota_set":{"volumes":{"in_use":3},"usage":false}}`
	client := quotaClient(t, func(*http.Request) (*http.Response, error) { return quotaReply(200, raw), nil })
	result, err := quotafetch.Read(context.Background(), client, "project", "", quotaProfile())
	if err != nil || string(result.Resource.Body["usage"]) != "{}" {
		t.Fatal(result, err)
	}
}
func TestQuotaQueriesOwnJSONInputsLastOptionsAndFetchControls(t *testing.T) {
	data := []string{"one", "two"}
	option := quotafetch.WithQuery("fields", data)
	data[0] = "late"
	calls := 0
	client := quotaClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		q := req.URL.Query()
		if q.Get("usage") != "False" || strings.Join(q["fields"], ",") != "one,two" || q.Get("flag") != "False" || q.Has("none") || q.Has("skip_cache") || q.Has("resource_response_key") || q.Has("microversion") || req.Header.Get("OpenStack-API-Version") != "volume 3.100" {
			t.Fatal(req.URL, req.Header)
		}
		return quotaReply(200, `{"quota_set":{"volumes":1}}`), nil
	})
	for range 2 {
		result, err := quotafetch.Read(context.Background(), client, "project", "", quotaProfile(), option, quotafetch.WithQuery("flag", false), quotafetch.WithQuery("none", nil), quotafetch.WithQuery("usage", true), quotafetch.WithUsage(false), quotafetch.WithQuery("microversion", "3.100"), quotafetch.WithQuery("skip_cache", true), quotafetch.WithQuery("resource_response_key", map[string]int{"ignored": 1}))
		if err != nil || result.Microversion != "3.100" {
			t.Fatal(result, err)
		}
	}
	if calls != 2 || client.Microversion != "" {
		t.Fatal(calls, client)
	}
}

type quotaExplodingMarshaler struct{ called *int }

func (v quotaExplodingMarshaler) MarshalJSON() ([]byte, error) {
	*v.called++
	return nil, errors.New("caller marshaler")
}
func TestQuotaPreflightRejectsScopeCollisionsNilOptionsAndRoutes(t *testing.T) {
	calls, marshaled := 0, 0
	client := quotaClient(t, func(*http.Request) (*http.Response, error) { calls++; return quotaReply(200, `{}`), nil })
	for _, option := range []quotafetch.Option{nil, quotafetch.WithQuery(" ", 1), quotafetch.WithQuery("a\n", 1), quotafetch.WithQuery("session", quotaExplodingMarshaler{&marshaled}), quotafetch.WithRequiresID(true), quotafetch.WithQuery("microversion", 42), quotafetch.WithQuery("user_id", "other"), quotafetch.WithBasePath("https://foreign.invalid/quotas"), quotafetch.WithBasePath("/quotas?unsafe=true"), quotafetch.WithOptions(quotafetch.Options{Query: map[string]json.RawMessage{"x": json.RawMessage("{broken")}})} {
		_, err := quotafetch.Read(context.Background(), client, "project", "user", quotaProfile(), option)
		if !errors.Is(err, resource.ErrInvalidOption) {
			t.Fatal(err)
		}
	}
	if calls != 0 || marshaled != 0 {
		t.Fatal(calls, marshaled)
	}
}
func TestQuotaBasePathAndErrorMessageStayOwnedAcrossRetry(t *testing.T) {
	calls, retries := 0, 0
	client := quotaClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Path != "/prefix/v3/os-quota-sets/project/defaults" || req.URL.RawQuery != "usage=False" {
			t.Fatal(req.URL)
		}
		return quotaReply(503, `{"error":"busy"}`), nil
	})
	client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
		retries++
		return errors.New("stop")
	}
	_, err := quotafetch.Read(context.Background(), client, "project", "", quotaProfile(), quotafetch.WithBasePath("/os-quota-sets/%(project_id)s/defaults"), quotafetch.WithErrorMessage("quota lookup"))
	if err == nil || !strings.Contains(err.Error(), "quota lookup") || calls != 1 || retries != 1 {
		t.Fatal(err, calls, retries)
	}
}

type quotaFaultBody struct {
	io.Reader
	closeCause error
	onClose    func()
	closed     int
}

func (b *quotaFaultBody) Close() error {
	b.closed++
	if b.onClose != nil {
		b.onClose()
	}
	return b.closeCause
}

type quotaFaultReader func([]byte) (int, error)

func (f quotaFaultReader) Read(p []byte) (int, error) { return f(p) }
func TestQuotaPhysicalFaultsKeepNative401OriginalAndAllCausesWithoutHooks(t *testing.T) {
	for _, code := range []int{200, 203, 401, 404, 429, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			raw := `{"quota_set":{"volumes":1}}`
			readCause, closeCause := errors.New("quota read"), errors.New("quota close")
			calls, retries, reauth := 0, 0, 0
			done := false
			body := &quotaFaultBody{closeCause: closeCause}
			body.Reader = quotaFaultReader(func(p []byte) (int, error) {
				if done {
					return 0, io.EOF
				}
				done = true
				return copy(p, raw), readCause
			})
			var target string
			client := quotaClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				target = req.URL.String()
				reply := quotaReply(code, raw)
				reply.Body = body
				return reply, nil
			})
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			client.ReauthFunc = func(context.Context) error { reauth++; return nil }
			result, err := quotafetch.Read(context.Background(), client, "project", "", quotaProfile())
			if !errors.Is(err, readCause) || !errors.Is(err, closeCause) || calls != 1 || body.closed != 1 || retries != 0 || reauth != 0 {
				t.Fatal(result, err, calls, body.closed, retries, reauth)
			}
			if code < 400 {
				assertQuotaProof(t, err, code, raw)
				if result == nil || result.Resource != nil || result.Wire != nil || result.StatusCode != code {
					t.Fatal(result)
				}
			} else {
				if result != nil {
					t.Fatal(result)
				}
				actual := err
				if code == 401 {
					var auth *gophercloud.ErrUnableToReauthenticate
					if !errors.As(err, &auth) || !errors.Is(auth.ErrReauth, readCause) || !errors.Is(auth.ErrReauth, closeCause) {
						t.Fatal(err, auth)
					}
					actual = auth.ErrOriginal
				}
				var native gophercloud.ErrUnexpectedResponseCode
				if !errors.As(actual, &native) || native.Actual != code || string(native.Body) != raw || native.URL != target || native.Method != "GET" || native.ResponseHeader.Get("X-Request-Id") != "actual-quota" {
					t.Fatal(err, native)
				}
			}
		})
	}
}
func TestQuotaCleanReauthenticationAndRetryKeepTokenTargetAndVersion(t *testing.T) {
	for _, code := range []int{401, 503} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls, retries, reauth := 0, 0, 0
			var client *gophercloud.ServiceClient
			client = quotaClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != "GET" || req.URL.RawQuery != "usage=True&user_id=user" || req.Header.Get("OpenStack-API-Version") != "volume 3.99" || req.Body != nil {
					t.Fatal(req.URL, req.Header, req.Body)
				}
				if calls == 1 {
					return quotaReply(code, `{"error":"renew"}`), nil
				}
				return quotaReply(200, `{"quota_set":{"volumes":7}}`), nil
			})
			client.ReauthFunc = func(context.Context) error { reauth++; client.SetToken("renewed"); return nil }
			client.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				retries++
				return nil
			}
			result, err := quotafetch.Read(context.Background(), client, "project", "user", quotaProfile(), quotafetch.WithUsage(true), quotafetch.WithMicroversion("3.99"))
			if err != nil || calls != 2 || string(result.Resource.Body["volumes"]) != "7" || client.Microversion != "" {
				t.Fatal(result, err, calls, retries, reauth)
			}
			if code == 401 && reauth != 1 || code == 503 && retries != 1 {
				t.Fatal(retries, reauth)
			}
		})
	}
}
func TestQuotaRetryCannotReplaceSelectedVersionOrRemoveIt(t *testing.T) {
	for _, selected := range []string{"", "3.100"} {
		t.Run(selected, func(t *testing.T) {
			calls := 0
			client := quotaClient(t, func(*http.Request) (*http.Response, error) { calls++; return quotaReply(503, `{"error":"busy"}`), nil })
			client.RetryFunc = func(_ context.Context, _, _ string, opts *gophercloud.RequestOpts, _ error, _ uint) error {
				opts.MoreHeaders["OpenStack-API-Version"] = "volume 3.5"
				return nil
			}
			_, err := quotafetch.Read(context.Background(), client, "project", "", quotaProfile(), quotafetch.WithMicroversion(selected))
			if !errors.Is(err, resource.ErrInvalidOption) || calls != 1 {
				t.Fatal(err, calls)
			}
		})
	}
}
func TestQuotaCapturedSourceAndCancellationPreventHTTP(t *testing.T) {
	calls := 0
	client := quotaClient(t, func(*http.Request) (*http.Response, error) { calls++; return quotaReply(200, `{}`), nil })
	changed := func(*quotafetch.Options) error { client.ResourceBase = "https://foreign.invalid/v3/"; return nil }
	_, err := quotafetch.Read(context.Background(), client, "project", "", quotaProfile(), changed)
	if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 {
		t.Fatal(err, calls)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller cancellation")
	cancel(cause)
	_, err = quotafetch.Read(ctx, nil, "project", "", quotaProfile())
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) {
		t.Fatal(err)
	}
}

func TestQuotaLocationUsesCapturedProjectAndIndependentCloudMetadata(t *testing.T) {
	cloud, region, name := "configured", "region-1", "current-project"
	location := resource.CloudLocation{Cloud: &cloud, RegionName: &region, Project: resource.CloudProject{ID: json.RawMessage(`"current"`), Name: &name}}
	option := quotafetch.WithLocation(location)
	cloud = "late"
	location.Project.ID[1] = 'X'
	client := quotaClient(t, func(*http.Request) (*http.Response, error) {
		return quotaReply(200, `{"quota_set":{"project_id":"spoof","availability_zone":"az-1","location":{"fake":true}}}`), nil
	})
	result, err := quotafetch.Read(context.Background(), client, "foreign-project", "", quotaProfile(), option)
	if err != nil || string(result.Resource.Body["project_id"]) != `"foreign-project"` || string(result.Wire.Body["project_id"]) != `"spoof"` {
		t.Fatal(result, err)
	}
	var actual resource.CloudLocation
	if err := json.Unmarshal(result.Resource.Body["location"], &actual); err != nil || actual.Cloud == nil || *actual.Cloud != "configured" || string(actual.Project.ID) != `"foreign-project"` || actual.Project.Name != nil || string(actual.Zone) != `"az-1"` {
		t.Fatal(actual, err)
	}
}

func TestCinderQuotaRawUsageRetainsLiteralRequestsValues(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{{`null`, ""}, {`"False"`, "usage=False"}, {`0`, "usage=0"}, {`[]`, ""}, {`[false,true]`, "usage=False&usage=True"}, {`true`, "usage=True"}} {
		t.Run(tc.raw, func(t *testing.T) {
			calls := 0
			client := quotaClient(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if req.URL.RawQuery != tc.want {
					t.Fatal(req.URL, tc.want)
				}
				return quotaReply(200, `{"quota_set":{}}`), nil
			})
			_, err := quotafetch.Read(context.Background(), client, "project", "", quotaProfile(), quotafetch.WithQuery("usage", json.RawMessage(tc.raw)))
			if err != nil || calls != 1 {
				t.Fatal(err, calls)
			}
		})
	}
}

func TestQuotaOptionsObserveSourceAndCancellationBeforeRestoringCallback(t *testing.T) {
	for _, mode := range []string{"source", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			calls, next := 0, 0
			client := quotaClient(t, func(*http.Request) (*http.Response, error) { calls++; return quotaReply(200, `{}`), nil })
			original := client.ResourceBase
			ctx, cancel := context.WithCancelCause(context.Background())
			defer cancel(nil)
			cause := errors.New("quota callback canceled")
			change := func(*quotafetch.Options) error {
				if mode == "source" {
					client.ResourceBase = "https://foreign.invalid/v3/"
				} else {
					cancel(cause)
				}
				return nil
			}
			restore := func(*quotafetch.Options) error { next++; client.ResourceBase = original; return nil }
			_, err := quotafetch.Read(ctx, client, "project", "", quotaProfile(), change, restore)
			if calls != 0 || next != 0 {
				t.Fatal(err, calls, next)
			}
			if mode == "source" && !errors.Is(err, resource.ErrInvalidOption) || mode == "cancel" && (!errors.Is(err, context.Canceled) || !errors.Is(err, cause)) {
				t.Fatal(err)
			}
		})
	}
}
func TestQuotaEachCallbackBoundaryOwnsOptionsBeforeLaterAliasedMutation(t *testing.T) {
	raw := json.RawMessage(`"original"`)
	shared := map[string]json.RawMessage{"tag": raw}
	calls := 0
	client := quotaClient(t, func(req *http.Request) (*http.Response, error) {
		calls++
		if req.URL.Query().Get("tag") != "original" || req.URL.Query().Has("late") {
			t.Fatal(req.URL)
		}
		return quotaReply(200, `{}`), nil
	})
	first := func(c *quotafetch.Options) error { c.Query = shared; return nil }
	later := func(c *quotafetch.Options) error { shared["late"] = json.RawMessage("true"); raw[1] = 'X'; return nil }
	_, err := quotafetch.Read(context.Background(), client, "project", "", quotaProfile(), first, later)
	if err != nil || calls != 1 {
		t.Fatal(err, calls)
	}
}

func TestQuotaAcceptedCloseCancellationRetainsActualReceiptWithoutResource(t *testing.T) {
	raw := `{"quota_set":{"volumes":7}}`
	cause := errors.New("quota Close canceled")
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	body := &quotaFaultBody{Reader: strings.NewReader(raw), onClose: func() { cancel(cause) }}
	calls := 0
	client := quotaClient(t, func(*http.Request) (*http.Response, error) {
		calls++
		wire := quotaReply(203, raw)
		wire.Body = body
		return wire, nil
	})
	result, err := quotafetch.Read(ctx, client, "project", "", quotaProfile())
	if !errors.Is(err, context.Canceled) || !errors.Is(err, cause) || result == nil || result.Resource != nil || result.StatusCode != 203 || body.closed != 1 || calls != 1 {
		t.Fatal(result, err, body.closed, calls)
	}
	assertQuotaProof(t, err, 203, raw)
}
func TestQuotaProfileDescriptorSliceIsOwnedBeforeCallerOptions(t *testing.T) {
	profile := quotaProfile()
	client := quotaClient(t, func(*http.Request) (*http.Response, error) {
		return quotaReply(200, `{"quota_set":{"volumes":"12","gigabytes":"13"}}`), nil
	})
	option := func(*quotafetch.Options) error { profile.IntegerFields[0] = "caller-mutated"; return nil }
	result, err := quotafetch.Read(context.Background(), client, "project", "", profile, option)
	if err != nil || string(result.Resource.Body["volumes"]) != "12" || result.Resource.Body["caller-mutated"] != nil {
		t.Fatal(result, err)
	}
}
