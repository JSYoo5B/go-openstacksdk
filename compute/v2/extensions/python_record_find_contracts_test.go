package extensions

import (
	"context"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/testhelper"
)

func computeExtensionCloud(t *testing.T) (*testcloud.Cloud, *API) {
	t.Helper()
	cloud := testcloud.New(t)
	return cloud, New(cloud.Client("compute", "/v2.1/p/"))
}
func extensionField(t *testing.T, value *ExtensionRecord, key, want string) {
	t.Helper()
	if value == nil || value.Resource == nil || string(value.Resource.Body[key]) != want {
		t.Fatalf("%s want=%s value=%+v", key, want, value)
	}
}
func TestComputeExtensionFindMemberOwnsSourceMROWithoutDiscovery(t *testing.T) {
	cloud, api := computeExtensionCloud(t)
	calls := 0
	body := `{"extension":{"alias":"returned","name":"name","namespace":"uri","updated":"literal","links":[null,1,{"href":"x","extra":1}],"unknown":17,"self":"wire"}}`
	cloud.Mux.HandleFunc("/v2.1/p/extensions/request", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testhelper.AssertEquals(t, "", r.Header.Get("OpenStack-API-Version"))
		testhelper.AssertEquals(t, "", r.URL.RawQuery)
		w.Header().Set("X-Proof", "actual")
		testcloud.JSON(w, 203, body)
	})
	value, err := api.FindRecord(context.Background(), "request")
	if err != nil {
		t.Fatal(err)
	}
	extensionField(t, value, "id", `"request"`)
	extensionField(t, value, "alias", `"returned"`)
	extensionField(t, value, "namespace", `"uri"`)
	extensionField(t, value, "updated_at", `"literal"`)
	extensionField(t, value, "links", `[{},{},{"href":"x","extra":1}]`)
	if calls != 1 || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Proof") != "actual" || value.Resource.Body["unknown"] != nil || string(value.Wire.Body["unknown"]) != "17" {
		t.Fatalf("calls=%d value=%+v", calls, value)
	}
	value.Resource.Body["alias"][1] = 'X'
	extensionField(t, value, "id", `"request"`)
	if string(value.Wire.Body["alias"]) != `"returned"` {
		t.Fatal("raw body aliases projected bytes")
	}
}
func TestComputeExtensionFindCleanFallbackAliasNameMissingAndAmbiguity(t *testing.T) {
	for _, status := range []int{400, 403, 404} {
		for _, mode := range []string{"alias", "name", "missing", "duplicate"} {
			t.Run(http.StatusText(status)+mode, func(t *testing.T) {
				cloud, api := computeExtensionCloud(t)
				calls := 0
				cloud.Mux.HandleFunc("/v2.1/p/extensions/target", func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, status, `{"error":"member"}`) })
				cloud.Mux.HandleFunc("/v2.1/p/extensions", func(w http.ResponseWriter, r *http.Request) {
					calls++
					testhelper.AssertEquals(t, "", r.URL.RawQuery)
					testhelper.AssertEquals(t, "application/json", r.Header.Get("Accept"))
					rows := `[]`
					switch mode {
					case "alias":
						rows = `[{"alias":"target","name":"other"}]`
					case "name":
						rows = `[{"alias":"different","name":"target"}]`
					case "duplicate":
						rows = `[{"alias":"target"},{"name":"target"}]`
					}
					testcloud.JSON(w, 200, `{"extensions":`+rows+`}`)
				})
				value, err := api.FindRecord(context.Background(), "target")
				if mode == "duplicate" {
					if !errors.Is(err, resource.ErrAmbiguous) {
						t.Fatal(err)
					}
				} else if mode == "missing" {
					if err != nil || value != nil {
						t.Fatal(value, err)
					}
				} else if err != nil || value == nil {
					t.Fatal(value, err)
				}
				if calls != 2 {
					t.Fatalf("calls=%d", calls)
				}
			})
		}
	}
}
func TestComputeExtensionFindStrictMissingKeepsExistingNativeAPIs(t *testing.T) {
	cloud, api := computeExtensionCloud(t)
	cloud.Mux.HandleFunc("/v2.1/p/extensions/target", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 404, `{"error":"missing"}`) })
	cloud.Mux.HandleFunc("/v2.1/p/extensions", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 200, `{"extensions":[]}`) })
	if value, err := api.FindRecord(context.Background(), "target", WithRecordFindIgnoreMissing(false)); value != nil || !errors.Is(err, resource.ErrNotFound) {
		t.Fatal(value, err)
	}
	if _, err := api.Get(context.Background(), "target"); err == nil {
		t.Fatal("native Get changed missing behavior")
	}
}
func TestComputeExtensionFindOwnedKwargsAndGetOnlyVersion(t *testing.T) {
	cloud, api := computeExtensionCloud(t)
	api.client.Microversion = "2.8"
	calls := 0
	cloud.Mux.HandleFunc("/v2.1/p/extensions/target", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testhelper.AssertEquals(t, "compute 2.9", r.Header.Get("OpenStack-API-Version"))
		testhelper.AssertEquals(t, "2.9", r.Header.Get("X-OpenStack-Nova-API-Version"))
		testhelper.AssertEquals(t, "opaque", r.URL.Query().Get("vendor"))
		testcloud.JSON(w, 404, `{"error":"member"}`)
	})
	cloud.Mux.HandleFunc("/v2.1/p/alternate-extensions", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testhelper.AssertEquals(t, "compute 2.8", r.Header.Get("OpenStack-API-Version"))
		testhelper.AssertEquals(t, "", r.URL.RawQuery)
		testcloud.JSON(w, 200, `{"extensions":[{"alias":"target","namespace":"same"}]}`)
	})
	value, err := api.FindRecord(context.Background(), "target", WithRecordFindMicroversion("2.9"), WithRecordFindListBasePath("alternate-extensions"), WithRecordFindQuery("vendor", "opaque"), WithRecordFindQuery("namespace", "same"))
	if err != nil || value == nil || calls != 2 || api.client.Microversion != "2.8" {
		t.Fatal(value, err, calls)
	}
}
func TestComputeExtensionFindLaterPageAndSourcePrimaryPresence(t *testing.T) {
	cloud, api := computeExtensionCloud(t)
	calls := 0
	cloud.Mux.HandleFunc("/v2.1/p/extensions/target", func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, 404, `{}`) })
	cloud.Mux.HandleFunc("/v2.1/p/extensions", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("marker") == "" {
			testcloud.JSON(w, 200, `{"extensions":[{"id":null,"alias":"target","name":"no"}],"extensions_links":[{"rel":"next","href":"`+cloud.Server.URL+`/v2.1/p/extensions?marker=one"}]}`)
		} else {
			testcloud.JSON(w, 200, `{"extensions":[{"alias":"target"}]}`)
		}
	})
	value, err := api.FindRecord(context.Background(), "target")
	if err != nil || value == nil || calls != 3 {
		t.Fatal(value, err, calls)
	}
	extensionField(t, value, "id", `"target"`)
}
func TestComputeExtensionFindOptionSourceGuardAndCancellation(t *testing.T) {
	cloud, api := computeExtensionCloud(t)
	calls, late := 0, 0
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(500) })
	_, err := api.FindRecord(context.Background(), "target", func(c *RecordFindOpts) error { api.client.ResourceBase = cloud.Server.URL + "/changed/"; return nil }, func(c *RecordFindOpts) error { late++; return nil })
	if !errors.Is(err, resource.ErrInvalidOption) || calls != 0 || late != 0 {
		t.Fatal(err, calls, late)
	}
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("cancel witness")
	cancel(cause)
	if _, err := api.FindRecord(ctx, "target"); !errors.Is(err, cause) || calls != 0 {
		t.Fatal(err, calls)
	}
}

type extensionFaultTransport func(*http.Request) (*http.Response, error)

func (f extensionFaultTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type extensionFaultBody struct {
	io.ReadCloser
	err    error
	closes *int
}

func (b *extensionFaultBody) Close() error {
	*b.closes++
	return errors.Join(b.ReadCloser.Close(), b.err)
}
func TestComputeExtensionFindPhysicalRejectionsNeverFallbackOrRunHooks(t *testing.T) {
	for _, status := range []int{400, 403, 404, 401, 429, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			cloud, api := computeExtensionCloud(t)
			calls, closes, hooks := 0, 0, 0
			body := `{"error":"physical"}`
			cloud.Mux.HandleFunc("/v2.1/p/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("X-Proof", "physical")
				testcloud.JSON(w, status, body)
			})
			cause := errors.New("Close fault")
			base := cloud.Provider.HTTPClient.Transport
			cloud.Provider.HTTPClient.Transport = extensionFaultTransport(func(r *http.Request) (*http.Response, error) {
				resp, err := base.RoundTrip(r)
				if resp != nil {
					resp.Body = &extensionFaultBody{ReadCloser: resp.Body, err: cause, closes: &closes}
				}
				return resp, err
			})
			cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
				hooks++
				return nil
			}
			cloud.Provider.ReauthFunc = func(context.Context) error { hooks++; return nil }
			cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error { hooks++; return nil }
			cloud.Provider.MaxBackoffRetries = 1
			_, err := api.FindRecord(context.Background(), "target")
			if err == nil || calls != 1 || closes != 1 || hooks != 0 {
				t.Fatal(err, calls, closes, hooks)
			}
			var native gophercloud.ErrUnexpectedResponseCode
			if status == 401 {
				var renewal *gophercloud.ErrUnableToReauthenticate
				if !errors.As(err, &renewal) || !errors.As(renewal.ErrOriginal, &native) || !errors.Is(renewal.ErrReauth, cause) {
					t.Fatal(err)
				}
			} else if !errors.As(err, &native) || !errors.Is(err, cause) {
				t.Fatal(err)
			}
			if native.Actual != status || string(native.Body) != body || native.URL != cloud.Server.URL+"/v2.1/p/extensions/target" || native.Method != http.MethodGet || native.ResponseHeader.Get("X-Proof") != "physical" {
				t.Fatal(native)
			}
		})
	}
}
func TestComputeExtensionFindCleanAuthenticationRenewalKeepsMember(t *testing.T) {
	cloud, api := computeExtensionCloud(t)
	calls, renew := 0, 0
	cloud.Mux.HandleFunc("/v2.1/p/extensions/target", func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			testcloud.JSON(w, 401, `{}`)
			return
		}
		testhelper.AssertEquals(t, "renewed", r.Header.Get("X-Auth-Token"))
		testcloud.JSON(w, 200, `{"extension":{"alias":"target"}}`)
	})
	cloud.Provider.ReauthFunc = func(context.Context) error { renew++; cloud.Provider.SetToken("renewed"); return nil }
	if value, err := api.FindRecord(context.Background(), "target"); err != nil || value == nil || calls != 2 || renew != 1 {
		t.Fatal(value, err, calls, renew)
	}
}
