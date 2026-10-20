package shareattachments

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/JSYoo5B/go-openstacksdk/internal/testcloud"
	"github.com/JSYoo5B/go-openstacksdk/request"
	"github.com/JSYoo5B/go-openstacksdk/resource"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/testhelper"
)

func selectedCloud(t *testing.T) (*testcloud.Cloud, *gophercloud.ServiceClient, *API) {
	t.Helper()
	cloud := testcloud.New(t)
	client := cloud.Client("compute", "/v2.1/projects/p/")
	client.Microversion = "2.97"
	return cloud, client, New(client)
}
func assertVersion(t *testing.T, r *http.Request, version string) {
	t.Helper()
	testhelper.AssertEquals(t, "compute "+version, r.Header.Get("OpenStack-API-Version"))
	testhelper.AssertEquals(t, version, r.Header.Get("X-OpenStack-Nova-API-Version"))
	testhelper.AssertEquals(t, "test-token", r.Header.Get("X-Auth-Token"))
}
func rawEquals(t *testing.T, fields map[string]json.RawMessage, key, want string) {
	t.Helper()
	if string(fields[key]) != want {
		t.Fatalf("%s=%s; want%s", key, fields[key], want)
	}
}
func TestShareCreateDirtyBodyAndSeededDescriptors(t *testing.T) {
	cloud, client, api := selectedCloud(t)
	calls := 0
	cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testhelper.AssertEquals(t, "POST", r.Method)
		assertVersion(t, r, "2.97")
		testhelper.AssertEquals(t, "", r.URL.RawQuery)
		testhelper.AssertEquals(t, "application/json", r.Header.Get("Content-Type"))
		testhelper.TestJSONRequest(t, r, `{"share":{"share_id":"share","tag":null,"name":"seed"}}`)
		w.Header().Set("X-Proof", "accepted")
		testcloud.JSON(w, 201, `{"share":{"uuid":"mapping","share_id":"returned","status":"inactive","extension":123456789012345678901234567890,"self":"secret"}}`)
	})
	tag := json.RawMessage("null")
	options := CreateOpts{Attributes: map[string]json.RawMessage{"tag": tag, "name": json.RawMessage(`"seed"`), "ignored": json.RawMessage("broken")}}
	value, err := api.CreateRecord(context.Background(), "server", "share", WithCreateOptions(options))
	if err != nil {
		t.Fatal(err)
	}
	if value == nil || value.Resource == nil || value.Wire == nil || value.StatusCode != 201 || calls != 1 {
		t.Fatalf("value=%+v calls=%d", value, calls)
	}
	rawEquals(t, value.Resource.Body, "id", `"returned"`)
	rawEquals(t, value.Resource.Body, "uuid", `"mapping"`)
	rawEquals(t, value.Resource.Body, "tag", `null`)
	rawEquals(t, value.Resource.Body, "name", `"seed"`)
	rawEquals(t, value.Resource.Body, "export_location", `""`)
	rawEquals(t, value.Resource.Body, "location", `null`)
	if _, ok := value.Resource.Body["extension"]; ok {
		t.Fatal("unknown extension leaked into Source view")
	}
	rawEquals(t, value.Wire.Body, "extension", `123456789012345678901234567890`)
	if client.Microversion != "2.97" || len(value.Discovery) != 0 || value.ServerID != "server" {
		t.Fatalf("source/result changed %+v", value)
	}
	value.Resource.Body["share_id"][1] = 'X'
	value.Header.Set("X-Proof", "modified")
	rawEquals(t, value.Wire.Body, "share_id", `"returned"`)
	if value.Wire.Header.Get("X-Proof") != "accepted" {
		t.Fatal("headers aliased")
	}
}
func TestShareGetFixesParentMemberAndProxySeed(t *testing.T) {
	cloud, _, api := selectedCloud(t)
	calls := 0
	cloud.Mux.HandleFunc("/v2.1/projects/p/", func(w http.ResponseWriter, r *http.Request) {
		calls++
		testhelper.AssertEquals(t, "GET", r.Method)
		testhelper.AssertEquals(t, "/v2.1/projects/p/servers/server%2Fpart/shares/share%3Fother%23x", r.URL.EscapedPath())
		assertVersion(t, r, "2.97")
		testhelper.AssertEquals(t, "", r.URL.RawQuery)
		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			if len(body) != 0 {
				t.Fatalf("GET body=%s", body)
			}
		}
		testcloud.JSON(w, 203, `{"share":{"share_id":null,"export_location":null,"status":17,"uuid":"mapping","server_id":"foreign"}}`)
	})
	value, err := api.GetRecord(context.Background(), "server/part", "share?other#x")
	if err != nil {
		t.Fatal(err)
	}
	rawEquals(t, value.Resource.Body, "id", `"share?other#x"`)
	rawEquals(t, value.Resource.Body, "share_id", `null`)
	rawEquals(t, value.Resource.Body, "export_location", `null`)
	rawEquals(t, value.Resource.Body, "status", `17`)
	if value.ServerID != "server/part" || calls != 1 || value.StatusCode != 203 {
		t.Fatalf("value=%+v calls=%d", value, calls)
	}
}
func TestShareMemberNonJSONRetainsSeedWithoutFakeWire(t *testing.T) {
	for _, method := range []string{"GET", "POST"} {
		for _, body := range []string{"", "not-json", "{broken", string([]byte{0xff})} {
			t.Run(method+body, func(t *testing.T) {
				cloud, _, api := selectedCloud(t)
				cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares", func(w http.ResponseWriter, r *http.Request) { testcloud.JSON(w, 204, body) })
				cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares/share", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(202); _, _ = w.Write([]byte(body)) })
				var value *Record
				var err error
				if method == "GET" {
					value, err = api.GetRecord(context.Background(), "server", "share")
				} else {
					value, err = api.CreateRecord(context.Background(), "server", "share")
				}
				if err != nil {
					t.Fatal(err)
				}
				if value.Wire != nil || value.Resource == nil {
					t.Fatalf("value=%+v", value)
				}
				rawEquals(t, value.Resource.Body, "id", `"share"`)
			})
		}
	}
}
func TestShareMemberParsedInvalidShapeKeepsActualReceipt(t *testing.T) {
	for _, body := range []string{`null`, `[]`, `{"share":null}`, `{"share":[]}`, `{"share":5}`} {
		t.Run(body, func(t *testing.T) {
			cloud, _, api := selectedCloud(t)
			cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares/share", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Proof", "shape")
				testcloud.JSON(w, 203, body)
			})
			value, err := api.GetRecord(context.Background(), "server", "share")
			var physical *resource.ResponseError
			if err == nil || !errors.As(err, &physical) || value.Resource != nil || value.Wire != nil || value.StatusCode != 203 || string(value.Envelope) != body || value.Header.Get("X-Proof") != "shape" {
				t.Fatalf("value=%+v err=%v", value, err)
			}
		})
	}
}
func TestShareVersionDiscoveryCeilingAndExplicitSelection(t *testing.T) {
	for _, operation := range []string{"Create", "Get", "List", "Delete"} {
		for _, selected := range []string{"", "2.96", "2.110"} {
			t.Run(operation+"/"+selected, func(t *testing.T) {
				cloud, client, api := selectedCloud(t)
				// A real Nova catalog project URL lets shared discovery strip exactly
				// one project segment, reaching /v2.1/ instead of a resource URL.
				client.Endpoint = cloud.Server.URL + "/v2.1/p/"
				client.Microversion = selected
				discovery, member := 0, 0
				cloud.Mux.HandleFunc("/v2.1/", func(w http.ResponseWriter, r *http.Request) {
					discovery++
					testhelper.AssertEquals(t, "/v2.1/", r.URL.EscapedPath())
					testhelper.AssertEquals(t, http.MethodGet, r.Method)
					for name := range r.Header {
						if isVersionHeader(name) {
							t.Fatalf("discovery version header leaked %s", name)
						}
					}
					testcloud.JSON(w, 200, `{"version":{"id":"v2.1","status":"CURRENT","min_version":"2.1","version":"2.110","links":[{"rel":"self","href":"`+cloud.Server.URL+`/v2.1/"}]}}`)
				})
				memberHandler := func(w http.ResponseWriter, r *http.Request) {
					member++
					want := selected
					if want == "" {
						want = "2.97"
					}
					assertVersion(t, r, want)
					wantMethod, wantPath := http.MethodGet, "/v2.1/p/servers/server/shares/share"
					if operation == "Create" {
						wantMethod, wantPath = http.MethodPost, "/v2.1/p/servers/server/shares"
					}
					if operation == "List" {
						wantPath = "/v2.1/p/servers/server/shares"
					}
					if operation == "Delete" {
						wantMethod = http.MethodDelete
					}
					testhelper.AssertEquals(t, wantMethod, r.Method)
					testhelper.AssertEquals(t, wantPath, r.URL.EscapedPath())
					testcloud.JSON(w, 200, `{"share":{"share_id":"share"},"shares":[{"share_id":"share"}]}`)
				}
				cloud.Mux.HandleFunc("/v2.1/p/servers/server/shares", memberHandler)
				cloud.Mux.HandleFunc("/v2.1/p/servers/server/shares/share", memberHandler)
				var value *Record
				var deleted *DeleteResult
				var err error
				switch operation {
				case "Create":
					value, err = api.CreateRecord(context.Background(), "server", "share")
				case "Get":
					value, err = api.GetRecord(context.Background(), "server", "share")
				case "List":
					for current, e := range api.ListRecords(context.Background(), "server") {
						value, err = current, e
					}
				case "Delete":
					deleted, err = api.DeleteRecord(context.Background(), "server", "share")
				}
				if err != nil {
					t.Fatal(err)
				}
				want := 0
				if selected == "" {
					want = 1
				}
				proofCount := 0
				if value != nil {
					proofCount = len(value.Discovery)
				}
				if deleted != nil {
					proofCount = len(deleted.Discovery)
				}
				if discovery != want || member != 1 || client.Microversion != selected || proofCount != want {
					t.Fatalf("discovery=%d member=%d value=%+v deleted=%+v", discovery, member, value, deleted)
				}
			})
		}
	}
}
func TestShareExplicitEmptyOverrideOwnsVersionAbsence(t *testing.T) {
	cloud, client, api := selectedCloud(t)
	client.MoreHeaders = map[string]string{"OpenStack-API-Version": "compute 2.97", "X-OpenStack-Nova-API-Version": "2.97"}
	cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares/share", func(w http.ResponseWriter, r *http.Request) {
		for name := range r.Header {
			if isVersionHeader(name) {
				t.Fatalf("version header leaked %s", name)
			}
		}
		testcloud.JSON(w, 200, `{"share":{}}`)
	})
	if _, err := api.GetRecord(context.Background(), "server", "share", WithReadMicroversion("")); err != nil {
		t.Fatal(err)
	}
	if client.Microversion != "2.97" {
		t.Fatal("shared version changed")
	}
}
func TestShareInvalidOptionsAndIdentityPrecedeDiscovery(t *testing.T) {
	cloud, client, api := selectedCloud(t)
	client.Microversion = ""
	calls, late := 0, 0
	cloud.Mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(500) })
	cases := []func() error{
		func() error { _, e := api.GetRecord(context.Background(), "..", "share"); return e },
		func() error {
			_, e := api.CreateRecord(context.Background(), "server", "share", WithCreateOptions(CreateOpts{Attributes: map[string]json.RawMessage{"tag": json.RawMessage("broken")}}))
			return e
		},
		func() error {
			_, e := api.GetRecord(context.Background(), "server", "share", request.WithQuery[ReadOpts]("q", "v"))
			return e
		},
		func() error {
			_, e := api.DeleteRecord(context.Background(), "server", "share", request.WithField[DeleteOpts]("x", 1))
			return e
		},
		func() error {
			for _, e := range api.ListRecords(context.Background(), "server", WithListLimit(-1)) {
				return e
			}
			return nil
		},
		func() error {
			_, e := api.GetRecord(context.Background(), "server", "share", func(c *request.Config[ReadOpts]) error { client.Endpoint = cloud.Server.URL + "/changed/"; return nil }, func(c *request.Config[ReadOpts]) error { late++; return nil })
			return e
		},
	}
	for i, invoke := range cases {
		before := client.Endpoint
		err := invoke()
		client.Endpoint = before
		if err == nil {
			t.Fatalf("case%d succeeded", i)
		}
	}
	if calls != 0 || late != 0 {
		t.Fatalf("calls=%d late=%d", calls, late)
	}
}
func TestShareDeleteDefaultOnlyCleanMissingAndBodylessSuccess(t *testing.T) {
	for _, status := range []int{200, 202, 204, 404, 409} {
		for _, strict := range []bool{false, true} {
			t.Run(http.StatusText(status)+string(rune('0'+status%10))+strconvBool(strict), func(t *testing.T) {
				cloud, _, api := selectedCloud(t)
				cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares/share", func(w http.ResponseWriter, r *http.Request) {
					testhelper.AssertEquals(t, "DELETE", r.Method)
					assertVersion(t, r, "2.97")
					if r.ContentLength != 0 || r.Body != nil && r.Body != http.NoBody {
						t.Fatal("DELETE had body")
					}
					w.Header().Set("X-Proof", "delete")
					testcloud.JSON(w, status, `{"receipt":"literal"}`)
				})
				var options []DeleteOption
				if strict {
					options = append(options, WithDeleteIgnoreMissing(false))
				}
				value, err := api.DeleteRecord(context.Background(), "server", "share", options...)
				wantErr := status == 409 || status == 404 && strict
				if (err != nil) != wantErr || value.StatusCode != status || value.IgnoredMissing != (status == 404 && !strict) {
					t.Fatalf("value=%+v err=%v", value, err)
				}
			})
		}
	}
}
func strconvBool(value bool) string {
	if value {
		return "strict"
	}
	return "default"
}
func TestShareListProjectionLocalCapAndLazyNoLookahead(t *testing.T) {
	cloud, _, api := selectedCloud(t)
	calls := 0
	cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares", func(w http.ResponseWriter, r *http.Request) {
		calls++
		assertVersion(t, r, "2.97")
		testhelper.AssertEquals(t, "2", r.URL.Query().Get("limit"))
		testhelper.AssertEquals(t, "application/json", r.Header.Get("Accept"))
		testcloud.JSON(w, 200, `{"shares":[{"share_id":"a","status":"inactive"},{"share_id":"b","status":"active","tag":null},{"share_id":"c","status":"active"}],"next":{"invalid":"unconsumed"}}`)
	})
	sequence := api.ListRecords(context.Background(), "server", WithListMaxItems(2), WithListFilter("status", "active"))
	if calls != 0 {
		t.Fatal("eager list")
	}
	var ids []string
	for value, err := range sequence {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, string(value.Resource.Body["id"]))
		if value.ServerID != "server" || value.StatusCode != 200 || !strings.Contains(string(value.Envelope), `"shares"`) {
			t.Fatalf("value=%+v", value)
		}
	}
	if !reflect.DeepEqual(ids, []string{`"b"`}) || calls != 1 {
		t.Fatalf("ids=%v calls=%d", ids, calls)
	}
}
func TestShareListAlternateMarkerPresenceAndContinuation(t *testing.T) {
	for _, primary := range []string{"", `,"id":"primary"`} {
		t.Run(primary, func(t *testing.T) {
			cloud, _, api := selectedCloud(t)
			calls := 0
			cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares", func(w http.ResponseWriter, r *http.Request) {
				calls++
				switch calls {
				case 1:
					testhelper.AssertEquals(t, "1", r.URL.Query().Get("limit"))
					testcloud.JSON(w, 200, `{"shares":[{"share_id":"alternate"`+primary+`} ]}`)
				case 2:
					want := "alternate"
					if primary != "" {
						want = "primary"
					}
					testhelper.AssertEquals(t, want, r.URL.Query().Get("marker"))
					testcloud.JSON(w, 200, `{"shares":[]}`)
				default:
					t.Fatal("extra page")
				}
			})
			count := 0
			for _, err := range api.ListRecords(context.Background(), "server", WithListLimit(1)) {
				if err != nil {
					t.Fatal(err)
				}
				count++
			}
			if count != 1 || calls != 2 {
				t.Fatalf("count=%d calls=%d", count, calls)
			}
		})
	}
}
func TestShareListNullPrimaryMarkerIsTerminalWithPageReceipt(t *testing.T) {
	cloud, _, api := selectedCloud(t)
	calls := 0
	body := `{"shares":[{"id":null,"share_id":"alternate"}]}`
	cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares", func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, 200, body) })
	count := 0
	var failure error
	for value, err := range api.ListRecords(context.Background(), "server", WithListLimit(1)) {
		if err != nil {
			failure = err
			break
		}
		rawEquals(t, value.Resource.Body, "id", `null`)
		count++
	}
	var proof *resource.ResponseError
	if count != 1 || calls != 1 || failure == nil || !errors.As(failure, &proof) || string(proof.Body) != body {
		t.Fatalf("count=%d calls=%d err=%v", count, calls, failure)
	}
}
func TestShareListUnknownDiscardAndControlRejectBeforeMarshaler(t *testing.T) {
	for _, key := range []string{"unknown", "jmespath_filters", "base_path", "server_id", "microversion"} {
		t.Run(key, func(t *testing.T) {
			cloud, _, api := selectedCloud(t)
			calls, encoded := 0, 0
			cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares", func(w http.ResponseWriter, r *http.Request) {
				calls++
				testhelper.AssertEquals(t, "", r.URL.RawQuery)
				testcloud.JSON(w, 200, `{"shares":[]}`)
			})
			var failure error
			for _, err := range api.ListRecords(context.Background(), "server", WithListFilter(key, countMarshal{calls: &encoded})) {
				failure = err
			}
			if encoded != 0 {
				t.Fatalf("marshaled ignored/control value %d", encoded)
			}
			if key == "unknown" {
				if failure != nil || calls != 1 {
					t.Fatalf("calls=%d err=%v", calls, failure)
				}
			} else if !errors.Is(failure, resource.ErrUnsupported) || calls != 0 {
				t.Fatalf("calls=%d err=%v", calls, failure)
			}
		})
	}
}

type countMarshal struct{ calls *int }

func (v countMarshal) MarshalJSON() ([]byte, error) {
	*v.calls++
	return nil, errors.New("must not marshal")
}

type faultTransport func(*http.Request) (*http.Response, error)

func (f faultTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type faultBody struct {
	io.ReadCloser
	readErr, closeErr error
	closeCount        *int
}

func (b *faultBody) Read(data []byte) (int, error) {
	n, err := b.ReadCloser.Read(data)
	if err == io.EOF && b.readErr != nil {
		return n, b.readErr
	}
	return n, err
}
func (b *faultBody) Close() error {
	*b.closeCount++
	return errors.Join(b.ReadCloser.Close(), b.closeErr)
}
func TestSharePhysicalFaultsKeepEvidenceAndStopAllHooks(t *testing.T) {
	for _, operation := range []string{"Create", "Get", "List", "Delete"} {
		for _, status := range []int{200, 203, 401, 404, 409, 429, 500, 503} {
			for _, mode := range []string{"read", "close", "both"} {
				t.Run(operation+http.StatusText(status)+mode, func(t *testing.T) {
					cloud, _, api := selectedCloud(t)
					calls, closes, retry, reauth, backoff := 0, 0, 0, 0, 0
					body := `{"share":{"share_id":"share"},"shares":[]}`
					cloud.Mux.HandleFunc("/v2.1/projects/p/", func(w http.ResponseWriter, r *http.Request) {
						calls++
						w.Header().Set("X-Proof", "physical")
						testcloud.JSON(w, status, body)
					})
					readErr, closeErr := errors.New("read sentinel"), errors.New("close sentinel")
					var read, close error
					if mode != "close" {
						read = readErr
					}
					if mode != "read" {
						close = closeErr
					}
					base := cloud.Provider.HTTPClient.Transport
					cloud.Provider.HTTPClient.Transport = faultTransport(func(r *http.Request) (*http.Response, error) {
						response, err := base.RoundTrip(r)
						if response != nil {
							response.Body = &faultBody{ReadCloser: response.Body, readErr: read, closeErr: close, closeCount: &closes}
						}
						return response, err
					})
					cloud.Provider.RetryFunc = func(context.Context, string, string, *gophercloud.RequestOpts, error, uint) error {
						retry++
						return nil
					}
					cloud.Provider.ReauthFunc = func(context.Context) error { reauth++; return nil }
					cloud.Provider.RetryBackoffFunc = func(context.Context, *gophercloud.ErrUnexpectedResponseCode, error, uint) error {
						backoff++
						return nil
					}
					cloud.Provider.MaxBackoffRetries = 1
					var value *Record
					var deleted *DeleteResult
					var err error
					switch operation {
					case "Create":
						value, err = api.CreateRecord(context.Background(), "server", "share")
					case "Get":
						value, err = api.GetRecord(context.Background(), "server", "share")
					case "List":
						for _, failure := range api.ListRecords(context.Background(), "server") {
							err = failure
						}
					case "Delete":
						deleted, err = api.DeleteRecord(context.Background(), "server", "share")
					}
					if err == nil || calls != 1 || closes != 1 || retry != 0 || reauth != 0 || backoff != 0 {
						t.Fatalf("err=%v calls=%d close=%d retry=%d reauth=%d backoff=%d", err, calls, closes, retry, reauth, backoff)
					}
					wantMethod, wantPath := http.MethodGet, "/v2.1/projects/p/servers/server/shares/share"
					if operation == "Create" {
						wantMethod, wantPath = http.MethodPost, "/v2.1/projects/p/servers/server/shares"
					}
					if operation == "List" {
						wantPath = "/v2.1/projects/p/servers/server/shares"
					}
					if operation == "Delete" {
						wantMethod = http.MethodDelete
					}
					if value != nil && (value.Resource != nil || value.Wire != nil || value.StatusCode != status || string(value.Envelope) != body || value.Header.Get("X-Proof") != "physical") {
						t.Fatalf("member physical failure fabricated resource or lost receipt: %+v", value)
					}
					if operation != "List" && operation != "Delete" && value == nil {
						t.Fatal("member failure lost actual receipt")
					}
					if operation == "Delete" && (deleted == nil || deleted.IgnoredMissing || deleted.StatusCode != status || string(deleted.Envelope) != body || deleted.Header.Get("X-Proof") != "physical") {
						t.Fatalf("delete physical failure became missing or lost receipt: %+v", deleted)
					}
					if status == 401 {
						var renewed *gophercloud.ErrUnableToReauthenticate
						if !errors.As(err, &renewed) {
							t.Fatalf("native 401 wrapper lost: %T %v", err, err)
						}
						var native gophercloud.ErrUnexpectedResponseCode
						if !errors.As(renewed.ErrOriginal, &native) || native.Method != wantMethod || native.URL != cloud.Server.URL+wantPath || native.Actual != status || string(native.Body) != body || native.ResponseHeader.Get("X-Proof") != "physical" || !reflect.DeepEqual(native.Expected, codes(200)) {
							t.Fatalf("native 401 ErrOriginal public evidence changed: %+v wrapper=%+v", native, renewed)
						}
						if read != nil && !errors.Is(renewed.ErrReauth, readErr) || close != nil && !errors.Is(renewed.ErrReauth, closeErr) {
							t.Fatalf("native401 causes lost %+v", renewed)
						}
					} else {
						if read != nil && !errors.Is(err, readErr) || close != nil && !errors.Is(err, closeErr) {
							t.Fatalf("fault causes lost %v", err)
						}
					}
					proof := nativeResponse(err)
					if proof == nil || proof.StatusCode != status || string(proof.Body) != body || proof.Header.Get("X-Proof") != "physical" {
						t.Fatalf("proof=%+v err=%v", proof, err)
					}
				})
			}
		}
	}
}
func TestShareCleanAuthenticationRenewalRetainsRequest(t *testing.T) {
	for _, operation := range []string{"Create", "Get", "List", "Delete"} {
		t.Run(operation, func(t *testing.T) {
			cloud, _, api := selectedCloud(t)
			calls, renewals := 0, 0
			var firstMethod, firstPath, firstBody string
			cloud.Mux.HandleFunc("/v2.1/projects/p/", func(w http.ResponseWriter, r *http.Request) {
				calls++
				body, _ := io.ReadAll(r.Body)
				testhelper.AssertEquals(t, "compute 2.97", r.Header.Get("OpenStack-API-Version"))
				if calls == 1 {
					firstMethod, firstPath, firstBody = r.Method, r.URL.EscapedPath(), string(body)
					testcloud.JSON(w, 401, `{"error":"renew"}`)
					return
				}
				if r.Method != firstMethod || r.URL.EscapedPath() != firstPath || string(body) != firstBody {
					t.Fatal("renewal changed request")
				}
				testhelper.AssertEquals(t, "renewed", r.Header.Get("X-Auth-Token"))
				testcloud.JSON(w, 200, `{"share":{"share_id":"share"},"shares":[]}`)
			})
			cloud.Provider.ReauthFunc = func(context.Context) error { renewals++; cloud.Provider.SetToken("renewed"); return nil }
			var err error
			switch operation {
			case "Create":
				_, err = api.CreateRecord(context.Background(), "server", "share", WithCreateAttribute("tag", "tag"))
			case "Get":
				_, err = api.GetRecord(context.Background(), "server", "share")
			case "List":
				for _, e := range api.ListRecords(context.Background(), "server") {
					err = e
				}
			case "Delete":
				_, err = api.DeleteRecord(context.Background(), "server", "share")
			}
			if err != nil || calls != 2 || renewals != 1 {
				t.Fatalf("err=%v calls=%d renewals=%d", err, calls, renewals)
			}
		})
	}
}

func TestShareOwnedConfigRepairsNilMapsAndCopiesPolicyAroundCallbacks(t *testing.T) {
	cloud, client, api := selectedCloud(t)
	selected := "2.98"
	headers := map[string]string{"X-Caller": "initial"}
	calls := 0
	cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares/share", func(w http.ResponseWriter, r *http.Request) {
		calls++
		assertVersion(t, r, "2.98")
		testhelper.AssertEquals(t, "last", r.Header.Get("X-Caller"))
		testcloud.JSON(w, 200, `{"share":{}}`)
	})
	_, err := api.GetRecord(context.Background(), "server", "share", func(c *request.Config[ReadOpts]) error {
		c.Options.Microversion = &selected
		c.Headers = headers
		c.Query = nil
		c.Fields = nil
		c.Arguments = nil
		return nil
	}, func(c *request.Config[ReadOpts]) error {
		selected = "2.99"
		headers["X-Caller"] = "outside"
		if *c.Options.Microversion != "2.98" || c.Headers["X-Caller"] != "initial" || c.Query == nil || c.Fields == nil || c.Arguments == nil {
			return errors.New("option carriers leaked or maps stayed nil")
		}
		c.Headers = nil
		return nil
	}, WithReadHeader("X-Caller", "last"))
	if err != nil || calls != 1 || client.Microversion != "2.97" {
		t.Fatalf("err=%v calls=%d selected=%s", err, calls, client.Microversion)
	}
}

func TestShareListActualLinksUseFixedRouteAndSinglePageStopsBeforeLink(t *testing.T) {
	for _, single := range []bool{false, true} {
		t.Run(strconvBool(single), func(t *testing.T) {
			cloud, _, api := selectedCloud(t)
			calls := 0
			cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares", func(w http.ResponseWriter, r *http.Request) {
				calls++
				assertVersion(t, r, "2.97")
				if calls == 1 {
					testcloud.JSON(w, 200, `{"shares":{"share_id":"one"},"shares_links":[{"rel":"next","href":"`+cloud.Server.URL+`/v2.1/projects/p/servers/server/shares?marker=one"}]}`)
				} else {
					testhelper.AssertEquals(t, "one", r.URL.Query().Get("marker"))
					testcloud.JSON(w, 200, `{"shares":[{"share_id":"two"}],"next":1e-400}`)
				}
			})
			var options []ListOption
			if single {
				options = append(options, WithListPaginated(false))
			}
			count := 0
			for _, err := range api.ListRecords(context.Background(), "server", options...) {
				if err != nil {
					t.Fatal(err)
				}
				count++
			}
			want := 2
			if single {
				want = 1
			}
			if count != want || calls != want {
				t.Fatalf("count=%d calls=%d", count, calls)
			}
		})
	}
}

func TestShareListForeignContinuationFailsWithActualPage(t *testing.T) {
	cloud, _, api := selectedCloud(t)
	calls := 0
	body := `{"shares":[{"share_id":"one"}],"next":"http://other.invalid/servers/server/shares?marker=one"}`
	cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares", func(w http.ResponseWriter, r *http.Request) { calls++; testcloud.JSON(w, 200, body) })
	count := 0
	var failure error
	for _, err := range api.ListRecords(context.Background(), "server") {
		if err != nil {
			failure = err
			break
		}
		count++
	}
	var proof *resource.ResponseError
	if count != 1 || calls != 1 || failure == nil || !errors.As(failure, &proof) || string(proof.Body) != body || proof.StatusCode != 200 {
		t.Fatalf("count=%d calls=%d err=%v", count, calls, failure)
	}
}

func TestShareSourceLoadedLocalEqualityAndDictionaryTruthiness(t *testing.T) {
	cases := []struct {
		name   string
		filter json.RawMessage
		rows   string
		want   []string
	}{
		{"boolean-is-integer", json.RawMessage(`true`), `{"share_id":"integer","tag":1},{"share_id":"boolean","tag":true},{"share_id":"zero","tag":0}`, []string{`"integer"`, `"boolean"`}},
		{"float-underflow", json.RawMessage(`0`), `{"share_id":"underflow","tag":1e-400},{"share_id":"false","tag":false},{"share_id":"integer","tag":0},{"share_id":"different","tag":1}`, []string{`"underflow"`, `"false"`, `"integer"`}},
		{"float-rounding-and-exact-integer", json.RawMessage(`9007199254740993.0`), `{"share_id":"float","tag":9007199254740993.0},{"share_id":"adjacent","tag":9007199254740992},{"share_id":"exact","tag":9007199254740993}`, []string{`"float"`, `"adjacent"`}},
		{"nested-dictionary-subset", json.RawMessage(`{"child":{"value":1}}`), `{"share_id":"one","tag":{"child":{"value":true,"extra":2},"other":3}},{"share_id":"empty","tag":{"child":{}}},{"share_id":"different","tag":{"child":{"value":2}}}`, []string{`"one"`}},
		{"empty-wanted-uses-actual-truthiness", json.RawMessage(`{}`), `{"share_id":"object","tag":{"a":1}},{"share_id":"text","tag":"truthy"},{"share_id":"empty","tag":{}},{"share_id":"null","tag":null},{"share_id":"underflow","tag":1e-400}`, []string{`"object"`, `"text"`}},
		{"missing-nested-member-is-none", json.RawMessage(`{"missing":null}`), `{"share_id":"match","tag":{"other":1}},{"share_id":"empty","tag":{}}`, []string{`"match"`}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			cloud, _, api := selectedCloud(t)
			cloud.Mux.HandleFunc("/v2.1/projects/p/servers/server/shares", func(w http.ResponseWriter, r *http.Request) {
				testhelper.AssertEquals(t, "", r.URL.RawQuery)
				testcloud.JSON(w, 200, `{"shares":[`+test.rows+`]}`)
			})
			var ids []string
			for value, err := range api.ListRecords(context.Background(), "server", WithListFilter("tag", test.filter)) {
				if err != nil {
					t.Fatal(err)
				}
				ids = append(ids, string(value.Resource.Body["id"]))
			}
			if !reflect.DeepEqual(ids, test.want) {
				t.Fatalf("ids=%v want=%v", ids, test.want)
			}
		})
	}
}
